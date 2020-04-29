package rod_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"path/filepath"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/input"
)

func (s *S) TestSetCookies() {
	url, _, close := serve()
	defer close()

	page := s.page.SetCookies(cdp.Object{
		"name":  "a",
		"value": "1",
		"url":   url,
	}, cdp.Object{
		"name":  "b",
		"value": "2",
		"url":   url,
	}).Navigate(url)

	s.Equal("a=1; b=2", page.Eval(`() => document.cookie`).String())
}

func (s *S) TestClosePage() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()
	page.Element("button")
}

func (s *S) TestPageContext() {
	p := s.page.Timeout(time.Minute).CancelTimeout().Cancel()
	_, err := p.CallE(`() => {}`, nil)
	s.Error(err)
}

func (s *S) TestRelease() {
	res, err := s.page.EvalE(false, "", `() => document`, nil)
	kit.E(err)
	s.page.Release(res.Get("result.objectId").String())
}

func (s *S) TestPageCall() {
	s.Greater(s.page.Call("DOM.getDocument", nil).Get("root.nodeId").Int(), int64(0))
}

func (s *S) TestWindow() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()

	bounds := page.GetWindow()
	defer page.Window(
		bounds.Get("left").Int(),
		bounds.Get("top").Int(),
		bounds.Get("width").Int(),
		bounds.Get("height").Int(),
	)

	page.WindowMaximize()
	page.WindowNormal()
	page.WindowFullscreen()
	page.WindowNormal()
	page.WindowMinimize()
	page.WindowNormal()
	page.Window(0, 0, 1211, 611)
	s.EqualValues(1211, page.Eval(`() => window.innerWidth`).Int())
	s.EqualValues(611, page.Eval(`() => window.innerHeight`).Int())
}

func (s *S) TestSetViewport() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()
	page.Viewport(317, 419, 0, false)
	res := page.Eval(`() => [window.innerWidth, window.innerHeight]`)
	s.EqualValues(317, res.Get("0").Int())
	s.EqualValues(419, res.Get("1").Int())

	page2 := s.browser.Page(srcFile("fixtures/click.html"))
	defer page2.Close()
	res = page2.Eval(`() => [window.innerWidth, window.innerHeight]`)
	s.NotEqual(int64(317), res.Get("0").Int())
}

func (s *S) TestPageAddScriptTag() {
	p := s.page.Navigate(srcFile("fixtures/click.html")).WaitLoad()

	res := p.AddScriptTag(srcFile("fixtures/add-script-tag.js")).Eval(`() => count()`)
	s.EqualValues(0, res.Int())

	res = p.AddScriptTag(srcFile("fixtures/add-script-tag.js")).Eval(`() => count()`)
	s.EqualValues(1, res.Int())

	kit.E(p.AddScriptTagE("", `let ok = 'yes'`))
	res = p.Eval(`() => ok`)
	s.Equal("yes", res.String())
}

func (s *S) TestPageAddStyleTag() {
	p := s.page.Navigate(srcFile("fixtures/click.html")).WaitLoad()

	res := p.AddStyleTag(srcFile("fixtures/add-style-tag.css")).
		Element("h4").Eval(`() => getComputedStyle(this).color`)
	s.Equal("rgb(255, 0, 0)", res.String())

	p.AddStyleTag(srcFile("fixtures/add-style-tag.css"))
	s.Len(p.Elements("link"), 1)

	kit.E(p.AddStyleTagE("", "h4 { color: green; }"))
	res = p.Element("h4").Eval(`() => getComputedStyle(this).color`)
	s.Equal("rgb(0, 128, 0)", res.String())
}

func (s *S) TestUntilPage() {
	page := s.page.Timeout(3 * time.Second).Navigate(srcFile("fixtures/open-page.html"))
	defer page.CancelTimeout()

	wait := page.WaitPage()

	page.Element("a").Click()

	newPage := wait()

	s.Equal("click me", newPage.Element("button").Text())

	wait()
}

func (s *S) TestPageWaitRequestIdle() {
	url, engine, close := serve()
	defer close()

	engine.GET("/r1", func(ctx kit.GinContext) {})
	engine.GET("/r2", func(ctx kit.GinContext) { kit.Sleep(1) })
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
	s.True(time.Since(start) > time.Second)

	wait = page.WaitRequestIdle("/r2")
	page.Element("button").Click()
	start = time.Now()
	wait()
	s.True(time.Since(start) < time.Second)
}

func (s *S) TestPageWaitIdle() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	p.Element("button").Click()
	p.WaitIdle()

	s.True(p.Has("[a=ok]"))
}

func (s *S) TestPageWaitEvent() {
	wait := s.page.WaitEvent("Page.frameNavigated")
	s.page.Navigate(srcFile("fixtures/click.html"))
	wait()
}

func (s *S) TestAlert() {
	page := s.page.Navigate(srcFile("fixtures/alert.html"))

	wait := page.HandleDialog(true, "")

	go page.Element("button").Click()

	wait()
}

func (s *S) TestDownloadFile() {
	url, engine, close := serve()
	defer close()

	content := "test content"

	engine.GET("/d", func(ctx kit.GinContext) {
		kit.E(ctx.Writer.Write([]byte(content)))
	})
	engine.GET("/", ginHTML(fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, url)))

	page := s.page.Navigate(url)

	wait := page.GetDownloadFile("*")

	page.Element("a").Click()

	_, data := wait()

	s.Equal(content, string(data))
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

func (s *S) TestDrag() {
	s.T().Skip("not able to use mouse event to simulate it for now")

	page := s.page.Navigate(srcFile("fixtures/drag.html"))
	mouse := page.Mouse

	mouse.Move(60, 30)
	mouse.Down("left")
	kit.E(mouse.MoveE(60, 80, 5))
	mouse.Up("left")

	page.Element(".dropzone:nth-child(2) #draggable")
}

func (s *S) TestPagePause() {
	go s.page.Pause()
	kit.Sleep(0.03)
	go s.page.Eval(`() => 10`)
	kit.Sleep(0.03)
	kit.E(s.page.CallE("Debugger.resume", nil))
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

	kit.E(kit.Remove(filepath.FromSlash("tmp/screenshots")))
	p.Screenshot("")
	s.Len(kit.Walk(filepath.FromSlash("tmp/screenshots/*")).MustList(), 1)
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
	p := s.page.Navigate(srcFile("fixtures/scroll.html")).WaitLoad()
	p.Mouse.Scroll(100, 200)
	kit.E(p.Mouse.ScrollE(200, 300, 5))
	p.Element("button").WaitStable()
	s.EqualValues(300, p.Eval("() => window.pageXOffset").Int())
	s.EqualValues(500, p.Eval("() => window.pageYOffset").Int())
}

func (s *S) TestPageOthers() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))

	s.Equal("body", p.ElementByJS(`() => document.body`).Describe().Get("localName").String())
	s.Len(p.ElementsByJS(`() => document.querySelectorAll('input')`), 3)
	s.EqualValues(1, p.Eval(`() => 1`).Int())

	s.Panics(func() {
		rod.CancelPanic(errors.New("err"))
	})

	s.False(rod.IsError(io.EOF, rod.ErrElementNotFound))

	p.Mouse.Click("")
	p.Mouse.Down("left")
	defer p.Mouse.Up("left")
	p.Mouse.Down("right")
	defer p.Mouse.Up("right")
}

func (s *S) TestFonts() {
	/*
		I don't want to include a large OCR lib just for this test
		So this one should be checked manually:

		docker build -t ysmood/rod lib
		GOOS=linux go test -c -o tmp/rod.test
		docker run --rm -itv $(pwd):/t -w /t ysmood/rod sh
		./tmp/rod.test -test.v -test.run Test/TestFonts
		open tmp/fonts.pdf
	*/

	p := s.page.Navigate(srcFile("fixtures/fonts.html")).WaitLoad()

	kit.E(kit.OutputFile("tmp/fonts.pdf", p.PDF(), nil))
}

func (s *S) TestPageErrors() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Context(ctx).NavigateE("")
	s.Error(err)

	err = p.Context(ctx).WindowE(nil)
	s.Error(err)

	_, err = p.Context(ctx).GetDownloadFileE("", "")
	s.Error(err)

	_, err = p.Context(ctx).ScreenshotE(nil)
	s.Error(err)

	err = p.Context(ctx).PauseE()
	s.Error(err)
}
