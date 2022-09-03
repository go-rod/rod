package rod_test

import (
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func TestPageEvalOnNewDocument(t *testing.T) {
	g := setup(t)

	p := g.newPage()

	p.MustEvalOnNewDocument(`window.rod = 'ok'`)

	// to activate the script
	p.MustNavigate(g.blank())

	g.Eq(p.MustEval("() => rod").String(), "ok")

	g.Panic(func() {
		g.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		p.MustEvalOnNewDocument(`1`)
	})
}

func TestPageEval(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.blank())

	g.Eq(3, page.MustEval(`
		(a, b) => a + b
	`, 1, 2).Int())

	g.Eq(10, page.MustEval(`(a, b, c, d) => a + b + c + d`, 1, 2, 3, 4).Int())

	g.Eq(page.MustEval(`function() {
		return 11
	}`).Int(), 11)

	g.Eq(page.MustEval(`	 ; () => 1; `).Int(), 1)

	// reuse obj
	obj := page.MustEvaluate(rod.Eval(`() => () => 'ok'`).ByObject())
	g.Eq("ok", page.MustEval(`f => f()`, obj).Str())

	_, err := page.Eval(`10`)
	g.Has(err.Error(), `eval js error: TypeError: 10.apply is not a function`)

	_, err = page.Eval(`() => notExist()`)
	g.Is(err, &rod.ErrEval{})
	g.Has(err.Error(), `eval js error: ReferenceError: notExist is not defined`)
}

func TestPageEvaluateRetry(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.blank())

	g.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
		g.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(nil), cdp.ErrCtxNotFound
		})
		return gson.New(nil), cdp.ErrCtxNotFound
	})
	g.Eq(1, page.MustEval(`() => 1`).Int())
}

func TestPageUpdateJSCtxIDErr(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("./fixtures/click-iframe.html"))

	g.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
		g.mc.stubErr(1, proto.RuntimeEvaluate{})
		return gson.New(nil), cdp.ErrCtxNotFound
	})
	g.Err(page.Eval(`() => 1`))

	frame := page.MustElement("iframe").MustFrame()

	frame.MustReload()
	g.mc.stubErr(1, proto.DOMDescribeNode{})
	g.Err(frame.Element(`button`))

	frame.MustReload()
	g.mc.stubErr(1, proto.DOMResolveNode{})
	g.Err(frame.Element(`button`))
}

func TestPageExpose(t *testing.T) {
	g := setup(t)

	page := g.newPage(g.blank()).MustWaitLoad()

	stop := page.MustExpose("exposedFunc", func(g gson.JSON) (interface{}, error) {
		return g.Get("k").Str(), nil
	})

	utils.All(func() {
		res := page.MustEval(`() => exposedFunc({k: 'a'})`)
		g.Eq("a", res.Str())
	}, func() {
		res := page.MustEval(`() => exposedFunc({k: 'b'})`)
		g.Eq("b", res.Str())
	})()

	// survive the reload
	page.MustReload().MustWaitLoad()
	res := page.MustEval(`() => exposedFunc({k: 'ok'})`)
	g.Eq("ok", res.Str())

	stop()

	g.Panic(func() {
		stop()
	})
	g.Panic(func() {
		page.MustReload().MustWaitLoad().MustEval(`() => exposedFunc()`)
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		page.MustExpose("exposedFunc", nil)
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeAddBinding{})
		page.MustExpose("exposedFunc2", nil)
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		page.MustExpose("exposedFunc", nil)
	})
}

func TestObjectRelease(t *testing.T) {
	g := setup(t)

	res, err := g.page.Evaluate(rod.Eval(`() => document`).ByObject())
	g.E(err)
	g.page.MustRelease(res)
}

func TestPromiseLeak(t *testing.T) {
	g := setup(t)

	/*
		Perform a slow action then navigate the page to another url,
		we can see the slow operation will still be executed.
	*/

	p := g.page.MustNavigate(g.blank())

	utils.All(func() {
		_, err := p.Eval(`() => new Promise(r => setTimeout(() => r(location.href), 1000))`)
		g.Is(err, cdp.ErrCtxDestroyed)
	}, func() {
		utils.Sleep(0.3)
		p.MustNavigate(g.blank())
	})()
}

func TestObjectLeak(t *testing.T) {
	g := setup(t)

	/*
		Seems like it won't leak
	*/

	p := g.page.MustNavigate(g.blank())

	obj := p.MustEvaluate(rod.Eval("() => ({a:1})").ByObject())
	p.MustReload().MustWaitLoad()
	g.Panic(func() {
		p.MustEvaluate(rod.Eval(`obj => obj`, obj))
	})
}

func TestPageObjectErr(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		g.page.MustObjectToJSON(&proto.RuntimeRemoteObject{
			ObjectID: "not-exists",
		})
	})
	g.Panic(func() {
		g.page.MustElementFromNode(&proto.DOMNode{NodeID: -1})
	})
	g.Panic(func() {
		node := g.page.MustNavigate(g.blank()).MustElement(`body`).MustDescribe()
		g.mc.stubErr(1, proto.DOMResolveNode{})
		g.page.MustElementFromNode(node)
	})
}

func TestGetJSHelperRetry(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate(g.srcFile("fixtures/click.html"))

	g.mc.stub(1, proto.RuntimeCallFunctionOn{}, func(send StubSend) (gson.JSON, error) {
		return gson.JSON{}, cdp.ErrCtxNotFound
	})
	g.page.MustElements("button")
}

func TestConcurrentEval(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.blank())
	list := make(chan int, 2)

	start := time.Now()
	utils.All(func() {
		list <- p.MustEval(`() => new Promise(r => setTimeout(r, 2000, 2))`).Int()
	}, func() {
		list <- p.MustEval(`() => new Promise(r => setTimeout(r, 1000, 1))`).Int()
	})()
	duration := time.Since(start)

	g.Gt(duration, 1000*time.Millisecond)
	g.Lt(duration, 3000*time.Millisecond)
	g.Eq([]int{<-list, <-list}, []int{1, 2})
}

func TestPageSlowRender(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("./fixtures/slow-render.html"))
	g.Eq(p.MustElement("div").MustText(), "ok")
}

func TestPageIframeReload(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("./fixtures/click-iframe.html"))
	frame := p.MustElement("iframe").MustFrame()
	btn := frame.MustElement("button")
	g.Eq(btn.MustText(), "click me")

	frame.MustReload()
	btn = frame.MustElement("button")
	g.Eq(btn.MustText(), "click me")

	g.Has(*p.MustElement("iframe").MustAttribute("src"), "click.html")
}

func TestPageObjCrossNavigation(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.blank())
	obj := p.MustEvaluate(rod.Eval(`() => ({})`).ByObject())

	g.page.MustNavigate(g.blank())

	_, err := p.Evaluate(rod.Eval(`() => 1`).This(obj))
	g.Is(err, &rod.ErrObjectNotFound{})
	g.Has(err.Error(), "cannot find object: {\"type\":\"object\"")
}

func TestEnsureJSHelperErr(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.blank())

	g.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
	g.Err(p.Elements(`button`))
}

func TestEvalOptionsString(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	g.Eq(rod.Eval(`() => this.parentElement`).This(el.Object).String(), "() => this.parentElement() button")
}

func TestEvalObjectReferenceChainIsTooLong(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.blank())

	obj, err := p.Evaluate(&rod.EvalOptions{
		JS: `() => {
			let a = {b: 1}
			a.c = a
			return a
		}`,
	})
	g.E(err)

	_, err = p.Eval(`a => a`, obj)
	g.Eq(err.Error(), "{-32000 Object reference chain is too long }")

	val := p.MustEval(`a => a.c.c.c.c.b`, obj)
	g.Eq(val.Int(), 1)
}
