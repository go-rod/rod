package rod_test

import (
	"bytes"
	"context"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
	"github.com/ysmood/gson"
)

func (t T) GetPageURL() {
	t.page.MustNavigate(t.srcFile("fixtures/click-iframe.html")).MustWaitLoad()
	t.Regex(`/fixtures/click-iframe.html\z`, t.page.MustInfo().URL)
}

func (t T) SetCookies() {
	s := t.Serve()

	page := t.page.MustSetCookies(&proto.NetworkCookieParam{
		Name:  "cookie-a",
		Value: "1",
		URL:   s.URL(),
	}, &proto.NetworkCookieParam{
		Name:  "cookie-b",
		Value: "2",
		URL:   s.URL(),
	}).MustNavigate(s.URL())

	cookies := page.MustCookies()

	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Value < cookies[j].Value
	})

	t.Eq("1", cookies[0].Value)
	t.Eq("2", cookies[1].Value)

	t.E(proto.NetworkClearBrowserCookies{}.Call(page))

	cookies = page.MustCookies()
	t.Len(cookies, 0)

	t.Panic(func() {
		t.mc.stubErr(1, proto.TargetGetTargetInfo{})
		page.MustCookies()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.NetworkGetCookies{})
		page.MustCookies()
	})
}

func (t T) SetExtraHeaders() {
	s := t.Serve()

	wg := sync.WaitGroup{}
	var header http.Header
	s.Mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		header = r.Header
		wg.Done()
	})

	p := t.newPage("")
	cleanup := p.MustSetExtraHeaders("a", "1", "b", "2")

	wg.Add(1)
	p.MustNavigate(s.URL())
	wg.Wait()

	t.Eq(header.Get("a"), "1")
	t.Eq(header.Get("b"), "2")

	cleanup()

	// TODO: I don't know why it will fail randomly
	if false {
		wg.Add(1)
		p.MustReload()
		wg.Wait()

		t.Eq(header.Get("a"), "")
		t.Eq(header.Get("b"), "")
	}
}

func (t T) SetUserAgent() {
	s := t.Serve()

	ua := ""
	lang := ""

	wg := sync.WaitGroup{}
	wg.Add(1)

	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		lang = r.Header.Get("Accept-Language")
		wg.Done()
	})

	t.newPage("").MustSetUserAgent(nil).MustNavigate(s.URL())
	wg.Wait()

	t.Eq("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36", ua)
	t.Eq("en", lang)
}

func (t T) PageCloseCancel() {
	page := t.browser.MustPage(t.srcFile("fixtures/prevent-close.html"))
	page.MustElement("body").MustClick() // only focused page will handle beforeunload event

	go page.MustHandleDialog(false, "")()
	t.Eq(page.Close().Error(), "page close canceled")

	// TODO: this is a bug of chrome, it should not kill the target only in headless mode
	if !t.browser.Headless() {
		go page.MustHandleDialog(true, "")()
		page.MustClose()
	}
}

func (t T) LoadState() {
	t.True(t.page.LoadState(&proto.PageEnable{}))
}

func (t T) PageContext() {
	t.page.Timeout(time.Hour).CancelTimeout().MustEval(`1`)
}

func (t T) Release() {
	res, err := t.page.Evaluate(rod.NewEval(`document`).ByObject())
	t.E(err)
	t.page.MustRelease(res)
}

func (t T) Window() {
	page := t.newPage(t.srcFile("fixtures/click.html"))

	t.E(page.SetViewport(nil))

	bounds := page.MustGetWindow()
	defer page.MustSetWindow(
		bounds.Left,
		bounds.Top,
		bounds.Width,
		bounds.Height,
	)

	page.MustWindowMaximize()
	page.MustWindowNormal()
	page.MustWindowFullscreen()
	page.MustWindowNormal()
	page.MustWindowMinimize()
	page.MustWindowNormal()
	page.MustSetWindow(0, 0, 1211, 611)
	t.Eq(1211, page.MustEval(`window.innerWidth`).Int())
	t.Eq(611, page.MustEval(`window.innerHeight`).Int())

	t.Panic(func() {
		t.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustGetWindow()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.BrowserGetWindowBounds{})
		page.MustGetWindow()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustSetWindow(0, 0, 1000, 1000)
	})
}

func (t T) SetViewport() {
	page := t.newPage(t.srcFile("fixtures/click.html"))
	page.MustSetViewport(317, 419, 0, false)
	res := page.MustEval(`[window.innerWidth, window.innerHeight]`)
	t.Eq(317, res.Get("0").Int())
	t.Eq(419, res.Get("1").Int())

	page2 := t.newPage(t.srcFile("fixtures/click.html"))
	res = page2.MustEval(`[window.innerWidth, window.innerHeight]`)
	t.Neq(int(317), res.Get("0").Int())
}

func (t T) EmulateDevice() {
	page := t.newPage(t.srcFile("fixtures/click.html"))
	page.MustEmulate(devices.IPhone6or7or8Plus)
	res := page.MustEval(`[window.innerWidth, window.innerHeight, navigator.userAgent]`)
	t.Eq(980, res.Get("0").Int())
	t.Eq(1743, res.Get("1").Int())
	t.Eq(
		"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
		res.Get("2").String(),
	)
	t.Panic(func() {
		t.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		page.MustEmulate(devices.IPad)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.EmulationSetTouchEmulationEnabled{})
		page.MustEmulate(devices.IPad)
	})
}

func (t T) PageCloseErr() {
	page := t.newPage(t.srcFile("fixtures/click.html"))
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageClose{})
		page.MustClose()
	})
}

func (t T) PageAddScriptTag() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustWaitLoad()

	res := p.MustAddScriptTag(t.srcFile("fixtures/add-script-tag.js")).MustEval(`count()`)
	t.Eq(0, res.Int())

	res = p.MustAddScriptTag(t.srcFile("fixtures/add-script-tag.js")).MustEval(`count()`)
	t.Eq(1, res.Int())

	t.E(p.AddScriptTag("", `let ok = 'yes'`))
	res = p.MustEval(`ok`)
	t.Eq("yes", res.String())
}

func (t T) PageAddStyleTag() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustWaitLoad()

	res := p.MustAddStyleTag(t.srcFile("fixtures/add-style-tag.css")).
		MustElement("h4").MustEval(`getComputedStyle(this).color`)
	t.Eq("rgb(255, 0, 0)", res.String())

	p.MustAddStyleTag(t.srcFile("fixtures/add-style-tag.css"))
	t.Len(p.MustElements("link"), 1)

	t.E(p.AddStyleTag("", "h4 { color: green; }"))
	res = p.MustElement("h4").MustEval(`getComputedStyle(this).color`)
	t.Eq("rgb(0, 128, 0)", res.String())
}

func (t T) PageEvalOnNewDocument() {
	p := t.newPage("")

	p.MustEvalOnNewDocument(`window.rod = 'ok'`)

	// to activate the script
	p.MustNavigate(t.srcFile("fixtures/click.html"))

	t.Eq(p.MustEval("rod").String(), "ok")

	t.Panic(func() {
		t.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		p.MustEvalOnNewDocument(`1`)
	})
}

func (t T) PageEval() {
	page := t.page.MustNavigate(t.srcFile("fixtures/click.html"))

	t.Eq(3, page.MustEval(`
		(a, b) => a + b
	`, 1, 2).Int())
	t.Eq(1, page.MustEval(`a => 1`).Int())
	t.Eq(1, page.MustEval(`function() { return 1 }`).Int())
	t.Eq(1, page.MustEval(`((1))`).Int())
	t.Neq(1, page.MustEval(`a = () => 1`).Int())
	t.Neq(1, page.MustEval(`a = function() { return 1 }`))
	t.Neq(1, page.MustEval(`/* ) */`))

	// reuse obj
	obj := page.MustEvaluate(rod.NewEval(`() => () => 'ok'`).ByObject())
	t.Eq("ok", page.MustEval(`f => f()`, obj).Str())
}

func (t T) PageEvalNilContext() {
	page := t.newPage(t.srcFile("fixtures/click.html"))

	t.mc.stub(1, proto.RuntimeEvaluate{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), &cdp.Error{Code: -32000}
	})
	t.Eq(1, page.MustEval(`1`).Int())
}

func (t T) PageExposeJSHelper() {
	page := t.newPage(t.srcFile("fixtures/click.html"))

	t.Eq("undefined", page.MustEval("typeof(rod)").Str())
	page.ExposeJSHelper()
	t.Eq("object", page.MustEval("typeof(rod)").Str())
}

func (t T) PageWaitOpen() {
	page := t.page.MustNavigate(t.srcFile("fixtures/open-page.html"))

	wait := page.MustWaitOpen()

	page.MustElement("a").MustClick()

	newPage := wait()
	defer newPage.MustClose()

	t.Eq("new page", newPage.MustEval("window.a").String())
}

func (t T) PageWaitPauseOpen() {
	page := t.page.Timeout(5 * time.Second).MustNavigate(t.srcFile("fixtures/open-page.html"))
	defer page.CancelTimeout()

	wait, resume := page.MustWaitPauseOpen()

	go page.MustElement("a").MustClick()

	pageA := wait()
	pageA.MustEvalOnNewDocument(`window.a = 'ok'`)
	resume()
	t.Eq("ok", pageA.MustEval(`window.a`).String())
	pageA.MustClose()

	w := page.MustWaitOpen()
	page.MustElement("a").MustClick()
	pageB := w()
	pageB.MustWait(`window.a == 'new page'`)
	pageB.MustClose()

	{ // enable TargetSetAutoAttach err
		t.mc.stubErr(1, proto.TargetSetAutoAttach{})
		t.Err(t.page.WaitPauseOpen())
	}
	{ // disable TargetSetAutoAttach err
		p := t.page.MustNavigate(t.srcFile("fixtures/open-page.html"))
		wait, resume, _ := p.WaitPauseOpen()
		go p.MustElement("a").MustClick()
		newP, _ := wait()
		t.mc.stubErr(1, proto.TargetSetAutoAttach{})
		t.Err(resume())
		t.Nil(resume())
		newP.MustWaitLoad().MustClose()
	}
}

func (t T) PageWait() {
	page := t.page.Timeout(5 * time.Second).MustNavigate(t.srcFile("fixtures/click.html"))
	page.MustWait(`document.querySelector('button') !== null`)

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		page.MustWait(``)
	})
}

func (t T) PageNavigateBlank() {
	t.page.MustNavigate("")
}

func (t T) PageWaitNavigation() {
	s := t.Serve().Route("/", "")
	wait := t.page.MustWaitNavigation()
	t.page.MustNavigate(s.URL())
	wait()
}

func (t T) PageWaitRequestIdle() {
	s := t.Serve()

	sleep := time.Second

	s.Route("/r1", "")
	s.Mux.HandleFunc("/r2", func(w http.ResponseWriter, r *http.Request) {
		t.E(w.Write([]byte("part")))
		ctx, cancel := context.WithTimeout(t.Context(), sleep)
		defer cancel()
		<-ctx.Done()
	})
	s.Route("/r3", "")
	s.Route("/", ".html", `<html></html>`)

	page := t.newPage(s.URL()).MustWaitLoad()

	code := ` () => {
		fetch('/r2').then(r => r.text())
		fetch('/r1')
		fetch('/r3')
	}`

	waitReq := ""
	t.browser.Logger(utils.Log(func(msg ...interface{}) {
		tm := msg[0].(*rod.TraceMsg)
		if tm.Type == rod.TraceTypeWaitRequests {
			list := tm.Details.(map[string]string)
			for _, v := range list {
				waitReq = v
				break
			}
		}
	}))
	defer t.browser.Logger(rod.DefaultLogger)

	t.browser.Trace(true)
	wait := page.MustWaitRequestIdle("/r1")
	t.browser.Trace(defaults.Trace)
	page.MustEval(code)
	start := time.Now()
	wait()
	t.Gt(time.Since(start), sleep)
	t.Regex("/r2$", waitReq)

	wait = page.MustWaitRequestIdle("/r2")
	page.MustEval(code)
	start = time.Now()
	wait()
	t.Lt(time.Since(start), sleep)

	t.Panic(func() {
		wait()
	})
}

func (t T) PageWaitIdle() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	p.MustElement("button").MustClick()
	p.MustWaitIdle()

	t.True(p.MustHas("[a=ok]"))
}

func (t T) PageEventSession() {
	s := t.Serve()
	p := t.newPage(s.URL())

	p.EnableDomain(proto.NetworkEnable{})
	go t.page.Context(t.Context()).EachEvent(func(e *proto.NetworkRequestWillBeSent) {
		t.Log("should not goes to here")
		t.Fail()
	})()
	p.MustEval(`u => fetch(u)`, s.URL())
}

func (t T) PageWaitEvent() {
	wait := t.page.WaitEvent(&proto.PageFrameNavigated{})
	t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	wait()
}

func (t T) PageEvent() {
	p := t.newPage("")
	s := p.Event().Subscribe(t.Context())
	p.MustNavigate(t.srcFile("fixtures/click.html"))
	for e := range s {
		if rod.Event(e, &proto.PageFrameNavigated{}) {
			break
		}
	}
}

func (t T) Alert() {
	page := t.page.MustNavigate(t.srcFile("fixtures/alert.html"))

	go page.MustHandleDialog(true, "")()
	page.MustElement("button").MustClick()
}

func (t T) Mouse() {
	page := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse

	t.browser.Trace(true)
	mouse.MustScroll(0, 10)
	t.browser.Trace(defaults.Trace)
	mouse.MustMove(140, 160)
	mouse.MustDown("left")
	mouse.MustUp("left")

	t.True(page.MustHas("[a=ok]"))

	t.Panic(func() {
		t.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustScroll(0, 10)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustDown(proto.InputMouseButtonLeft)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustUp(proto.InputMouseButtonLeft)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustClick(proto.InputMouseButtonLeft)
	})
}

func (t T) MouseClick() {
	t.browser.Slowmotion(1)
	defer func() { t.browser.Slowmotion(0) }()

	page := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse
	mouse.MustMove(140, 160)
	mouse.MustClick("left")
	t.True(page.MustHas("[a=ok]"))
}

func (t T) MouseDrag() {
	wait := t.page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	page := t.page.MustNavigate(t.srcFile("fixtures/drag.html")).MustWaitLoad()
	wait()
	mouse := page.Timeout(3 * time.Second).Mouse

	mouse.MustMove(3, 3)
	mouse.MustDown("left")
	t.E(mouse.Move(60, 80, 3))
	mouse.MustUp("left")

	page.MustWait(`dragTrack == " move 3 3 down 3 3 move 22 28 move 41 54 move 60 80 up 60 80"`)
}

func (t T) NativeDrag(got.Skip) { // devtools doesn't support to use mouse event to simulate it for now
	page := t.page.MustNavigate(t.srcFile("fixtures/drag.html"))
	mouse := page.Mouse

	pt := page.MustElement("#draggable").MustShape().OnePointInside()
	toY := page.MustElement(".dropzone:nth-child(2)").MustShape().OnePointInside().Y

	page.Overlay(pt.X, pt.Y, 10, 10, "from")
	page.Overlay(pt.X, toY, 10, 10, "to")

	mouse.MustMove(pt.X, pt.Y)
	mouse.MustDown("left")
	t.E(mouse.Move(pt.X, toY, 5))
	page.MustScreenshot("")
	mouse.MustUp("left")

	page.MustElement(".dropzone:nth-child(2) #draggable")
}

func (t T) Touch() {
	page := t.newPage("").MustEmulate(devices.IPad)

	wait := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	page.MustNavigate(t.srcFile("fixtures/touch.html"))
	wait()

	touch := page.Touch

	touch.MustTap(10, 20)

	p := &proto.InputTouchPoint{X: 30, Y: 40}

	touch.MustStart(p).MustEnd()
	touch.MustStart(p)
	p.MoveTo(50, 60)
	touch.MustMove(p).MustCancel()

	page.MustWait(`touchTrack == ' start 10 20 end start 30 40 end start 30 40 move 50 60 cancel'`)

	t.Panic(func() {
		t.mc.stubErr(1, proto.InputDispatchTouchEvent{})
		touch.MustTap(1, 2)
	})
}

func (t T) PageScreenshot() {
	f := filepath.Join("tmp", "screenshots", t.Srand(16)+".png")
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	p.MustElement("button")
	p.MustScreenshot()
	data := p.MustScreenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	t.E(err)
	t.Eq(800, img.Bounds().Dx())
	t.Eq(600, img.Bounds().Dy())
	t.Nil(os.Stat(f))

	p.MustScreenshot("")

	t.Panic(func() {
		t.mc.stubErr(1, proto.PageCaptureScreenshot{})
		p.MustScreenshot()
	})
}

func (t T) ScreenshotFullPage() {
	p := t.page.MustNavigate(t.srcFile("fixtures/scroll.html"))
	p.MustElement("button")
	data := p.MustScreenshotFullPage()
	img, err := png.Decode(bytes.NewBuffer(data))
	t.E(err)
	res := p.MustEval(`({w: document.documentElement.scrollWidth, h: document.documentElement.scrollHeight})`)
	t.Eq(res.Get("w").Int(), img.Bounds().Dx())
	t.Eq(res.Get("h").Int(), img.Bounds().Dy())

	// after the full page screenshot the window size should be the same as before
	res = p.MustEval(`({w: innerWidth, h: innerHeight})`)
	t.Eq(800, res.Get("w").Int())
	t.Eq(600, res.Get("h").Int())

	p.MustScreenshotFullPage("")

	noEmulation := t.newPage(t.srcFile("fixtures/click.html"))
	t.E(noEmulation.SetViewport(nil))
	noEmulation.MustScreenshotFullPage()

	t.Panic(func() {
		t.mc.stubErr(1, proto.PageGetLayoutMetrics{})
		p.MustScreenshotFullPage()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		p.MustScreenshotFullPage()
	})
}

func (t T) ScreenshotFullPageInit() {
	p := t.newPage(t.srcFile("fixtures/scroll.html"))

	// should not panic
	p.MustScreenshotFullPage()
}

func (t T) PageInput() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))

	el := p.MustElement("input")
	el.MustFocus()
	t.browser.Trace(true)
	p.Keyboard.MustPress('A')
	p.Keyboard.MustInsertText(" Test")
	t.browser.Trace(defaults.Trace)
	p.Keyboard.MustPress(input.Tab)

	t.Eq("A Test", el.MustText())

	t.Panic(func() {
		t.mc.stubErr(1, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustDown('a')
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustUp('a')
	})
	t.Panic(func() {
		t.mc.stubErr(3, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustPress('a')
	})
}

func (t T) PageScroll() {
	p := t.page.MustNavigate(t.srcFile("fixtures/scroll.html")).MustWaitLoad()

	p.Mouse.MustScroll(0, 10)
	p.Mouse.MustScroll(100, 190)
	t.E(p.Mouse.Scroll(200, 300, 5))
	p.MustElement("button").MustWaitStable()
	offset := p.MustEval("({x: window.pageXOffset, y: window.pageYOffset})")
	t.Lt(int(300), offset.Get("y").Int())
}

func (t T) PageConsoleLog() {
	p := t.newPage(t.srcFile("fixtures/click.html")).MustWaitLoad()
	e := &proto.RuntimeConsoleAPICalled{}
	wait := p.WaitEvent(e)
	p.MustEval(`console.log(1, {b: ['test']})`)
	wait()
	t.Eq("test", p.MustObjectToJSON(e.Args[1]).Get("b.0").String())
	t.Eq(`1 map[b:[test]]`, p.MustObjectsToJSON(e.Args).Join(" "))
}

func (t T) PageOthers() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))

	t.Eq("body", p.MustElementByJS(`document.body`).MustDescribe().LocalName)
	t.Len(p.MustElementsByJS(`document.querySelectorAll('input')`), 5)
	t.Eq(1, p.MustEval(`1`).Int())

	p.Mouse.MustDown("left")
	defer p.Mouse.MustUp("left")
	p.Mouse.MustDown("right")
	defer p.Mouse.MustUp("right")
}

func (t T) Fonts() {
	t.timeoutAfter(time.Minute)

	p := t.page.MustNavigate(t.srcFile("fixtures/fonts.html")).MustWaitLoad()

	p.MustPDF("tmp", "fonts.pdf") // download the file from Github Actions Artifacts
}

func (t T) PagePDF() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	p.MustPDF("")

	t.Panic(func() {
		t.mc.stubErr(1, proto.PagePrintToPDF{})
		p.MustPDF()
	})
}

func (t T) PageExpose() {
	cb, stop := t.page.MustExpose("exposedFunc")

	t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustWaitLoad()

	t.page.MustEval(`exposedFunc({a: 'ok'})`)
	t.Eq("ok", (<-cb)[0].Get("a").Str())

	t.page.MustEval(`exposedFunc('ok')`)
	stop()

	t.Panic(func() {
		stop()
	})
	t.Panic(func() {
		t.page.MustReload().MustWaitLoad().MustEval(`exposedFunc()`)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		t.page.MustExpose("exposedFunc")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeAddBinding{})
		t.page.MustExpose("exposedFunc2")
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
		id := t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustElement(`body`).MustNodeID()
		t.mc.stubErr(1, proto.DOMResolveNode{})
		t.page.MustElementFromNode(id)
	})
	t.Panic(func() {
		id := t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustElement(`body`).MustNodeID()
		t.mc.stubErr(1, proto.DOMDescribeNode{})
		t.page.MustElementFromNode(id)
	})
}

func (t T) PageNavigateErr() {
	// dns error
	err := t.page.Navigate("http://" + t.Srand(16))
	t.Is(err, &rod.ErrNavigation{})
	t.Is(err.Error(), "navigation failed: net::ERR_NAME_NOT_RESOLVED")

	s := t.Serve()

	s.Mux.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	s.Mux.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	// will not panic
	t.page.MustNavigate(s.URL("/404"))
	t.page.MustNavigate(s.URL("/500"))

	t.Panic(func() {
		t.mc.stubErr(1, proto.PageStopLoading{})
		t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageNavigate{})
		t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	})
}

func (t T) PageWaitLoadErr() {
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		t.page.MustWaitLoad()
	})
}

func (t T) PageGoBackGoForward() {
	p := t.newPage("").MustReload()

	wait := p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigate(t.srcFile("fixtures/click.html"))
	wait()

	wait = p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigate(t.srcFile("fixtures/selector.html"))
	wait()

	wait = p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigateBack()
	wait()
	t.Regex("fixtures/click.html$", p.MustInfo().URL)

	wait = p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigateForward()
	wait()
	t.Regex("fixtures/selector.html$", p.MustInfo().URL)
}

func (t T) PageInitJSErr() {
	p := t.newPage(t.srcFile("fixtures/click-iframe.html")).MustElement("iframe").MustFrame()

	t.Panic(func() {
		t.mc.stubErr(1, proto.PageCreateIsolatedWorld{})
		p.MustEval(`1`)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeEvaluate{})
		p.MustEval(`1`)
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		p.MustEval(`1`)
	})
}

func (t T) PagePool() {
	pool := rod.NewPagePool(3)
	create := func() *rod.Page { return t.browser.MustPage("") }
	p := pool.Get(create)
	pool.Put(p)
	pool.Cleanup(func(p *rod.Page) {
		p.MustClose()
	})
}
