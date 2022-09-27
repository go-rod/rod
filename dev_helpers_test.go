package rod_test

import (
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func TestMonitor(t *testing.T) {
	g := setup(t)

	b := rod.New().MustConnect()
	defer b.MustClose()
	p := b.MustPage(g.blank()).MustWaitLoad()

	b, cancel := b.WithCancel()
	defer cancel()
	host := b.Context(g.Context()).ServeMonitor("")

	page := g.page.MustNavigate(host)
	g.Has(page.MustElement("#targets a").MustParent().MustHTML(), string(p.TargetID))

	page.MustNavigate(host + "/page/" + string(p.TargetID))
	page.MustWait(`(id) => document.title.includes(id)`, p.TargetID)

	img := g.Req("", host+"/screenshot").Bytes()
	g.Gt(img.Len(), 10)

	res := g.Req("", host+"/api/page/test")
	g.Eq(400, res.StatusCode)
	g.Eq(-32602, gson.New(res.Body).Get("code").Int())
}

func TestMonitorErr(t *testing.T) {
	g := setup(t)

	l := launcher.New()
	u := l.MustLaunch()
	defer l.Kill()

	g.Panic(func() {
		rod.New().Monitor("abc").ControlURL(u).MustConnect()
	})
}

func TestTrace(t *testing.T) {
	g := setup(t)

	g.Eq(rod.TraceTypeInput.String(), "[input]")

	var msg []interface{}
	g.browser.Logger(utils.Log(func(list ...interface{}) { msg = list }))
	g.browser.Trace(true).SlowMotion(time.Microsecond)
	defer func() {
		g.browser.Logger(rod.DefaultLogger)
		g.browser.Trace(defaults.Trace).SlowMotion(defaults.Slow)
	}()

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html")).MustWaitLoad()

	g.Eq(rod.TraceTypeWait, msg[0])
	g.Eq("load", msg[1])
	g.Eq(p, msg[2])

	el := p.MustElement("button")
	el.MustClick()

	g.Eq(rod.TraceTypeInput, msg[0])
	g.Eq("left click", msg[1])
	g.Eq(el, msg[2])

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	_ = p.Mouse.MoveTo(proto.NewPoint(10, 10))
}

func TestTraceLogs(t *testing.T) {
	g := setup(t)

	g.browser.Logger(utils.LoggerQuiet)
	g.browser.Trace(true)
	defer func() {
		g.browser.Logger(rod.DefaultLogger)
		g.browser.Trace(defaults.Trace)
	}()

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	p.Overlay(0, 0, 100, 30, "")
}

func TestExposeHelpers(t *testing.T) {
	g := setup(t)

	p := g.newPage(g.srcFile("fixtures/click.html"))
	p.ExposeHelpers(js.ElementR)

	g.Eq(p.MustElementByJS(`() => rod.elementR('button', 'click me')`).MustText(), "click me")
}
