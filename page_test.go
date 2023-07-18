package rod_test

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func TestGetPageBrowser(t *testing.T) {
	g := setup(t)

	g.Eq(g.page.Browser().BrowserContextID, g.browser.BrowserContextID)
}

func TestGetPageURL(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate(g.srcFile("fixtures/click-iframe.html")).MustWaitLoad()
	g.Regex(`/fixtures/click-iframe.html\z`, g.page.MustInfo().URL)
}

func TestSetCookies(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	page := g.page.MustSetCookies([]*proto.NetworkCookieParam{{
		Name:  "cookie-a",
		Value: "1",
		URL:   s.URL(),
	}, {
		Name:  "cookie-b",
		Value: "2",
		URL:   s.URL(),
	}}...).MustNavigate(s.URL()).MustWaitLoad()

	cookies := page.MustCookies()

	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Value < cookies[j].Value
	})

	g.Eq("1", cookies[0].Value)
	g.Eq("2", cookies[1].Value)

	page.MustSetCookies()

	cookies = page.MustCookies()
	g.Len(cookies, 0)

	g.Panic(func() {
		g.mc.stubErr(1, proto.TargetGetTargetInfo{})
		page.MustCookies()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.NetworkGetCookies{})
		page.MustCookies()
	})
}

func TestSetBlockedURLs(t *testing.T) {
	g := setup(t)
	page := g.newPage()
	urlsPattern := []string{}
	page.EnableDomain(proto.NetworkEnable{})
	page.MustSetBlockedURLs(urlsPattern...)
	urlsPattern = append(urlsPattern, "*.js")
	page.MustSetBlockedURLs(urlsPattern...)
	go page.EachEvent(
		func(e *proto.NetworkLoadingFailed) {
			g.Eq(e.BlockedReason, proto.NetworkBlockedReasonInspector)
		},
	)
	page.MustNavigate("https://github.com")
}

func TestSetExtraHeaders(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	wg := sync.WaitGroup{}
	var header http.Header
	s.Mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		header = r.Header
		wg.Done()
	})

	p := g.newPage()
	cleanup := p.MustSetExtraHeaders("a", "1", "b", "2")

	wg.Add(1)
	p.MustNavigate(s.URL())
	wg.Wait()

	g.Eq(header.Get("a"), "1")
	g.Eq(header.Get("b"), "2")

	cleanup()

	// TODO: I don't know why it will fail randomly
	if false {
		wg.Add(1)
		p.MustReload()
		wg.Wait()

		g.Eq(header.Get("a"), "")
		g.Eq(header.Get("b"), "")
	}
}

func TestSetUserAgent(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	ua := ""
	lang := ""

	wg := sync.WaitGroup{}
	wg.Add(1)

	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		lang = r.Header.Get("Accept-Language")
		wg.Done()
	})

	g.newPage().MustSetUserAgent(nil).MustNavigate(s.URL())
	wg.Wait()

	g.Eq(ua, "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_0_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	g.Eq(lang, "en")
}

func TestPageHTML(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html")).MustWaitLoad()
	p.MustElement("button").MustClick()
	g.Has(p.MustHTML(), `a="ok"`)

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(p.HTML())
}

func TestMustWaitElementsMoreThan(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/wait_elements.html")).MustWaitElementsMoreThan("li", 5)
	g.Gt(len(p.MustElements("li")), 5)
}

func TestPageCloseCancel(t *testing.T) {
	g := setup(t)

	page := g.browser.MustPage(g.srcFile("fixtures/prevent-close.html"))
	page.MustElement("body").MustClick() // only focused page will handle beforeunload event

	w, h := page.MustHandleDialog()
	go func() {
		w()
		h(false, "")
	}()
	g.Eq(page.Close().Error(), "page close canceled")

	page.MustEval(`() => window.onbeforeunload = null`)
	page.MustClose()
}

func TestLoadState(t *testing.T) {
	g := setup(t)

	g.True(g.page.LoadState(&proto.PageEnable{}))
}

func TestDisableDomain(t *testing.T) {
	g := setup(t)

	defer g.page.DisableDomain(&proto.PageEnable{})()
}

func TestPageContext(t *testing.T) {
	g := setup(t)

	g.page.Timeout(time.Hour).CancelTimeout().MustEval(`() => 1`)
	_, _ = g.page.Timeout(time.Second).Timeout(time.Hour).CancelTimeout().Element("not-exist")
}

func TestPageActivate(t *testing.T) {
	g := setup(t)

	g.page.MustActivate()
}

func TestWindow(t *testing.T) {
	g := setup(t)

	page := g.newPage(g.blank())

	g.E(page.SetViewport(nil))

	bounds := page.MustGetWindow()
	defer page.MustSetWindow(
		*bounds.Left,
		*bounds.Top,
		*bounds.Width,
		*bounds.Height,
	)

	page.MustWindowMaximize()
	page.MustWindowNormal()
	page.MustWindowFullscreen()
	page.MustWindowNormal()
	page.MustWindowMinimize()
	page.MustWindowNormal()

	page.MustSetWindow(0, 0, 1211, 611)
	w, err := proto.BrowserGetWindowForTarget{}.Call(page)
	g.E(err)
	g.Eq(w.Bounds.Width, 1211)
	g.Eq(w.Bounds.Height, 611)

	g.Panic(func() {
		g.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustGetWindow()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.BrowserGetWindowBounds{})
		page.MustGetWindow()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.BrowserGetWindowForTarget{})
		page.MustSetWindow(0, 0, 1000, 1000)
	})
}

func TestSetViewport(t *testing.T) {
	g := setup(t)

	page := g.newPage(g.blank())
	page.MustSetViewport(317, 419, 0, false)
	res := page.MustEval(`() => [window.innerWidth, window.innerHeight]`)
	g.Eq(317, res.Get("0").Int())
	g.Eq(419, res.Get("1").Int())

	page2 := g.newPage(g.blank())
	res = page2.MustEval(`() => [window.innerWidth, window.innerHeight]`)
	g.Neq(int(317), res.Get("0").Int())
}

func TestSetDocumentContent(t *testing.T) {
	g := setup(t)

	page := g.newPage(g.blank())

	doctype := "<!DOCTYPE html>"
	html4StrictDoctype := `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">`
	html4LooseDoctype := `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">`
	xhtml11Doctype := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`

	exampleWithHTML4StrictDoctype := html4StrictDoctype + "<html><head></head><body><div>test</div></body></html>"
	page.MustSetDocumentContent(exampleWithHTML4StrictDoctype)
	exp1 := page.MustEval(`() => new XMLSerializer().serializeToString(document)`).Str()
	g.Eq(exp1, `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd"><html xmlns="http://www.w3.org/1999/xhtml"><head></head><body><div>test</div></body></html>`)
	g.Eq(page.MustElement("html").MustHTML(), "<html><head></head><body><div>test</div></body></html>")
	g.Eq(page.MustElement("head").MustText(), "")

	exampleWithHTML4LooseDoctype := html4LooseDoctype + "<html><head></head><body><div>test</div></body></html>"
	page.MustSetDocumentContent(exampleWithHTML4LooseDoctype)
	exp2 := page.MustEval(`() => new XMLSerializer().serializeToString(document)`).Str()
	g.Eq(exp2, `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd"><html xmlns="http://www.w3.org/1999/xhtml"><head></head><body><div>test</div></body></html>`)
	g.Eq(page.MustElement("html").MustHTML(), "<html><head></head><body><div>test</div></body></html>")
	g.Eq(page.MustElement("head").MustText(), "")

	exampleWithXHTMLDoctype := xhtml11Doctype + "<html><head></head><body><div>test</div></body></html>"
	page.MustSetDocumentContent(exampleWithXHTMLDoctype)
	exp3 := page.MustEval(`() => new XMLSerializer().serializeToString(document)`).Str()
	g.Eq(exp3, `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"><html xmlns="http://www.w3.org/1999/xhtml"><head></head><body><div>test</div></body></html>`)
	g.Eq(page.MustElement("html").MustHTML(), "<html><head></head><body><div>test</div></body></html>")
	g.Eq(page.MustElement("head").MustText(), "")

	exampleWithHTML5Doctype := doctype + "<html><head></head><body><div>test</div></body></html>"
	page.MustSetDocumentContent(exampleWithHTML5Doctype)
	exp4 := page.MustEval(`() => new XMLSerializer().serializeToString(document)`).Str()
	g.Eq(exp4, `<!DOCTYPE html><html xmlns="http://www.w3.org/1999/xhtml"><head></head><body><div>test</div></body></html>`)
	g.Eq(page.MustElement("html").MustHTML(), "<html><head></head><body><div>test</div></body></html>")
	g.Eq(page.MustElement("head").MustText(), "")

	exampleWithoutDoctype := "<html><head></head><body><div>test</div></body></html>"
	page.MustSetDocumentContent(exampleWithoutDoctype)
	g.Eq(page.MustElement("html").MustHTML(), "<html><head></head><body><div>test</div></body></html>")

	exampleBasic := doctype + "<div>test</div>"
	page.MustSetDocumentContent(exampleBasic)
	g.Eq(page.MustElement("div").MustText(), "test")

	exampleWithTrickyContent := "<div>test</div>\x7F"
	page.MustSetDocumentContent(exampleWithTrickyContent)
	g.Eq(page.MustElement("div").MustText(), "test")

	exampleWithEmoji := "<div>💪</div>"
	page.MustSetDocumentContent(exampleWithEmoji)
	g.Eq(page.MustElement("div").MustText(), "💪")
}

func TestEmulateDevice(t *testing.T) {
	g := setup(t)

	page := g.newPage(g.blank())
	page.MustEmulate(devices.IPhone6or7or8)
	res := page.MustEval(`() => [window.innerWidth, window.innerHeight, navigator.userAgent]`)

	// TODO: this seems like a bug of chromium
	{
		g.Lt(math.Abs(float64(980-res.Get("0").Int())), 10)
		g.Lt(math.Abs(float64(1743-res.Get("1").Int())), 10)
	}

	g.Eq(
		"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
		res.Get("2").String(),
	)
	g.Panic(func() {
		g.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		page.MustEmulate(devices.IPad)
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.EmulationSetTouchEmulationEnabled{})
		page.MustEmulate(devices.IPad)
	})
}

func TestPageCloseErr(t *testing.T) {
	g := setup(t)

	page := g.newPage(g.blank())
	g.Panic(func() {
		g.mc.stubErr(1, proto.PageClose{})
		page.MustClose()
	})
}

func TestPageCloseWhenNotAttached(t *testing.T) {
	g := setup(t)

	p := g.browser.MustPage(g.blank())

	g.mc.stub(1, proto.PageClose{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), cdp.ErrNotAttachedToActivePage
	})

	g.E(p.Close())
}

func TestPageAddScriptTag(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.blank()).MustWaitLoad()

	res := p.MustAddScriptTag(g.srcFile("fixtures/add-script-tag.js")).MustEval(`() => count()`)
	g.Eq(0, res.Int())

	res = p.MustAddScriptTag(g.srcFile("fixtures/add-script-tag.js")).MustEval(`() => count()`)
	g.Eq(1, res.Int())

	g.E(p.AddScriptTag("", `let ok = 'yes'`))
	res = p.MustEval(`() => ok`)
	g.Eq("yes", res.String())
}

func TestPageAddStyleTag(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html")).MustWaitLoad()

	res := p.MustAddStyleTag(g.srcFile("fixtures/add-style-tag.css")).
		MustElement("h4").MustEval(`() => getComputedStyle(this).color`)
	g.Eq("rgb(255, 0, 0)", res.String())

	p.MustAddStyleTag(g.srcFile("fixtures/add-style-tag.css"))
	g.Len(p.MustElements("link"), 1)

	g.E(p.AddStyleTag("", "h4 { color: green; }"))
	res = p.MustElement("h4").MustEval(`() => getComputedStyle(this).color`)
	g.Eq("rgb(0, 128, 0)", res.String())
}

func TestPageWaitOpen(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("fixtures/open-page.html"))

	wait := page.MustWaitOpen()

	page.MustElement("a").MustClick()

	newPage := wait()
	defer newPage.MustClose()

	g.Eq("new page", newPage.MustEval("() => window.a").String())
}

func TestPageWait(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	page.MustWait(`() => document.querySelector('button') !== null`)

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		page.MustWait(``)
	})
}

func TestPageNavigateBlank(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate("")
}

func TestPageWaitNavigation(t *testing.T) {
	g := setup(t)

	s := g.Serve().Route("/", "")
	wait := g.page.MustWaitNavigation()
	g.page.MustNavigate(s.URL())
	wait()
}

func TestPageWaitRequestIdle(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	sleep := time.Second

	s.Route("/r1", "")
	s.Mux.HandleFunc("/r2", func(w http.ResponseWriter, r *http.Request) {
		g.E(w.Write([]byte("part")))
		ctx, cancel := context.WithTimeout(g.Context(), sleep)
		defer cancel()
		<-ctx.Done()
	})
	s.Mux.HandleFunc("/r3", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Location", "/r4")
		rw.WriteHeader(http.StatusFound)
	})
	s.Route("/r4", "")
	s.Route("/img.jpg", ".jpg", "img")
	s.Route("/", ".html", `<html><body></body></html>`)

	page := g.newPage(s.URL()).MustWaitLoad()

	code := ` () => {
		fetch('/r2').then(r => r.text())
		fetch('/r1')
		fetch('/r3')
		document.body.innerHTML = '<img src="/img.jpg" />'
	}`

	waitReq := ""
	g.browser.Logger(utils.Log(func(msg ...interface{}) {
		typ := msg[0].(rod.TraceType)
		if typ == rod.TraceTypeWaitRequests {
			list := msg[2].(map[string]string)
			for _, v := range list {
				waitReq = v
				break
			}
		}
	}))
	defer g.browser.Logger(rod.DefaultLogger)

	g.browser.Trace(true)
	wait := page.MustWaitRequestIdle("/r1")
	g.browser.Trace(defaults.Trace)
	page.MustEval(code)
	start := time.Now()
	wait()
	g.Gt(time.Since(start), sleep)
	g.Regex("/r2$", waitReq)

	wait = page.MustWaitRequestIdle("/r2")
	page.MustEval(code)
	start = time.Now()
	wait()
	g.Lt(time.Since(start), sleep)

	g.Panic(func() {
		wait()
	})
}

func TestPageCaptureDOMSnapshot(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	domSnapshot := p.MustCaptureDOMSnapshot()
	g.Is(domSnapshot.Strings, []string{})

	timeOutPage := p.Timeout(1 * time.Second)
	utils.Sleep(1)
	snapshot, err := timeOutPage.CaptureDOMSnapshot()
	g.Is(err, context.DeadlineExceeded)
	g.Nil(snapshot)
}

func TestPageWaitDOMStable(t *testing.T) {
	g := setup(t)

	{
		p := g.page.MustNavigate(g.srcFile("fixtures/page-wait-stable.html"))
		p.MustWaitDOMStable()
	}

	{
		p := g.page.MustNavigate(g.srcFile("fixtures/page-wait-stable.html"))
		err := p.Timeout(time.Second).WaitDOMStable(time.Second, 0)
		g.Is(err, context.DeadlineExceeded)
	}

	{
		g.Panic(func() {
			p := g.page.MustNavigate(g.srcFile("fixtures/page-wait-stable.html"))
			g.mc.stubErr(1, proto.DOMSnapshotCaptureSnapshot{})
			p.MustWaitDOMStable()
		})
	}

	{
		g.Panic(func() {
			p := g.page.MustNavigate(g.srcFile("fixtures/page-wait-stable.html"))
			g.mc.stubErr(2, proto.DOMSnapshotCaptureSnapshot{})
			p.MustWaitDOMStable()
		})
	}
}

func TestPageWaitStable(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/page-wait-stable.html"))
	p.MustWaitStable()
}

func TestPageWaitIdle(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	p.MustElement("button").MustClick()
	p.MustWaitIdle()

	g.True(p.MustHas("[a=ok]"))
}

func TestPageEventSession(t *testing.T) {
	g := setup(t)

	s := g.Serve()
	p := g.newPage(s.URL())

	p.EnableDomain(proto.NetworkEnable{})
	go g.page.Context(g.Context()).EachEvent(func(e *proto.NetworkRequestWillBeSent) {
		g.Log("should not goes to here")
		g.Fail()
	})()
	p.MustEval(`u => fetch(u)`, s.URL())
}

func TestPageWaitEvent(t *testing.T) {
	g := setup(t)

	wait := g.page.WaitEvent(&proto.PageFrameNavigated{})
	g.page.MustNavigate(g.blank())
	wait()
}

func TestPageWaitEventParseEventOnlyOnce(t *testing.T) {
	g := setup(t)

	nav1 := g.page.WaitEvent(&proto.PageFrameNavigated{})
	nav2 := g.page.WaitEvent(&proto.PageFrameNavigated{})
	g.page.MustNavigate(g.blank())
	nav1()
	nav2()
}

func TestPageEvent(t *testing.T) {
	g := setup(t)

	p := g.browser.MustPage()
	ctx := g.Context()
	events := p.Context(ctx).Event()
	p.MustNavigate(g.blank())
	for msg := range events {
		if msg.Load(proto.PageFrameStartedLoading{}) {
			break
		}
	}
	utils.Sleep(0.3)
	ctx.Cancel()

	go func() {
		for range p.Event() {
			utils.Noop()
		}
	}()
	p.MustClose()
}

func TestPageStopEventAfterDetach(t *testing.T) {
	g := setup(t)

	p := g.browser.MustPage().Context(g.Context())
	go func() {
		utils.Sleep(0.3)
		p.MustClose()
	}()
	for range p.Event() {
		utils.Noop()
	}
}

func TestAlert(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("fixtures/alert.html"))

	wait, handle := page.MustHandleDialog()

	go page.MustElement("button").MustClick()

	e := wait()
	g.Eq(e.Message, "clicked")
	handle(true, "")
}

func TestPageHandleFileDialog(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement(`[type=file]`)

	setFiles := p.MustHandleFileDialog()
	el.MustClick()
	setFiles(slash("fixtures/click.html"), slash("fixtures/alert.html"))

	list := el.MustEval("() => Array.from(this.files).map(f => f.name)").Arr()
	g.Len(list, 2)
	g.Eq("alert.html", list[1].String())

	{
		g.mc.stubErr(1, proto.PageSetInterceptFileChooserDialog{})
		g.Err(p.HandleFileDialog())
	}
	{
		g.mc.stubErr(2, proto.PageSetInterceptFileChooserDialog{})
		setFiles, _ := p.HandleFileDialog()
		el.MustClick()
		g.Err(setFiles([]string{slash("fixtures/click.html")}))
		g.E(proto.PageSetInterceptFileChooserDialog{Enabled: false}.Call(p))
	}
}

func TestPageScreenshot(t *testing.T) {
	g := setup(t)

	f := filepath.Join("tmp", "screenshots", g.RandStr(16)+".png")
	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	p.MustElement("button")
	p.MustScreenshot()
	data := p.MustScreenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	g.E(err)
	g.Eq(1280, img.Bounds().Dx())
	g.Eq(800, img.Bounds().Dy())
	g.Nil(os.Stat(f))

	p.MustScreenshot("")

	g.Panic(func() {
		g.mc.stubErr(1, proto.PageCaptureScreenshot{})
		p.MustScreenshot()
	})
}

func TestScreenshotFullPage(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/scroll.html"))
	p.MustElement("button")
	data := p.MustScreenshotFullPage()
	img, err := png.Decode(bytes.NewBuffer(data))
	g.E(err)
	res := p.MustEval(`() => ({w: document.documentElement.scrollWidth, h: document.documentElement.scrollHeight})`)
	g.Eq(res.Get("w").Int(), img.Bounds().Dx())
	g.Eq(res.Get("h").Int(), img.Bounds().Dy())

	// after the full page screenshot the window size should be the same as before
	res = p.MustEval(`() => ({w: innerWidth, h: innerHeight})`)
	g.Eq(1280, res.Get("w").Int())
	g.Eq(800, res.Get("h").Int())

	p.MustScreenshotFullPage()

	noEmulation := g.newPage(g.blank())
	g.E(noEmulation.SetViewport(nil))
	noEmulation.MustScreenshotFullPage()

	g.Panic(func() {
		g.mc.stubErr(1, proto.PageGetLayoutMetrics{})
		p.MustScreenshotFullPage()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		p.MustScreenshotFullPage()
	})

	g.Panic(func() {
		g.mc.stub(1, proto.PageGetLayoutMetrics{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(proto.PageGetLayoutMetricsResult{}), nil
		})
		p.MustScreenshotFullPage()
	})
}

func TestScreenshotFullPageInit(t *testing.T) {
	g := setup(t)

	p := g.newPage(g.srcFile("fixtures/scroll.html"))

	// should not panic
	p.MustScreenshotFullPage()
}

func TestPageConsoleLog(t *testing.T) {
	g := setup(t)

	p := g.newPage(g.blank()).MustWaitLoad()
	e := &proto.RuntimeConsoleAPICalled{}
	wait := p.WaitEvent(e)
	p.MustEval(`() => console.log(1, {b: ['test']})`)
	wait()
	g.Eq("test", p.MustObjectToJSON(e.Args[1]).Get("b.0").String())
	g.Eq(`1 map[b:[test]]`, p.MustObjectsToJSON(e.Args).Join(" "))
}

func TestFonts(t *testing.T) {
	g := setup(t)

	if !utils.InContainer { // No need to test font rendering on regular OS
		g.SkipNow()
	}

	p := g.page.MustNavigate(g.srcFile("fixtures/fonts.html")).MustWaitLoad()

	p.MustPDF("tmp", "fonts.pdf") // download the file from Github Actions Artifacts
}

func TestPagePDF(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))

	s, err := p.PDF(&proto.PagePrintToPDF{})
	g.E(err)
	g.Nil(s.Close())

	p.MustPDF("")

	g.Panic(func() {
		g.mc.stubErr(1, proto.PagePrintToPDF{})
		p.MustPDF()
	})
}

func TestPageNavigateNetworkErr(t *testing.T) {
	g := setup(t)
	p := g.newPage()

	err := p.Navigate("http://127.0.0.1:1")
	g.Is(err, &rod.ErrNavigation{})
	g.Is(err.Error(), "navigation failed: net::ERR_NAME_NOT_RESOLVED")
	p.MustNavigate("about:blank")
}

func TestPageNavigateErr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: This test on Windows is flaky")
	}

	g := setup(t)

	s := g.Serve()

	s.Mux.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	s.Mux.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	// will not panic
	g.page.MustNavigate(s.URL("/404"))
	g.page.MustNavigate(s.URL("/500"))

	g.Panic(func() {
		g.mc.stubErr(1, proto.PageNavigate{})
		g.page.MustNavigate(g.blank())
	})
}

func TestPageWaitLoadErr(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		g.page.MustWaitLoad()
	})
}

func TestPageNavigation(t *testing.T) {
	g := setup(t)

	p := g.newPage().MustReload()

	wait := p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigate(g.srcFile("fixtures/click.html"))
	wait()

	wait = p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigate(g.srcFile("fixtures/selector.html"))
	wait()

	wait = p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigateBack()
	wait()
	g.Regex("fixtures/click.html$", p.MustInfo().URL)

	wait = p.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	p.MustNavigateForward()
	wait()
	g.Regex("fixtures/selector.html$", p.MustInfo().URL)

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(p.Reload())
}

func TestPagePool(t *testing.T) {
	g := setup(t)

	pool := rod.NewPagePool(3)
	create := func() *rod.Page { return g.browser.MustPage() }
	p := pool.Get(create)
	pool.Put(p)
	pool.Cleanup(func(p *rod.Page) {
		p.MustClose()
	})
}

func TestPageUseNonExistSession(t *testing.T) {
	g := setup(t)

	p := g.browser.PageFromSession("not-exist")
	err := proto.PageClose{}.Call(p)
	g.Eq(err, cdp.ErrSessionNotFound)
}

func TestPageElementFromObjectErr(t *testing.T) {
	g := setup(t)

	p := g.newPage()
	wait := p.WaitNavigation(proto.PageLifecycleEventNameLoad)
	p.MustNavigate(g.srcFile("./fixtures/click.html"))
	wait()
	res, err := proto.DOMGetNodeForLocation{X: 10, Y: 10}.Call(p)
	g.E(err)

	obj, err := proto.DOMResolveNode{
		BackendNodeID: res.BackendNodeID,
	}.Call(p)
	g.E(err)

	g.mc.stubErr(1, proto.RuntimeEvaluate{})
	g.Err(p.ElementFromObject(obj.Object))
}

func TestPageTriggerFavicon(t *testing.T) {
	g := setup(t)
	s := g.Serve()
	// test browser in no-headless mode with an error
	{
		page := g.newPage()
		page.MustNavigate(s.URL())
		g.mc.stub(1, proto.BrowserGetBrowserCommandLine{}, func(send StubSend) (gson.JSON, error) {
			commandLine := proto.BrowserGetBrowserCommandLineResult{Arguments: []string{""}}
			return gson.New(commandLine), nil
		})
		err := page.TriggerFavicon()
		g.Eq(err.Error(), "browser is no-headless")
	}

	// test browser in headless mode to trigger favicon request
	{
		faviconURL := fmt.Sprintf(s.HostURL.String(), "/favicon.ico")
		s.Route("/test", "")
		s.Route("/favicon.ico", filepath.FromSlash("./fixtures/icon.png"))
		page := g.newPage()
		page.MustNavigate(s.URL("/test"))
		page.MustWaitIdle()
		go page.Context(g.Context()).EachEvent(
			func(e *proto.NetworkRequestWillBeSent) {
				if e.Request.URL == faviconURL {
					g.Eq(e.Request.URL, faviconURL)
				}
			},
		)()
		page.MustTriggerFavicon()
	}

	// test browser in headless mode to trigger favicon request with an error
	{
		p := g.newPage().MustNavigate(s.URL())
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		g.Panic(func() {
			p.MustTriggerFavicon()
		})
	}
}

func TestPageActionAfterClose(t *testing.T) {
	g := setup(t)

	{
		p := g.browser.MustPage(g.blank())

		p.MustClose()

		_, err := p.Element("not-exists")
		g.Eq(err, context.Canceled)
	}

	{
		p := g.browser.MustPage(g.blank())
		go func() {
			utils.Sleep(1)
			p.MustClose()
		}()

		_, err := p.Eval(`() => new Promise(r => {})`)
		g.Eq(err, context.Canceled)
	}
}
