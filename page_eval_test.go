package rod_test

import (
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func (t T) PageEvalOnNewDocument() {
	p := t.newPage("")

	p.MustEvalOnNewDocument(`window.rod = 'ok'`)

	// to activate the script
	p.MustNavigate(t.blank())

	t.Eq(p.MustEval("rod").String(), "ok")

	t.Panic(func() {
		t.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		p.MustEvalOnNewDocument(`1`)
	})
}

func (t T) PageEval() {
	page := t.page.MustNavigate(t.blank())

	t.Eq(3, page.MustEval(`
		(a, b) => a + b
	`, 1, 2).Int())
	t.Eq(10, page.MustEval(`
		10
	`).Int())
	t.Eq(1, page.MustEval(`a => 1`).Int())
	t.Eq(1, page.MustEval(`function() { return 1 }`).Int())
	t.Eq(1, page.MustEval(`((1))`).Int())
	t.Neq(1, page.MustEval(`a = () => 1`).Int())
	t.Neq(1, page.MustEval(`a = function() { return 1 }`))
	t.Neq(1, page.MustEval(`/* ) */`))

	// reuse obj
	obj := page.MustEvaluate(rod.Eval(`() => () => 'ok'`).ByObject())
	t.Eq("ok", page.MustEval(`f => f()`, obj).Str())
}

func (t T) PageEvaluateRetry() {
	page := t.page.MustNavigate(t.blank())

	t.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
		t.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(nil), cdp.ErrCtxNotFound
		})
		return gson.New(nil), cdp.ErrCtxNotFound
	})
	t.Eq(1, page.MustEval(`1`).Int())
}

func (t T) PageUpdateJSCtxIDErr() {
	page := t.page.MustNavigate(t.srcFile("./fixtures/click-iframe.html"))

	t.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
		t.mc.stubErr(1, proto.RuntimeEvaluate{})
		return gson.New(nil), cdp.ErrCtxNotFound
	})
	t.Err(page.Eval(`1`))

	el := page.MustElement("iframe")

	t.mc.stubErr(1, proto.DOMGetFrameOwner{})
	t.Err(el.Frame())

	t.mc.stubErr(2, proto.DOMDescribeNode{})
	t.Err(el.Frame())

	t.mc.stubErr(1, proto.DOMResolveNode{})
	t.Err(el.Frame())
}

func (t T) PageExpose() {
	page := t.newPage(t.blank()).MustWaitLoad()

	stop := page.MustExpose("exposedFunc", func(g gson.JSON) (interface{}, error) {
		return g.Get("k").Str(), nil
	})

	utils.All(func() {
		res := page.MustEval(`exposedFunc({k: 'a'})`)
		t.Eq("a", res.Str())
	}, func() {
		res := page.MustEval(`exposedFunc({k: 'b'})`)
		t.Eq("b", res.Str())
	})()

	// survive the reload
	page.MustReload().MustWaitLoad()
	res := page.MustEval(`exposedFunc({k: 'ok'})`)
	t.Eq("ok", res.Str())

	stop()

	t.Panic(func() {
		stop()
	})
	t.Panic(func() {
		page.MustReload().MustWaitLoad().MustEval(`exposedFunc()`)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		page.MustExpose("exposedFunc", nil)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeAddBinding{})
		page.MustExpose("exposedFunc2", nil)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		page.MustExpose("exposedFunc", nil)
	})
}

func (t T) Release() {
	res, err := t.page.Evaluate(rod.Eval(`document`).ByObject())
	t.E(err)
	t.page.MustRelease(res)
}

func (t T) PromiseLeak() {
	/*
		Perform a slow action then navigate the page to another url,
		we can see the slow operation will still be executed.
	*/

	p := t.page.MustNavigate(t.blank())

	utils.All(func() {
		_, err := p.Eval(`new Promise(r => setTimeout(() => r(location.href), 300))`)
		t.Is(err, cdp.ErrCtxDestroyed)
	}, func() {
		utils.Sleep(0.1)
		p.MustNavigate(t.blank())
	})()
}

func (t T) ObjectLeak() {
	/*
		Seems like it won't leak
	*/

	p := t.page.MustNavigate(t.blank())

	obj := p.MustEvaluate(rod.Eval("{a:1}").ByObject())
	p.MustReload().MustWaitLoad()
	t.Panic(func() {
		p.MustEvaluate(rod.Eval(`obj => obj`, obj))
	})
}

func (t T) PageObjectErr() {
	t.Panic(func() {
		t.page.MustObjectToJSON(&proto.RuntimeRemoteObject{
			ObjectID: "not-exists",
		})
	})
	t.Panic(func() {
		t.page.MustElementFromNode(-1)
	})
	t.Panic(func() {
		id := t.page.MustNavigate(t.blank()).MustElement(`body`).MustNodeID()
		t.mc.stubErr(1, proto.DOMResolveNode{})
		t.page.MustElementFromNode(id)
	})
}

func (t T) GetJSHelperRetry() {
	t.page.MustNavigate(t.srcFile("fixtures/click.html"))

	t.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
		return gson.JSON{}, cdp.ErrCtxNotFound
	})
	t.page.MustElements("button")
}

func (t T) ConcurrentEval() {
	p := t.page.MustNavigate(t.blank())
	list := make(chan int, 2)

	start := time.Now()
	utils.All(func() {
		list <- p.MustEval(`new Promise(r => setTimeout(r, 1000, 2))`).Int()
	}, func() {
		list <- p.MustEval(`new Promise(r => setTimeout(r, 500, 1))`).Int()
	})()
	duration := time.Since(start)

	t.Lt(duration, 1500*time.Millisecond)
	t.Gt(duration, 1000*time.Millisecond)
	t.Eq([]int{<-list, <-list}, []int{1, 2})
}

func (t T) PageSlowRender() {
	p := t.page.MustNavigate(t.srcFile("./fixtures/slow-render.html"))
	t.Eq(p.MustElement("div").MustText(), "ok")
}

func (t T) PageIframeReload() {
	p := t.page.MustNavigate(t.srcFile("./fixtures/click-iframe.html"))
	frame := p.MustElement("iframe").MustFrame()
	btn := frame.MustElement("button")
	t.Eq(btn.MustText(), "click me")

	frame.MustReload()
	btn = frame.MustElement("button")
	t.Eq(btn.MustText(), "click me")

	t.Has(*p.MustElement("iframe").MustAttribute("src"), "click.html")
}

func (t T) PageObjCrossNavigation() {
	p := t.page.MustNavigate(t.blank())
	obj := p.MustEvaluate(rod.Eval(`{}`).ByObject())

	t.page.MustNavigate(t.blank())

	_, err := p.Evaluate(rod.Eval(`1`).This(obj))
	t.Is(err, &rod.ErrObjectNotFound{})
	t.Has(err.Error(), "cannot find object: {\"type\":\"object\"")
}

func (t T) EnsureJSHelperErr() {
	p := t.page.MustNavigate(t.blank())

	t.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
	t.Err(p.Evaluate(rod.EvalHelper(js.Overlay, "test", 0, 0, 10, 10, "msg")))
}

func (t T) EvalOptionsString() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	t.Eq(rod.Eval(`this.parentElement`).This(el.Object).String(), "this.parentElement() button")
}
