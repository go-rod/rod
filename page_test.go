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
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
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

	t.Eq(ua, "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_0_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	t.Eq(lang, "en")
}

func (t T) PageCloseCancel() {
	page := t.browser.MustPage(t.srcFile("fixtures/prevent-close.html"))
	page.MustElement("body").MustClick() // only focused page will handle beforeunload event

	w, h := page.MustHandleDialog()
	go func() {
		w()
		h(false, "")
	}()
	t.Eq(page.Close().Error(), "page close canceled")

	page.MustEval(`window.onbeforeunload = null`)
	page.MustClose()
}

func (t T) LoadState() {
	t.True(t.page.LoadState(&proto.PageEnable{}))
}

func (t T) DisableDomain() {
	defer t.page.DisableDomain(&proto.PageEnable{})()
}

func (t T) PageContext() {
	t.page.Timeout(time.Hour).CancelTimeout().MustEval(`1`)
}

func (t T) PageActivate() {
	t.page.MustActivate()
}

func (t T) Window() {
	page := t.newPage(t.blank())

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
	page := t.newPage(t.blank())
	page.MustSetViewport(317, 419, 0, false)
	res := page.MustEval(`[window.innerWidth, window.innerHeight]`)
	t.Eq(317, res.Get("0").Int())
	t.Eq(419, res.Get("1").Int())

	page2 := t.newPage(t.blank())
	res = page2.MustEval(`[window.innerWidth, window.innerHeight]`)
	t.Neq(int(317), res.Get("0").Int())
}

func (t T) EmulateDevice() {
	page := t.newPage(t.blank())
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
	page := t.newPage(t.blank())
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageClose{})
		page.MustClose()
	})
}

func (t T) PageAddScriptTag() {
	p := t.page.MustNavigate(t.blank()).MustWaitLoad()

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

func (t T) PageWaitOpen() {
	page := t.page.MustNavigate(t.srcFile("fixtures/open-page.html"))

	wait := page.MustWaitOpen()

	page.MustElement("a").MustClick()

	newPage := wait()
	defer newPage.MustClose()

	t.Eq("new page", newPage.MustEval("window.a").String())
}

func (t T) PageWait() {
	page := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
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
	s.Mux.HandleFunc("/r3", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Location", "/r4")
		rw.WriteHeader(http.StatusFound)
	})
	s.Route("/r4", "")
	s.Route("/", ".html", `<html></html>`)

	page := t.newPage(s.URL()).MustWaitLoad()

	code := ` () => {
		fetch('/r2').then(r => r.text())
		fetch('/r1')
		fetch('/r3')
	}`

	waitReq := ""
	t.browser.Logger(utils.Log(func(msg ...interface{}) {
		typ := msg[0].(rod.TraceType)
		if typ == rod.TraceTypeWaitRequests {
			list := msg[2].(map[string]string)
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
	t.page.MustNavigate(t.blank())
	wait()
}

func (t T) PageWaitEventParseEventOnlyOnce() {
	nav1 := t.page.WaitEvent(&proto.PageFrameNavigated{})
	nav2 := t.page.WaitEvent(&proto.PageFrameNavigated{})
	t.page.MustNavigate(t.blank())
	nav1()
	nav2()
}

func (t T) PageEvent() {
	p := t.browser.MustPage("")
	ctx := t.Context()
	events := p.Context(ctx).Event()
	p.MustNavigate(t.blank())
	for msg := range events {
		if msg.Load(proto.PageFrameStartedLoading{}) {
			break
		}
	}
	utils.Sleep(0.1)
	ctx.Cancel()

	p.Event()
	p.MustClose()
}

func (t T) Alert() {
	page := t.page.MustNavigate(t.srcFile("fixtures/alert.html"))

	wait, handle := page.MustHandleDialog()

	go page.MustElement("button").MustClick()

	e := wait()
	t.Eq(e.Message, "clicked")
	handle(true, "")
}

func (t T) Mouse() {
	page := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse

	mouse.MustScroll(0, 10)
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

func (t T) MouseHoldMultiple() {
	p := t.page.MustNavigate(t.blank())

	p.Mouse.MustDown("left")
	defer p.Mouse.MustUp("left")
	p.Mouse.MustDown("right")
	defer p.Mouse.MustUp("right")
}

func (t T) MouseClick() {
	t.browser.SlowMotion(1)
	defer func() { t.browser.SlowMotion(0) }()

	page := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse
	mouse.MustMove(140, 160)
	mouse.MustClick("left")
	t.True(page.MustHas("[a=ok]"))
}

func (t T) MouseDrag() {
	page := t.newPage("").MustNavigate(t.srcFile("fixtures/drag.html")).MustWaitLoad()
	mouse := page.Mouse

	mouse.MustMove(3, 3)
	mouse.MustDown("left")
	t.E(mouse.Move(60, 80, 3))
	mouse.MustUp("left")

	utils.Sleep(0.3)
	t.Eq(page.MustEval(`dragTrack`).Str(), " move 3 3 down 3 3 move 22 28 move 41 54 move 60 80 up 60 80")
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
	t.Eq(1280, img.Bounds().Dx())
	t.Eq(800, img.Bounds().Dy())
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
	t.Eq(1280, res.Get("w").Int())
	t.Eq(800, res.Get("h").Int())

	p.MustScreenshotFullPage("")

	noEmulation := t.newPage(t.blank())
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
	p.Keyboard.MustPress('A')
	p.Keyboard.MustInsertText(" Test")
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

func (t T) PageInputDate() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	p.MustElement("[type=date]").MustInput("12")
}

func (t T) PageScroll() {
	p := t.page.MustNavigate(t.srcFile("fixtures/scroll.html")).MustWaitLoad()

	p.Mouse.MustMove(30, 30)
	p.Mouse.MustClick(proto.InputMouseButtonLeft)

	p.Mouse.MustScroll(0, 10)
	p.Mouse.MustScroll(100, 190)
	t.E(p.Mouse.Scroll(200, 300, 5))

	p.MustWait(`pageXOffset > 200 && pageYOffset > 300`)
}

func (t T) PageConsoleLog() {
	p := t.newPage(t.blank()).MustWaitLoad()
	e := &proto.RuntimeConsoleAPICalled{}
	wait := p.WaitEvent(e)
	p.MustEval(`console.log(1, {b: ['test']})`)
	wait()
	t.Eq("test", p.MustObjectToJSON(e.Args[1]).Get("b.0").String())
	t.Eq(`1 map[b:[test]]`, p.MustObjectsToJSON(e.Args).Join(" "))
}

func (t T) Fonts() {
	if !utils.InContainer { // No need to test font rendering on regular OS
		t.SkipNow()
	}

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
		t.page.MustNavigate(t.blank())
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageNavigate{})
		t.page.MustNavigate(t.blank())
	})
}

func (t T) PageWaitLoadErr() {
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		t.page.MustWaitLoad()
	})
}

func (t T) PageNavigation() {
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

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	t.Err(p.Reload())
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

func (t T) PageUseNonExistSession() {
	// TODO: chrome bug that hangs for closing non-exist session id
	// Related chrome ticket: https://bugs.chromium.org/p/chromium/issues/detail?id=1151822
	p := t.browser.PageFromSession("nonexist").Timeout(300 * time.Millisecond)
	err := proto.PageClose{}.Call(p)
	t.Is(err, context.DeadlineExceeded)
}

func (t T) PageElementFromObjectErr() {
	p := t.newPage(t.srcFile("./fixtures/click.html"))
	utils.Sleep(0.1)
	res, err := proto.DOMGetNodeForLocation{X: 10, Y: 10}.Call(p)
	t.E(err)

	obj, err := proto.DOMResolveNode{
		BackendNodeID: res.BackendNodeID,
	}.Call(p)
	t.E(err)

	t.mc.stubErr(1, proto.RuntimeEvaluate{})
	t.Err(p.ElementFromObject(obj.Object))
}
