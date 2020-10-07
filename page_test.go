package rod_test

import (
	"bytes"
	"context"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func (c C) GetPageURL() {
	c.page.MustNavigate(c.srcFile("fixtures/click-iframe.html")).MustWaitLoad()
	c.Regex(`/fixtures/click-iframe.html\z`, c.page.MustInfo().URL)
}

func (c C) SetCookies() {
	s := c.Serve()

	page := c.page.MustSetCookies(&proto.NetworkCookieParam{
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

	c.Eq("1", cookies[0].Value)
	c.Eq("2", cookies[1].Value)

	c.E(proto.NetworkClearBrowserCookies{}.Call(page))

	cookies = page.MustCookies()
	c.Len(cookies, 0)

	c.Panic(func() {
		c.mc.stubErr(1, proto.TargetGetTargetInfo{})
		page.MustCookies()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.NetworkGetCookies{})
		page.MustCookies()
	})
}

func (c C) SetExtraHeaders() {
	s := c.Serve()
	s.Route("/", ".html", `ok`)

	page := c.browser.MustPage("")
	defer page.MustClose()

	cleanup := page.MustSetExtraHeaders("a", "1", "b", "2")

	var e proto.NetworkResponseReceived
	wait := page.WaitEvent(&e)
	page.MustNavigate(s.URL())
	wait()

	c.Eq("1", e.Response.RequestHeaders["a"].Str())
	c.Eq("2", e.Response.RequestHeaders["b"].Str())

	cleanup()

	e = proto.NetworkResponseReceived{}
	wait = page.WaitEvent(&e)
	page.MustNavigate(s.URL())
	wait()

	c.Nil(e.Response.RequestHeaders["a"].Val())
	c.Nil(e.Response.RequestHeaders["b"].Val())
}

func (c C) SetUserAgent() {
	s := c.Serve()

	ua := ""
	lang := ""

	wg := sync.WaitGroup{}
	wg.Add(1)

	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		lang = r.Header.Get("Accept-Language")
		wg.Done()
	})

	p := c.browser.MustPage("").MustSetUserAgent(nil).MustNavigate(s.URL())
	defer p.MustClose()
	wg.Wait()

	c.Eq("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36", ua)
	c.Eq("en", lang)
}

func (c C) PageCloseCancel() {
	page := c.browser.MustPage(c.srcFile("fixtures/prevent-close.html"))
	page.MustElement("body").MustClick() // only focused page will handle beforeunload event

	go page.MustHandleDialog(false, "")()
	c.Eq(rod.ErrPageCloseCanceled, page.Close())

	// TODO: this is a bug of chrome, it should not kill the target only in headless mode
	if !c.browser.Headless() {
		go page.MustHandleDialog(true, "")()
		page.MustClose()
	}
}

func (c C) LoadState() {
	c.True(c.page.LoadState(&proto.PageEnable{}))
}

func (c C) PageContext() {
	c.page.Timeout(time.Hour).CancelTimeout().MustEval(`1`)
}

func (c C) Release() {
	res, err := c.page.Evaluate(rod.NewEval(`document`).ByObject())
	c.E(err)
	c.page.MustRelease(res)
}

func (c C) Window() {
	page := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer page.MustClose()

	c.E(page.SetViewport(nil))

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
	c.Eq(1211, page.MustEval(`window.innerWidth`).Int())
	c.Eq(611, page.MustEval(`window.innerHeight`).Int())

	c.Panic(func() {
		c.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustGetWindow()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.BrowserGetWindowBounds{})
		page.MustGetWindow()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustSetWindow(0, 0, 1000, 1000)
	})
}

func (c C) SetViewport() {
	page := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer page.MustClose()
	page.MustSetViewport(317, 419, 0, false)
	res := page.MustEval(`[window.innerWidth, window.innerHeight]`)
	c.Eq(317, res.Get("0").Int())
	c.Eq(419, res.Get("1").Int())

	page2 := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer page2.MustClose()
	res = page2.MustEval(`[window.innerWidth, window.innerHeight]`)
	c.Neq(int(317), res.Get("0").Int())
}

func (c C) EmulateDevice() {
	page := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer page.MustClose()
	page.MustEmulate(devices.IPhone6or7or8Plus)
	res := page.MustEval(`[window.innerWidth, window.innerHeight, navigator.userAgent]`)
	c.Eq(980, res.Get("0").Int())
	c.Eq(1743, res.Get("1").Int())
	c.Eq(
		"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
		res.Get("2").String(),
	)
	c.Panic(func() {
		c.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		page.MustEmulate(devices.IPad)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.EmulationSetTouchEmulationEnabled{})
		page.MustEmulate(devices.IPad)
	})
}

func (c C) PageCloseErr() {
	page := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer page.MustClose()
	c.Panic(func() {
		c.mc.stubErr(1, proto.PageStopLoading{})
		page.MustClose()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.PageClose{})
		page.MustClose()
	})
}

func (c C) PageAddScriptTag() {
	p := c.page.MustNavigate(c.srcFile("fixtures/click.html")).MustWaitLoad()

	res := p.MustAddScriptTag(c.srcFile("fixtures/add-script-tag.js")).MustEval(`count()`)
	c.Eq(0, res.Int())

	res = p.MustAddScriptTag(c.srcFile("fixtures/add-script-tag.js")).MustEval(`count()`)
	c.Eq(1, res.Int())

	c.E(p.AddScriptTag("", `let ok = 'yes'`))
	res = p.MustEval(`ok`)
	c.Eq("yes", res.String())
}

func (c C) PageAddStyleTag() {
	p := c.page.MustNavigate(c.srcFile("fixtures/click.html")).MustWaitLoad()

	res := p.MustAddStyleTag(c.srcFile("fixtures/add-style-tag.css")).
		MustElement("h4").MustEval(`getComputedStyle(this).color`)
	c.Eq("rgb(255, 0, 0)", res.String())

	p.MustAddStyleTag(c.srcFile("fixtures/add-style-tag.css"))
	c.Len(p.MustElements("link"), 1)

	c.E(p.AddStyleTag("", "h4 { color: green; }"))
	res = p.MustElement("h4").MustEval(`getComputedStyle(this).color`)
	c.Eq("rgb(0, 128, 0)", res.String())
}

func (c C) PageEvalOnNewDocument() {
	p := c.browser.MustPage("")
	defer p.MustClose()

	p.MustEvalOnNewDocument(`
  		Object.defineProperty(navigator, 'rod', {
    		get: () => "rod",
  		});`)

	// to activate the script
	p.MustNavigate("")

	c.Eq("rod", p.MustEval("navigator.rod").String())

	c.Panic(func() {
		c.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		p.MustEvalOnNewDocument(`1`)
	})
}

func (c C) PageEval() {
	page := c.page.MustNavigate(c.srcFile("fixtures/click.html"))

	c.Eq(3, page.MustEval(`
		(a, b) => a + b
	`, 1, 2).Int())
	c.Eq(1, page.MustEval(`a => 1`).Int())
	c.Eq(1, page.MustEval(`function() { return 1 }`).Int())
	c.Eq(1, page.MustEval(`((1))`).Int())
	c.Neq(1, page.MustEval(`a = () => 1`).Int())
	c.Neq(1, page.MustEval(`a = function() { return 1 }`))
	c.Neq(1, page.MustEval(`/* ) */`))

	// reuse obj
	obj := page.MustEvaluate(rod.NewEval(`() => () => 'ok'`).ByObject())
	c.Eq("ok", page.MustEval(`f => f()`, obj).Str())
}

func (c C) PageEvalNilContext() {
	page := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer page.MustClose()

	c.mc.stub(1, proto.RuntimeEvaluate{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), &cdp.Error{Code: -32000}
	})
	c.Eq(1, page.MustEval(`1`).Int())
}

func (c C) PageExposeJSHelper() {
	page := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer page.MustClose()

	c.Eq("undefined", page.MustEval("typeof(rod)").Str())
	page.ExposeJSHelper()
	c.Eq("object", page.MustEval("typeof(rod)").Str())
}

func (c C) PageWaitOpen() {
	page := c.page.Timeout(3 * time.Second).MustNavigate(c.srcFile("fixtures/open-page.html"))
	defer page.CancelTimeout()

	wait := page.MustWaitOpen()

	page.MustElement("a").MustClick()

	newPage := wait()
	defer newPage.MustClose()

	c.Eq("new page", newPage.MustEval("window.a").String())
}

func (c C) PageWaitPauseOpen() {
	page := c.page.Timeout(5 * time.Second).MustNavigate(c.srcFile("fixtures/open-page.html"))
	defer page.CancelTimeout()

	wait, resume := page.MustWaitPauseOpen()

	go page.MustElement("a").MustClick()

	pageA := wait()
	pageA.MustEvalOnNewDocument(`window.a = 'ok'`)
	resume()
	c.Eq("ok", pageA.MustEval(`window.a`).String())
	pageA.MustClose()

	w := page.MustWaitOpen()
	page.MustElement("a").MustClick()
	pageB := w()
	pageB.MustWait(`window.a == 'new page'`)
	pageB.MustClose()

	c.Panic(func() {
		defer func() {
			_ = proto.TargetSetAutoAttach{
				Flatten: true,
			}.Call(c.browser)
		}()

		p := c.browser.MustPage("")
		defer p.MustClose()
		c.mc.stubErr(1, proto.TargetSetAutoAttach{})
		p.MustWaitPauseOpen()
	})
	c.Panic(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		defer func() {
			_ = proto.TargetSetAutoAttach{
				Flatten: true,
			}.Call(c.browser)
		}()

		p := c.browser.Context(ctx).MustPage("")
		defer p.MustClose()
		c.mc.stubErr(2, proto.TargetSetAutoAttach{})
		_, r := p.MustWaitPauseOpen()
		r()
	})
}

func (c C) PageWait() {
	page := c.page.Timeout(5 * time.Second).MustNavigate(c.srcFile("fixtures/click.html"))
	page.MustWait(`document.querySelector('button') !== null`)

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		page.MustWait(``)
	})
}
func (c C) PageWaitNavigation() {
	s := c.Serve().Route("/", "")
	wait := c.page.MustWaitNavigation()
	c.page.MustNavigate(s.URL())
	wait()
}

func (c C) PageWaitRequestIdle() {
	s := c.Serve()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sleep := 2 * time.Second

	s.Route("/r1", "")
	s.Mux.HandleFunc("/r2", func(w http.ResponseWriter, r *http.Request) {
		c.E(w.Write([]byte("part")))
		ctx, cancel := context.WithTimeout(ctx, sleep)
		defer cancel()
		<-ctx.Done()
	})
	s.Route("/r3", "")
	s.Route("/", ".html", `<html></html>`)

	page := c.browser.MustPage(s.URL()).MustWaitLoad()
	defer page.MustClose()

	code := ` () => {
		fetch('/r2').then(r => r.text())
		fetch('/r1')
		fetch('/r3')
	}`

	waitReq := ""
	c.browser.Logger(utils.Log(func(msg ...interface{}) {
		tm := msg[0].(*rod.TraceMsg)
		if tm.Type == rod.TraceTypeWaitRequests {
			list := tm.Details.(map[string]string)
			for _, v := range list {
				waitReq = v
				break
			}
		}
	}))
	defer c.browser.Logger(rod.DefaultLogger)

	c.browser.Trace(true)
	wait := page.MustWaitRequestIdle("/r1")
	c.browser.Trace(defaults.Trace)
	page.MustEval(code)
	start := time.Now()
	wait()
	c.Gt(time.Since(start), sleep)
	c.Regex("/r2$", waitReq)

	wait = page.MustWaitRequestIdle("/r2")
	page.MustEval(code)
	start = time.Now()
	wait()
	c.Lt(time.Since(start), time.Second)

	c.Panic(func() {
		wait()
	})
}

func (c C) PageWaitIdle() {
	p := c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	p.MustElement("button").MustClick()
	p.MustWaitIdle()

	c.True(p.MustHas("[a=ok]"))
}

func (c C) PageWaitEvent() {
	wait := c.page.WaitEvent(&proto.PageFrameNavigated{})
	c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	wait()
}

func (c C) Alert() {
	page := c.page.MustNavigate(c.srcFile("fixtures/alert.html"))

	go page.MustHandleDialog(true, "")()
	page.MustElement("button").MustClick()
}

func (c C) Mouse() {
	page := c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse

	c.browser.Trace(true)
	mouse.MustScroll(0, 10)
	c.browser.Trace(defaults.Trace)
	mouse.MustMove(140, 160)
	mouse.MustDown("left")
	mouse.MustUp("left")

	c.True(page.MustHas("[a=ok]"))

	c.Panic(func() {
		c.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustScroll(0, 10)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustDown(proto.InputMouseButtonLeft)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustUp(proto.InputMouseButtonLeft)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustClick(proto.InputMouseButtonLeft)
	})
}

func (c C) MouseClick() {
	c.browser.Slowmotion(1)
	defer func() { c.browser.Slowmotion(0) }()

	page := c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse
	mouse.MustMove(140, 160)
	mouse.MustClick("left")
	c.True(page.MustHas("[a=ok]"))
}

func (c C) MouseDrag() {
	wait := c.page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	page := c.page.MustNavigate(c.srcFile("fixtures/drag.html")).MustWaitLoad()
	wait()
	mouse := page.Mouse

	logs := []string{}
	wait = page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
		log := page.MustObjectsToJSON(e.Args).Join(" ")
		logs = append(logs, log)
		if strings.HasPrefix(log, `up`) {
			return true
		}
		return false
	})

	mouse.MustMove(3, 3)
	mouse.MustDown("left")
	c.E(mouse.Move(60, 80, 3))
	mouse.MustUp("left")

	wait()

	c.Eq([]string{"move 3 3", "down 3 3", "move 22 28", "move 41 54", "move 60 80", "up 60 80"}, logs)
}

func (c C) NativeDrag() {
	// devtools doesn't support to use mouse event to simulate it for now
	c.Testable.(*testing.T).SkipNow()

	page := c.page.MustNavigate(c.srcFile("fixtures/drag.html"))
	mouse := page.Mouse

	pt := page.MustElement("#draggable").MustShape().OnePointInside()
	toY := page.MustElement(".dropzone:nth-child(2)").MustShape().OnePointInside().Y

	page.Overlay(pt.X, pt.Y, 10, 10, "from")
	page.Overlay(pt.X, toY, 10, 10, "to")

	mouse.MustMove(pt.X, pt.Y)
	mouse.MustDown("left")
	c.E(mouse.Move(pt.X, toY, 5))
	page.MustScreenshot("")
	mouse.MustUp("left")

	page.MustElement(".dropzone:nth-child(2) #draggable")
}

func (c C) Touch() {
	page := c.browser.MustPage("")
	defer page.MustClose()

	page.MustEmulate(devices.IPad).
		MustNavigate(c.srcFile("fixtures/touch.html")).
		MustWaitLoad()

	wait := make(chan struct{})
	logs := []string{}
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
		log := page.MustObjectsToJSON(e.Args).Join(" ")
		logs = append(logs, log)
		if strings.HasPrefix(log, `cancel`) {
			close(wait)
			return true
		}
		return false
	})()

	touch := page.Touch

	touch.MustTap(10, 20)

	p := &proto.InputTouchPoint{X: 30, Y: 40}

	touch.MustStart(p).MustEnd()
	touch.MustStart(p)
	p.MoveTo(50, 60)
	touch.MustMove(p).MustCancel()

	<-wait

	c.Eq([]string{"start 10 20", "end", "start 30 40", "end", "start 30 40", "move 50 60", "cancel"}, logs)

	c.Panic(func() {
		c.mc.stubErr(1, proto.InputDispatchTouchEvent{})
		touch.MustTap(1, 2)
	})
}

func (c C) PageScreenshot() {
	f := filepath.Join("tmp", "screenshots", c.Srand(16)+".png")
	p := c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	p.MustElement("button")
	p.MustScreenshot()
	data := p.MustScreenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	c.E(err)
	c.Eq(800, img.Bounds().Dx())
	c.Eq(600, img.Bounds().Dy())
	c.Nil(os.Stat(f))

	p.MustScreenshot("")

	c.Panic(func() {
		c.mc.stubErr(1, proto.PageCaptureScreenshot{})
		p.MustScreenshot()
	})
}

func (c C) ScreenshotFullPage() {
	p := c.page.MustNavigate(c.srcFile("fixtures/scroll.html"))
	p.MustElement("button")
	data := p.MustScreenshotFullPage()
	img, err := png.Decode(bytes.NewBuffer(data))
	c.E(err)
	res := p.MustEval(`({w: document.documentElement.scrollWidth, h: document.documentElement.scrollHeight})`)
	c.Eq(res.Get("w").Int(), img.Bounds().Dx())
	c.Eq(res.Get("h").Int(), img.Bounds().Dy())

	// after the full page screenshot the window size should be the same as before
	res = p.MustEval(`({w: innerWidth, h: innerHeight})`)
	c.Eq(800, res.Get("w").Int())
	c.Eq(600, res.Get("h").Int())

	p.MustScreenshotFullPage("")

	noEmulation := c.browser.MustPage(c.srcFile("fixtures/click.html"))
	defer noEmulation.MustClose()
	c.E(noEmulation.SetViewport(nil))
	noEmulation.MustScreenshotFullPage()

	c.Panic(func() {
		c.mc.stubErr(1, proto.PageGetLayoutMetrics{})
		p.MustScreenshotFullPage()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		p.MustScreenshotFullPage()
	})
}

func (c C) ScreenshotFullPageInit() {
	p := c.browser.MustPage(c.srcFile("fixtures/scroll.html"))
	defer p.MustClose()

	// should not panic
	p.MustScreenshotFullPage()
}

func (c C) PageInput() {
	p := c.page.MustNavigate(c.srcFile("fixtures/input.html"))

	el := p.MustElement("input")
	el.MustFocus()
	c.browser.Trace(true)
	p.Keyboard.MustPress('A')
	p.Keyboard.MustInsertText(" Test")
	c.browser.Trace(defaults.Trace)
	p.Keyboard.MustPress(input.Tab)

	c.Eq("A Test", el.MustText())

	c.Panic(func() {
		c.mc.stubErr(1, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustDown('a')
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustUp('a')
	})
	c.Panic(func() {
		c.mc.stubErr(3, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustPress('a')
	})
}

func (c C) PageScroll() {
	p := c.page.MustNavigate(c.srcFile("fixtures/scroll.html")).MustWaitLoad()

	p.Mouse.MustScroll(0, 10)
	p.Mouse.MustScroll(100, 190)
	c.E(p.Mouse.Scroll(200, 300, 5))
	p.MustElement("button").MustWaitStable()
	offset := p.MustEval("({x: window.pageXOffset, y: window.pageYOffset})")
	c.Lt(int(300), offset.Get("y").Int())
}

func (c C) PageConsoleLog() {
	p := c.page.MustNavigate("").MustWaitLoad()
	e := &proto.RuntimeConsoleAPICalled{}
	wait := p.WaitEvent(e)
	p.MustEval(`console.log(1, {b: ['test']})`)
	wait()
	c.Eq("test", p.MustObjectToJSON(e.Args[1]).Get("b.0").String())
	c.Eq(`1 map[b:[test]]`, p.MustObjectsToJSON(e.Args).Join(" "))
}

func (c C) PageOthers() {
	p := c.page.MustNavigate(c.srcFile("fixtures/input.html"))

	c.Eq("body", p.MustElementByJS(`document.body`).MustDescribe().LocalName)
	c.Len(p.MustElementsByJS(`document.querySelectorAll('input')`), 5)
	c.Eq(1, p.MustEval(`1`).Int())

	p.Mouse.MustDown("left")
	defer p.Mouse.MustUp("left")
	p.Mouse.MustDown("right")
	defer p.Mouse.MustUp("right")
}

func (c C) Fonts() {
	p := c.page.MustNavigate(c.srcFile("fixtures/fonts.html")).MustWaitLoad()

	p.MustPDF("tmp", "fonts.pdf") // download the file from Github Actions Artifacts
}

func (c C) PagePDF() {
	p := c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	p.MustPDF("")

	c.Panic(func() {
		c.mc.stubErr(1, proto.PagePrintToPDF{})
		p.MustPDF()
	})
}

func (c C) PageExpose() {
	cb, stop := c.page.MustExpose("exposedFunc")

	c.page.MustNavigate(c.srcFile("fixtures/click.html")).MustWaitLoad()

	c.page.MustEval(`exposedFunc({a: 'ok'})`)
	c.Eq("ok", (<-cb)[0].Get("a").Str())

	c.page.MustEval(`exposedFunc('ok')`)
	stop()

	c.Panic(func() {
		stop()
	})
	c.Panic(func() {
		c.page.MustReload().MustWaitLoad().MustEval(`exposedFunc()`)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		c.page.MustExpose("exposedFunc")
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeAddBinding{})
		c.page.MustExpose("exposedFunc2")
	})
}

func (c C) PageObjectErr() {
	c.Panic(func() {
		c.page.MustObjectToJSON(&proto.RuntimeRemoteObject{
			ObjectID: "not-exists",
		})
	})
	c.Panic(func() {
		c.page.MustElementFromNode(-1)
	})
	c.Panic(func() {
		id := c.page.MustNavigate(c.srcFile("fixtures/click.html")).MustElement(`body`).MustNodeID()
		c.mc.stubErr(1, proto.DOMResolveNode{})
		c.page.MustElementFromNode(id)
	})
	c.Panic(func() {
		id := c.page.MustNavigate(c.srcFile("fixtures/click.html")).MustElement(`body`).MustNodeID()
		c.mc.stubErr(1, proto.DOMDescribeNode{})
		c.page.MustElementFromNode(id)
	})
}

func (c C) PageNavigateErr() {
	// dns error
	c.Panic(func() {
		c.page.MustNavigate("http://" + c.Srand(16))
	})

	s := c.Serve()

	s.Mux.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	s.Mux.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	// will not panic
	c.page.MustNavigate(s.URL("/404"))
	c.page.MustNavigate(s.URL("/500"))

	c.Panic(func() {
		c.mc.stubErr(1, proto.PageStopLoading{})
		c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.PageNavigate{})
		c.page.MustNavigate(c.srcFile("fixtures/click.html"))
	})
}

func (c C) PageWaitLoadErr() {
	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		c.page.MustWaitLoad()
	})
}

func (c C) PageGoBackGoForward() {
	p := c.browser.MustPage("").MustReload()
	defer p.MustClose()

	p.
		MustNavigate(c.srcFile("fixtures/click.html")).MustWaitLoad().
		MustNavigate(c.srcFile("fixtures/selector.html")).MustWaitLoad()

	p.MustNavigateBack().MustWaitLoad()
	c.Regex("fixtures/click.html$", p.MustInfo().URL)

	p.MustNavigateForward().MustWaitLoad()
	c.Regex("fixtures/selector.html$", p.MustInfo().URL)
}

func (c C) PageInitJSErr() {
	p := c.browser.MustPage(c.srcFile("fixtures/click-iframe.html")).MustElement("iframe").MustFrame()
	defer p.MustClose()

	c.Panic(func() {
		c.mc.stubErr(1, proto.PageCreateIsolatedWorld{})
		p.MustEval(`1`)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeEvaluate{})
		p.MustEval(`1`)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		p.MustEval(`1`)
	})
}
