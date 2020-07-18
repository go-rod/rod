package rod_test

import (
	"bytes"
	"context"
	"image/png"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/kit"
)

func (s *S) TestGetPageURL() {
	s.page.Navigate(srcFile("fixtures/click-iframe.html")).WaitLoad()
	s.Regexp(`/fixtures/click-iframe.html\z`, s.page.Info().URL)
}

func (s *S) TestSetCookies() {
	url, _, close := serve()
	defer close()

	page := s.page.SetCookies(&proto.NetworkCookieParam{
		Name:  "a",
		Value: "1",
		URL:   url,
	}, &proto.NetworkCookieParam{
		Name:  "b",
		Value: "2",
		URL:   url,
	}).Navigate(url)

	cookies := page.Cookies()

	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Value < cookies[j].Value
	})

	s.Equal("1", cookies[0].Value)
	s.Equal("2", cookies[1].Value)
}

func (s *S) TestSetExtraHeaders() {
	url, engine, close := serve()
	defer close()

	key1 := kit.RandString(8)
	key2 := kit.RandString(8)

	wg := sync.WaitGroup{}
	wg.Add(1)

	var out1, out2 string
	engine.NoRoute(func(ctx kit.GinContext) {
		out1 = ctx.GetHeader(key1)
		out2 = ctx.GetHeader(key2)
		wg.Done()
	})

	defer s.page.SetExtraHeaders(key1, "1", key2, "2")()
	s.page.Navigate(url)
	wg.Wait()

	s.Equal("1", out1)
	s.Equal("2", out2)
}

func (s *S) TestSetUserAgent() {
	url, engine, close := serve()
	defer close()

	ua := ""
	lang := ""

	wg := sync.WaitGroup{}
	wg.Add(1)

	engine.NoRoute(func(ctx kit.GinContext) {
		ua = ctx.GetHeader("User-Agent")
		lang = ctx.GetHeader("Accept-Language")
		wg.Done()
	})

	p := s.browser.Page("").SetUserAgent(nil).Navigate(url)
	defer p.Close()
	wg.Wait()

	s.Equal("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36", ua)
	s.Equal("en", lang)
}

func (s *S) TestClosePage() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()
	page.Element("button")
}

func (s *S) TestLoadState() {
	s.True(s.page.LoadState(&proto.PageEnable{}))
}

func (s *S) TestPageContext() {
	p := s.page.Timeout(time.Minute).CancelTimeout()
	s.Panics(func() { p.Eval(`() => {}`) })
}

func (s *S) TestRelease() {
	res, err := s.page.EvalE(false, "", `document`, nil)
	kit.E(err)
	s.page.Release(res.ObjectID)
}

func (s *S) TestWindow() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()

	bounds := page.GetWindow()
	defer page.Window(
		bounds.Left,
		bounds.Top,
		bounds.Width,
		bounds.Height,
	)

	page.WindowMaximize()
	page.WindowNormal()
	page.WindowFullscreen()
	page.WindowNormal()
	page.WindowMinimize()
	page.WindowNormal()
	page.Window(0, 0, 1211, 611)
	s.EqualValues(1211, page.Eval(`window.innerWidth`).Int())
	s.EqualValues(611, page.Eval(`window.innerHeight`).Int())
}

func (s *S) TestSetViewport() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()
	page.Viewport(317, 419, 0, false)
	res := page.Eval(`[window.innerWidth, window.innerHeight]`)
	s.EqualValues(317, res.Get("0").Int())
	s.EqualValues(419, res.Get("1").Int())

	page2 := s.browser.Page(srcFile("fixtures/click.html"))
	defer page2.Close()
	res = page2.Eval(`[window.innerWidth, window.innerHeight]`)
	s.NotEqual(int64(317), res.Get("0").Int())
}

func (s *S) TestEmulateDevice() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()
	page.Emulate(devices.IPhone6or7or8Plus)
	res := page.Eval(`[window.innerWidth, window.innerHeight, navigator.userAgent]`)
	s.EqualValues(980, res.Get("0").Int())
	s.EqualValues(1743, res.Get("1").Int())
	s.Equal(
		"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
		res.Get("2").String(),
	)
}

func (s *S) TestPageAddScriptTag() {
	p := s.page.Navigate(srcFile("fixtures/click.html")).WaitLoad()

	res := p.AddScriptTag(srcFile("fixtures/add-script-tag.js")).Eval(`count()`)
	s.EqualValues(0, res.Int())

	res = p.AddScriptTag(srcFile("fixtures/add-script-tag.js")).Eval(`count()`)
	s.EqualValues(1, res.Int())

	kit.E(p.AddScriptTagE("", `let ok = 'yes'`))
	res = p.Eval(`ok`)
	s.Equal("yes", res.String())
}

func (s *S) TestPageAddStyleTag() {
	p := s.page.Navigate(srcFile("fixtures/click.html")).WaitLoad()

	res := p.AddStyleTag(srcFile("fixtures/add-style-tag.css")).
		Element("h4").Eval(`getComputedStyle(this).color`)
	s.Equal("rgb(255, 0, 0)", res.String())

	p.AddStyleTag(srcFile("fixtures/add-style-tag.css"))
	s.Len(p.Elements("link"), 1)

	kit.E(p.AddStyleTagE("", "h4 { color: green; }"))
	res = p.Element("h4").Eval(`getComputedStyle(this).color`)
	s.Equal("rgb(0, 128, 0)", res.String())
}

func (s *S) TestPageEvalOnNewDocument() {
	p := s.browser.Page("")
	defer p.Close()

	p.EvalOnNewDocument(`
  		Object.defineProperty(navigator, 'rod', {
    		get: () => "rod",
  		});`)

	// to activate the script
	p.Navigate("")

	s.Equal("rod", p.Eval("navigator.rod").String())
}

func (s *S) TestPageEval() {
	page := s.page.Navigate(srcFile("fixtures/click.html"))

	s.EqualValues(1, page.Eval(`
		() => 1
	`).Int())
	s.EqualValues(1, page.Eval(`a => 1`).Int())
	s.EqualValues(1, page.Eval(`function() { return 1 }`).Int())
	s.NotEqualValues(1, page.Eval(`a = () => 1`).Int())
	s.NotEqualValues(1, page.Eval(`a = function() { return 1 }`))
}

func (s *S) TestPageExposeJSHelper() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()

	s.Equal("undefined", page.Eval("typeof(rod)").Str)
	page.ExposeJSHelper()
	s.Equal("object", page.Eval("typeof(rod)").Str)
}

func (s *S) TestUntilPage() {
	page := s.page.Timeout(3 * time.Second).Navigate(srcFile("fixtures/open-page.html"))
	defer page.CancelTimeout()

	wait := page.WaitOpen()

	page.Element("a").Click()

	newPage := wait()

	s.Equal("click me", newPage.Element("button").Text())
}

func (s *S) TestPageWait() {
	page := s.page.Timeout(3 * time.Second).Navigate(srcFile("fixtures/click.html"))
	page.Wait(`document.querySelector('button') !== null`)
}

func (s *S) TestPageWaitRequestIdle() {
	url, engine, close := serve()
	defer close()

	sleep := time.Second

	engine.GET("/r1", func(ctx kit.GinContext) {})
	engine.GET("/r2", func(ctx kit.GinContext) { time.Sleep(sleep) })
	engine.GET("/", ginHTML(`<html>
		<button>click</button>
		<script>
			document.querySelector("button").onclick = () => {
				fetch('/r1')
				fetch('/r2').then(r => r.text())
			}
		</script>
	</html>`))

	page := s.page.Navigate(url)

	wait := page.WaitRequestIdle("/r1")
	page.Element("button").Click()
	start := time.Now()
	wait()
	s.Greater(int64(time.Since(start)), int64(sleep))

	wait = page.WaitRequestIdle("/r2")
	page.Element("button").Click()
	start = time.Now()
	wait()
	s.Less(int64(time.Since(start)), int64(sleep))

	s.Panics(func() {
		wait()
	})
}

func (s *S) TestPageWaitIdle() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	p.Element("button").Click()
	p.WaitIdle()

	s.True(p.Has("[a=ok]"))
}

func (s *S) TestPageWaitEvent() {
	wait := s.page.WaitEvent(&proto.PageFrameNavigated{})
	s.page.Navigate(srcFile("fixtures/click.html"))
	wait()
}

func (s *S) TestAlert() {
	page := s.page.Navigate(srcFile("fixtures/alert.html"))

	wait := page.HandleDialog(true, "")

	go page.Element("button").Click()

	wait()
}

func (s *S) TestMouse() {
	page := s.page.Navigate(srcFile("fixtures/click.html"))
	page.Element("button")
	mouse := page.Mouse

	mouse.Move(140, 160)
	mouse.Down("left")
	mouse.Up("left")

	s.True(page.Has("[a=ok]"))
}

func (s *S) TestMouseClick() {
	s.browser.Slowmotion(1)
	defer func() { s.browser.Slowmotion(0) }()

	page := s.page.Navigate(srcFile("fixtures/click.html"))
	page.Element("button")
	mouse := page.Mouse
	mouse.Move(140, 160)
	mouse.Click("left")
	s.True(page.Has("[a=ok]"))
}

func (s *S) TestMouseDrag() {
	page := s.page.Navigate(srcFile("fixtures/drag.html")).WaitLoad()
	mouse := page.Mouse

	wait := make(chan kit.Nil)
	logs := []string{}
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
		log := page.ObjectsToJSON(e.Args).Join(" ")
		logs = append(logs, log)
		if strings.HasPrefix(log, `up`) {
			close(wait)
			return true
		}
		return false
	})()

	mouse.Move(3, 3)
	mouse.Down("left")
	kit.E(mouse.MoveE(60, 80, 3))
	mouse.Up("left")

	<-wait

	s.Equal([]string{"move 3 3", "down 3 3", "move 22 28", "move 41 54", "move 60 80", "up 60 80"}, logs)
}

func (s *S) TestNativeDrag() {
	// devtools doesn't support to use mouse event to simulate it for now
	s.T().SkipNow()

	page := s.page.Navigate(srcFile("fixtures/drag.html"))
	mouse := page.Mouse

	box := page.Element("#draggable").Box()
	x := box.X + 3
	y := box.Y + 3
	toY := page.Element(".dropzone:nth-child(2)").Box().Y + 3

	page.Overlay(x, y, 10, 10, "from")
	page.Overlay(x, toY, 10, 10, "to")

	mouse.Move(x, y)
	mouse.Down("left")
	kit.E(mouse.MoveE(x, toY, 5))
	page.Screenshot("")
	mouse.Up("left")

	page.Element(".dropzone:nth-child(2) #draggable")
}

func (s *S) TestPagePause() {
	go s.page.Pause()
	kit.Sleep(0.03)
	go s.page.Eval(`10`)
	kit.Sleep(0.03)
	kit.E(proto.DebuggerResume{}.Call(s.page))
}

func (s *S) TestPageScreenshot() {
	f := filepath.Join("tmp", kit.RandString(8)+".png")
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	p.Element("button")
	p.Screenshot()
	data := p.Screenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	kit.E(err)
	s.Equal(800, img.Bounds().Dx())
	s.Equal(600, img.Bounds().Dy())
	s.FileExists(f)

	kit.E(kit.Remove(slash("tmp/screenshots")))
	p.Screenshot("")
	s.Len(kit.Walk(slash("tmp/screenshots/*")).MustList(), 1)
}

func (s *S) TestScreenshotFullPage() {
	p := s.page.Navigate(srcFile("fixtures/scroll.html"))
	p.Element("button")
	data := p.ScreenshotFullPage()
	img, err := png.Decode(bytes.NewBuffer(data))
	kit.E(err)
	res := p.Eval(`({w: document.documentElement.scrollWidth, h: document.documentElement.scrollHeight})`)
	s.EqualValues(res.Get("w").Int(), img.Bounds().Dx())
	s.EqualValues(res.Get("h").Int(), img.Bounds().Dy())

	// after the full page screenshot the window size should be the same as before
	res = p.Eval(`({w: innerWidth, h: innerHeight})`)
	s.EqualValues(800, res.Get("w").Int())
	s.EqualValues(600, res.Get("h").Int())

	kit.E(kit.Remove(slash("tmp/screenshots")))
	p.ScreenshotFullPage("")
	s.Len(kit.Walk(slash("tmp/screenshots/*")).MustList(), 1)
}

func (s *S) TestScreenshotFullPageInit() {
	p := s.browser.Page(srcFile("fixtures/scroll.html"))
	defer p.Close()

	// should not panic
	p.ScreenshotFullPage()
}

func (s *S) TestPageInput() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))

	el := p.Element("input")
	el.Focus()
	p.Keyboard.Press('A')
	p.Keyboard.InsertText(" Test")
	p.Keyboard.Press(input.Tab)

	s.Equal("A Test", el.Text())
}

func (s *S) TestPageScroll() {
	kit.E(kit.Retry(context.Background(), kit.CountSleeper(10), func() (bool, error) {
		p := s.browser.Page(srcFile("fixtures/scroll.html")).WaitLoad()
		defer p.Close()

		p.Mouse.Scroll(0, 10)
		p.Mouse.Scroll(100, 190)
		kit.E(p.Mouse.ScrollE(200, 300, 5))
		p.Element("button").WaitStable()
		offset := p.Eval("({x: window.pageXOffset, y: window.pageYOffset})")
		if offset.Get("x").Int() == 300 {
			s.GreaterOrEqual(int64(10), 500-offset.Get("y").Int())
			return true, nil
		}
		return false, nil
	}))
}

func (s *S) TestPageConsoleLog() {
	p := s.page.Navigate("")
	e := &proto.RuntimeConsoleAPICalled{}
	wait := p.WaitEvent(e)
	p.Eval(`console.log(1, {b: ['test']})`)
	wait()
	s.Equal("test", p.ObjectToJSON(e.Args[1]).Get("b.0").String())
	s.Equal(`1 {"b":["test"]}`, p.ObjectsToJSON(e.Args).Join(" "))
}

func (s *S) TestPageOthers() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))

	s.Equal("body", p.ElementByJS(`document.body`).Describe().LocalName)
	s.Len(p.ElementsByJS(`document.querySelectorAll('input')`), 4)
	s.EqualValues(1, p.Eval(`1`).Int())

	p.Mouse.Down("left")
	defer p.Mouse.Up("left")
	p.Mouse.Down("right")
	defer p.Mouse.Up("right")
}

func (s *S) TestFonts() {
	/*
		I don't want to include a large OCR lib just for this test
		So this one should be checked manually:

		GOOS=linux go test -c -o tmp/rod.test
		docker run --rm -itv $(pwd):/t -w /t rodorg/rod sh
		./tmp/rod.test -test.v -test.run Test/TestFonts
		open tmp/fonts.pdf
	*/

	p := s.page.Navigate(srcFile("fixtures/fonts.html")).WaitLoad()

	kit.E(kit.OutputFile("tmp/fonts.pdf", p.PDF(), nil))
}

func (s *S) TestNavigateErr() {
	// dns error
	s.Panics(func() {
		s.page.Navigate("http://" + kit.RandString(8))
	})

	url, engine, close := serve()
	defer close()

	engine.GET("/404", func(ctx kit.GinContext) {
		ctx.Writer.WriteHeader(404)
	})
	engine.GET("/500", func(ctx kit.GinContext) {
		ctx.Writer.WriteHeader(500)
	})

	// will not panic
	s.page.Navigate(url + "/404")
	s.page.Navigate(url + "/500")
}
