package rod_test

import (
	"bytes"
	"context"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

func (s *S) TestGetPageURL() {
	s.page.MustNavigate(srcFile("fixtures/click-iframe.html")).MustWaitLoad()
	s.Regexp(`/fixtures/click-iframe.html\z`, s.page.MustInfo().URL)
}

func (s *S) TestSetCookies() {
	url, _, close := utils.Serve("")
	defer close()

	page := s.page.MustSetCookies(&proto.NetworkCookieParam{
		Name:  "a",
		Value: "1",
		URL:   url,
	}, &proto.NetworkCookieParam{
		Name:  "b",
		Value: "2",
		URL:   url,
	}).MustNavigate(url)

	cookies := page.MustCookies()

	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Value < cookies[j].Value
	})

	s.Equal("1", cookies[0].Value)
	s.Equal("2", cookies[1].Value)

	s.Panics(func() {
		s.mc.stubErr(1, proto.TargetGetTargetInfo{})
		page.MustCookies()
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.NetworkGetCookies{})
		page.MustCookies()
	})
}

func (s *S) TestSetExtraHeaders() {
	url, mux, close := utils.Serve("")
	defer close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	var out1, out2 string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		out1 = r.Header.Get("a")
		out2 = r.Header.Get("b")
		wg.Done()
	})

	page := s.browser.MustPage("")
	defer page.MustClose()

	defer page.MustSetExtraHeaders("a", "1", "b", "2")()
	page.MustNavigate(url)
	wg.Wait()

	s.Equal("1", out1)
	s.Equal("2", out2)
}

func (s *S) TestSetUserAgent() {
	url, mux, close := utils.Serve("")
	defer close()

	ua := ""
	lang := ""

	wg := sync.WaitGroup{}
	wg.Add(1)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		lang = r.Header.Get("Accept-Language")
		wg.Done()
	})

	p := s.browser.MustPage("").MustSetUserAgent(nil).MustNavigate(url)
	defer p.MustClose()
	wg.Wait()

	s.Equal("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36", ua)
	s.Equal("en", lang)
}

func (s *S) TestPageCloseCancel() {
	page := s.browser.MustPage(srcFile("fixtures/prevent-close.html"))
	page.MustElement("body").MustClick() // only focused page will handle beforeunload event

	go page.MustHandleDialog(false, "")()
	s.Equal(rod.ErrPageCloseCanceled, page.Close())

	// TODO: this is a bug of chrome, it should not kill the target only in headless mode
	if !s.browser.Headless() {
		go page.MustHandleDialog(true, "")()
		page.MustClose()
	}
}

func (s *S) TestLoadState() {
	s.True(s.page.LoadState(&proto.PageEnable{}))
}

func (s *S) TestPageContext() {
	s.page.Timeout(time.Hour).CancelTimeout().MustEval(`1`)
}

func (s *S) TestRelease() {
	res, err := s.page.EvalWithOptions(rod.NewEvalOptions(`document`, nil).ByObject())
	utils.E(err)
	s.page.MustRelease(res.ObjectID)
}

func (s *S) TestWindow() {
	page := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer page.MustClose()

	utils.E(page.SetViewport(nil))

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
	s.EqualValues(1211, page.MustEval(`window.innerWidth`).Int())
	s.EqualValues(611, page.MustEval(`window.innerHeight`).Int())

	s.Panics(func() {
		s.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustGetWindow()
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.BrowserGetWindowBounds{})
		page.MustGetWindow()
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustSetWindow(0, 0, 1000, 1000)
	})
}

func (s *S) TestSetViewport() {
	page := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer page.MustClose()
	page.MustSetViewport(317, 419, 0, false)
	res := page.MustEval(`[window.innerWidth, window.innerHeight]`)
	s.EqualValues(317, res.Get("0").Int())
	s.EqualValues(419, res.Get("1").Int())

	page2 := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer page2.MustClose()
	res = page2.MustEval(`[window.innerWidth, window.innerHeight]`)
	s.NotEqual(int64(317), res.Get("0").Int())
}

func (s *S) TestEmulateDevice() {
	page := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer page.MustClose()
	page.MustEmulate(devices.IPhone6or7or8Plus)
	res := page.MustEval(`[window.innerWidth, window.innerHeight, navigator.userAgent]`)
	s.EqualValues(980, res.Get("0").Int())
	s.EqualValues(1743, res.Get("1").Int())
	s.Equal(
		"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
		res.Get("2").String(),
	)
	s.Panics(func() {
		s.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		page.MustEmulate(devices.IPad)
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.EmulationSetTouchEmulationEnabled{})
		page.MustEmulate(devices.IPad)
	})
}

func (s *S) TestPageCloseErr() {
	page := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer page.MustClose()
	s.Panics(func() {
		s.mc.stubErr(1, proto.PageStopLoading{})
		page.MustClose()
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.PageClose{})
		page.MustClose()
	})
}

func (s *S) TestPageAddScriptTag() {
	p := s.page.MustNavigate(srcFile("fixtures/click.html")).MustWaitLoad()

	res := p.MustAddScriptTag(srcFile("fixtures/add-script-tag.js")).MustEval(`count()`)
	s.EqualValues(0, res.Int())

	res = p.MustAddScriptTag(srcFile("fixtures/add-script-tag.js")).MustEval(`count()`)
	s.EqualValues(1, res.Int())

	utils.E(p.AddScriptTag("", `let ok = 'yes'`))
	res = p.MustEval(`ok`)
	s.Equal("yes", res.String())
}

func (s *S) TestPageAddStyleTag() {
	p := s.page.MustNavigate(srcFile("fixtures/click.html")).MustWaitLoad()

	res := p.MustAddStyleTag(srcFile("fixtures/add-style-tag.css")).
		MustElement("h4").MustEval(`getComputedStyle(this).color`)
	s.Equal("rgb(255, 0, 0)", res.String())

	p.MustAddStyleTag(srcFile("fixtures/add-style-tag.css"))
	s.Len(p.MustElements("link"), 1)

	utils.E(p.AddStyleTag("", "h4 { color: green; }"))
	res = p.MustElement("h4").MustEval(`getComputedStyle(this).color`)
	s.Equal("rgb(0, 128, 0)", res.String())
}

func (s *S) TestPageEvalOnNewDocument() {
	p := s.browser.MustPage("")
	defer p.MustClose()

	p.MustEvalOnNewDocument(`
  		Object.defineProperty(navigator, 'rod', {
    		get: () => "rod",
  		});`)

	// to activate the script
	p.MustNavigate("")

	s.Equal("rod", p.MustEval("navigator.rod").String())

	s.Panics(func() {
		s.mc.stubErr(1, proto.PageAddScriptToEvaluateOnNewDocument{})
		p.MustEvalOnNewDocument(`1`)
	})
}

func (s *S) TestPageEval() {
	page := s.page.MustNavigate(srcFile("fixtures/click.html"))

	s.EqualValues(3, page.MustEval(`
		(a, b) => a + b
	`, 1, 2).Int())
	s.EqualValues(1, page.MustEval(`a => 1`).Int())
	s.EqualValues(1, page.MustEval(`function() { return 1 }`).Int())
	s.EqualValues(1, page.MustEval(`((1))`).Int())
	s.NotEqualValues(1, page.MustEval(`a = () => 1`).Int())
	s.NotEqualValues(1, page.MustEval(`a = function() { return 1 }`))
	s.NotEqualValues(1, page.MustEval(`/* ) */`))
}

func (s *S) TestPageEvalNilContext() {
	page := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer page.MustClose()

	s.mc.stub(1, proto.RuntimeEvaluate{}, func(func() ([]byte, error)) ([]byte, error) {
		return nil, &cdp.Error{Code: -32000}
	})
	s.EqualValues(1, page.MustEval(`1`).Int())
}

func (s *S) TestPageExposeJSHelper() {
	page := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer page.MustClose()

	s.Equal("undefined", page.MustEval("typeof(rod)").Str)
	page.ExposeJSHelper()
	s.Equal("object", page.MustEval("typeof(rod)").Str)
}

func (s *S) TestPageWaitOpen() {
	page := s.page.Timeout(3 * time.Second).MustNavigate(srcFile("fixtures/open-page.html"))
	defer page.CancelTimeout()

	wait := page.MustWaitOpen()

	page.MustElement("a").MustClick()

	newPage := wait()
	defer newPage.MustClose()

	s.Equal("new page", newPage.MustEval("window.a").String())
}

func (s *S) TestPageWaitPauseOpen() {
	page := s.page.Timeout(5 * time.Second).MustNavigate(srcFile("fixtures/open-page.html"))
	defer page.CancelTimeout()

	wait, resume := page.MustWaitPauseOpen()

	go page.MustElement("a").MustClick()

	pageA := wait()
	pageA.MustEvalOnNewDocument(`window.a = 'ok'`)
	resume()
	s.Equal("ok", pageA.MustEval(`window.a`).String())
	pageA.MustClose()

	w := page.MustWaitOpen()
	page.MustElement("a").MustClick()
	pageB := w()
	pageB.MustWait(`window.a == 'new page'`)
	pageB.MustClose()

	s.Panics(func() {
		defer func() {
			_ = proto.TargetSetAutoAttach{
				Flatten: true,
			}.Call(s.browser)
		}()

		p := s.browser.MustPage("")
		defer p.MustClose()
		s.mc.stubErr(1, proto.TargetSetAutoAttach{})
		p.MustWaitPauseOpen()
	})
	s.Panics(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		defer func() {
			_ = proto.TargetSetAutoAttach{
				Flatten: true,
			}.Call(s.browser)
		}()

		p := s.browser.Context(ctx).MustPage("")
		defer p.MustClose()
		s.mc.stubErr(2, proto.TargetSetAutoAttach{})
		_, r := p.MustWaitPauseOpen()
		r()
	})
}

func (s *S) TestPageWait() {
	page := s.page.Timeout(5 * time.Second).MustNavigate(srcFile("fixtures/click.html"))
	page.MustWait(`document.querySelector('button') !== null`)

	s.Panics(func() {
		s.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		page.MustWait(``)
	})
}
func (s *S) TestPageWaitNavigation() {
	url, mux, close := utils.Serve("")
	defer close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	wait := s.page.MustWaitNavigation()
	s.page.MustNavigate(url)
	wait()
}

func (s *S) TestPageWaitRequestIdle() {
	url, mux, close := utils.Serve("")
	defer close()

	sleep := time.Second
	timeout, cancel := context.WithTimeout(context.Background(), sleep)
	defer cancel()

	mux.HandleFunc("/r1", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/r2", func(w http.ResponseWriter, r *http.Request) {
		<-timeout.Done()
	})
	mux.HandleFunc("/", httpHTML(`<html>
		<button>click</button>
		<script>
			document.querySelector("button").onclick = () => {
				fetch('/r2').then(r => r.text())
				fetch('/r1')
			}
		</script>
	</html>`))

	page := s.page.MustNavigate(url)

	wait := page.MustWaitRequestIdle("/r1")
	start := time.Now()
	page.MustElement("button").MustClick()
	s.browser.Trace(true)
	wait()
	s.browser.Trace(defaults.Trace)
	s.Greater(time.Since(start), sleep)

	wait = page.MustWaitRequestIdle("/r2")
	page.MustElement("button").MustClick()
	start = time.Now()
	wait()
	s.Less(time.Since(start), sleep)

	s.Panics(func() {
		wait()
	})
}

func (s *S) TestPageWaitIdle() {
	p := s.page.MustNavigate(srcFile("fixtures/click.html"))
	p.MustElement("button").MustClick()
	p.MustWaitIdle()

	s.True(p.MustHas("[a=ok]"))
}

func (s *S) TestPageWaitEvent() {
	wait := s.page.WaitEvent(&proto.PageFrameNavigated{})
	s.page.MustNavigate(srcFile("fixtures/click.html"))
	wait()
}

func (s *S) TestAlert() {
	page := s.page.MustNavigate(srcFile("fixtures/alert.html"))

	go page.MustHandleDialog(true, "")()
	page.MustElement("button").MustClick()
}

func (s *S) TestMouse() {
	page := s.page.MustNavigate(srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse

	s.browser.Trace(true)
	mouse.MustScroll(0, 10)
	s.browser.Trace(defaults.Trace)
	mouse.MustMove(140, 160)
	mouse.MustDown("left")
	mouse.MustUp("left")

	s.True(page.MustHas("[a=ok]"))

	s.Panics(func() {
		s.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustScroll(0, 10)
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustDown(proto.InputMouseButtonLeft)
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustUp(proto.InputMouseButtonLeft)
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustClick(proto.InputMouseButtonLeft)
	})
}

func (s *S) TestMouseClick() {
	s.browser.Slowmotion(1)
	defer func() { s.browser.Slowmotion(0) }()

	page := s.page.MustNavigate(srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse
	mouse.MustMove(140, 160)
	mouse.MustClick("left")
	s.True(page.MustHas("[a=ok]"))
}

func (s *S) TestMouseDrag() {
	page := s.page.MustNavigate(srcFile("fixtures/drag.html")).MustWaitLoad()
	mouse := page.Mouse

	wait := make(chan struct{})
	logs := []string{}
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
		log := page.MustObjectsToJSON(e.Args).Join(" ")
		logs = append(logs, log)
		if strings.HasPrefix(log, `up`) {
			close(wait)
			return true
		}
		return false
	})()

	mouse.MustMove(3, 3)
	mouse.MustDown("left")
	utils.E(mouse.Move(60, 80, 3))
	mouse.MustUp("left")

	<-wait

	s.Equal([]string{"move 3 3", "down 3 3", "move 22 28", "move 41 54", "move 60 80", "up 60 80"}, logs)
}

func (s *S) TestNativeDrag() {
	// devtools doesn't support to use mouse event to simulate it for now
	s.T().SkipNow()

	page := s.page.MustNavigate(srcFile("fixtures/drag.html"))
	mouse := page.Mouse

	box := page.MustElement("#draggable").MustBox()
	x := box.X + 3
	y := box.Y + 3
	toY := page.MustElement(".dropzone:nth-child(2)").MustBox().Y + 3

	page.Overlay(x, y, 10, 10, "from")
	page.Overlay(x, toY, 10, 10, "to")

	mouse.MustMove(x, y)
	mouse.MustDown("left")
	utils.E(mouse.Move(x, toY, 5))
	page.MustScreenshot("")
	mouse.MustUp("left")

	page.MustElement(".dropzone:nth-child(2) #draggable")
}

func (s *S) TestTouch() {
	page := s.browser.MustPage("")
	defer page.MustClose()

	page.MustEmulate(devices.IPad).
		MustNavigate(srcFile("fixtures/touch.html")).
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

	s.Equal([]string{"start 10 20", "end", "start 30 40", "end", "start 30 40", "move 50 60", "cancel"}, logs)

	s.Panics(func() {
		s.mc.stubErr(1, proto.InputDispatchTouchEvent{})
		touch.MustTap(1, 2)
	})
}

func (s *S) TestPageScreenshot() {
	f := filepath.Join("tmp", "screenshots", utils.RandString(8)+".png")
	p := s.page.MustNavigate(srcFile("fixtures/click.html"))
	p.MustElement("button")
	p.MustScreenshot()
	data := p.MustScreenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	utils.E(err)
	s.Equal(800, img.Bounds().Dx())
	s.Equal(600, img.Bounds().Dy())
	s.FileExists(f)

	utils.E(os.RemoveAll(slash("tmp/screenshots")))
	p.MustScreenshot("")

	list, err := ioutil.ReadDir(slash("tmp/screenshots"))
	utils.E(err)
	s.Len(list, 1)

	s.Panics(func() {
		s.mc.stubErr(1, proto.PageCaptureScreenshot{})
		p.MustScreenshot()
	})
}

func (s *S) TestScreenshotFullPage() {
	p := s.page.MustNavigate(srcFile("fixtures/scroll.html"))
	p.MustElement("button")
	data := p.MustScreenshotFullPage()
	img, err := png.Decode(bytes.NewBuffer(data))
	utils.E(err)
	res := p.MustEval(`({w: document.documentElement.scrollWidth, h: document.documentElement.scrollHeight})`)
	s.EqualValues(res.Get("w").Int(), img.Bounds().Dx())
	s.EqualValues(res.Get("h").Int(), img.Bounds().Dy())

	// after the full page screenshot the window size should be the same as before
	res = p.MustEval(`({w: innerWidth, h: innerHeight})`)
	s.EqualValues(800, res.Get("w").Int())
	s.EqualValues(600, res.Get("h").Int())

	utils.E(os.RemoveAll(slash("tmp/screenshots")))
	p.MustScreenshotFullPage("")

	list, err := ioutil.ReadDir(slash("tmp/screenshots"))
	utils.E(err)
	s.Len(list, 1)

	noEmulation := s.browser.MustPage(srcFile("fixtures/click.html"))
	defer noEmulation.MustClose()
	utils.E(noEmulation.SetViewport(nil))
	noEmulation.MustScreenshotFullPage()

	s.Panics(func() {
		s.mc.stubErr(1, proto.PageGetLayoutMetrics{})
		p.MustScreenshotFullPage()
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		p.MustScreenshotFullPage()
	})
}

func (s *S) TestScreenshotFullPageInit() {
	p := s.browser.MustPage(srcFile("fixtures/scroll.html"))
	defer p.MustClose()

	// should not panic
	p.MustScreenshotFullPage()
}

func (s *S) TestPageInput() {
	p := s.page.MustNavigate(srcFile("fixtures/input.html"))

	el := p.MustElement("input")
	el.MustFocus()
	s.browser.Trace(true)
	p.Keyboard.MustPress('A')
	p.Keyboard.MustInsertText(" Test")
	s.browser.Trace(defaults.Trace)
	p.Keyboard.MustPress(input.Tab)

	s.Equal("A Test", el.MustText())

	s.Panics(func() {
		s.mc.stubErr(1, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustDown('a')
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustUp('a')
	})
	s.Panics(func() {
		s.mc.stubErr(3, proto.InputDispatchKeyEvent{})
		p.Keyboard.MustPress('a')
	})
}

func (s *S) TestPageScroll() {
	p := s.page.MustNavigate(srcFile("fixtures/scroll.html")).MustWaitLoad()

	p.Mouse.MustScroll(0, 10)
	p.Mouse.MustScroll(100, 190)
	utils.E(p.Mouse.Scroll(200, 300, 5))
	p.MustElement("button").MustWaitStable()
	offset := p.MustEval("({x: window.pageXOffset, y: window.pageYOffset})")
	s.Less(int64(300), offset.Get("y").Int())
}

func (s *S) TestPageConsoleLog() {
	p := s.page.MustNavigate("")
	e := &proto.RuntimeConsoleAPICalled{}
	wait := p.WaitEvent(e)
	p.MustEval(`console.log(1, {b: ['test']})`)
	wait()
	s.Equal("test", p.MustObjectToJSON(e.Args[1]).Get("b.0").String())
	s.Equal(`1 {"b":["test"]}`, p.MustObjectsToJSON(e.Args).Join(" "))
}

func (s *S) TestPageOthers() {
	p := s.page.MustNavigate(srcFile("fixtures/input.html"))

	s.Equal("body", p.MustElementByJS(`document.body`).MustDescribe().LocalName)
	s.Len(p.MustElementsByJS(`document.querySelectorAll('input')`), 5)
	s.EqualValues(1, p.MustEval(`1`).Int())

	p.Mouse.MustDown("left")
	defer p.Mouse.MustUp("left")
	p.Mouse.MustDown("right")
	defer p.Mouse.MustUp("right")
}

func (s *S) TestFonts() {
	p := s.page.MustNavigate(srcFile("fixtures/fonts.html")).MustWaitLoad()

	p.MustPDF("tmp", "fonts.pdf") // download the file from Github Actions Artifacts
}

func (s *S) TestPagePDF() {
	p := s.page.MustNavigate(srcFile("fixtures/click.html"))
	p.MustPDF("")

	s.Panics(func() {
		s.mc.stubErr(1, proto.PagePrintToPDF{})
		p.MustPDF()
	})
}

func (s *S) TestPageExpose() {
	cb, stop := s.page.MustExpose("exposedFunc")
	page := s.page.MustNavigate(srcFile("fixtures/click.html"))
	page.MustEval(`exposedFunc('ok')`)
	s.Equal("ok", <-cb)
	page.MustEval(`exposedFunc('ok')`)
	stop()
	s.Panics(func() {
		page := s.page.MustNavigate(srcFile("fixtures/click.html"))
		page.MustEval(`exposedFunc('')`)
	})

	s.Panics(func() {
		s.mc.stubErr(1, proto.RuntimeAddBinding{})
		page.MustExpose("exposedFunc")
	})
}

func (s *S) TestPageObjectErr() {
	s.Panics(func() {
		s.page.MustObjectToJSON(&proto.RuntimeRemoteObject{
			ObjectID: "not-exists",
		})
	})
	s.Panics(func() {
		s.page.MustElementFromNode(-1)
	})
	s.Panics(func() {
		id := s.page.MustNavigate(srcFile("fixtures/click.html")).MustElement(`body`).MustNodeID()
		s.mc.stubErr(1, proto.DOMResolveNode{})
		s.page.MustElementFromNode(id)
	})
	s.Panics(func() {
		id := s.page.MustNavigate(srcFile("fixtures/click.html")).MustElement(`body`).MustNodeID()
		s.mc.stubErr(1, proto.DOMDescribeNode{})
		s.page.MustElementFromNode(id)
	})
}

func (s *S) TestPageNavigateErr() {
	// dns error
	s.Panics(func() {
		s.page.MustNavigate("http://" + utils.RandString(8))
	})

	url, mux, close := utils.Serve("")
	defer close()

	mux.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	// will not panic
	s.page.MustNavigate(url + "/404")
	s.page.MustNavigate(url + "/500")

	s.Panics(func() {
		s.mc.stubErr(1, proto.PageStopLoading{})
		s.page.MustNavigate(srcFile("fixtures/click.html"))
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.PageNavigate{})
		s.page.MustNavigate(srcFile("fixtures/click.html"))
	})
}

func (s *S) TestPageWaitLoadErr() {
	s.Panics(func() {
		s.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		s.page.MustWaitLoad()
	})
}

func (s *S) TestPageGoBackGoForward() {
	p := s.browser.MustPage("").MustReload()
	defer p.MustClose()

	p.
		MustNavigate(srcFile("fixtures/click.html")).MustWaitLoad().
		MustNavigate(srcFile("fixtures/selector.html")).MustWaitLoad()

	p.MustNavigateBack().MustWaitLoad()
	s.Regexp("fixtures/click.html$", p.MustInfo().URL)

	p.MustNavigateForward().MustWaitLoad()
	s.Regexp("fixtures/selector.html$", p.MustInfo().URL)
}

func (s *S) TestPageInitJSErr() {
	p := s.browser.MustPage(srcFile("fixtures/click-iframe.html")).MustElement("iframe").MustFrame()
	defer p.MustClose()

	s.Panics(func() {
		s.mc.stubErr(1, proto.PageCreateIsolatedWorld{})
		p.MustEval(`1`)
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.RuntimeEvaluate{})
		p.MustEval(`1`)
	})
	s.Panics(func() {
		s.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		p.MustEval(`1`)
	})
}
