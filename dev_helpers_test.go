package rod_test

import (
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func (t T) Monitor() {
	b := rod.New().MustConnect()
	defer b.MustClose()
	p := b.MustPage(t.blank()).MustWaitLoad()

	b, cancel := b.WithCancel()
	defer cancel()
	host := b.Context(t.Context()).ServeMonitor("")

	page := t.page.MustNavigate(host)
	t.Has(page.MustElement("#targets a").MustParent().MustHTML(), string(p.TargetID))

	page.MustNavigate(host + "/page/" + string(p.TargetID))
	page.MustWait(`(id) => document.title.includes(id)`, p.TargetID)

	img := t.Req("", host+"/screenshot").Bytes()
	t.Gt(len(img), 10)

	res := t.Req("", host+"/api/page/test")
	t.Eq(400, res.StatusCode)
	t.Eq(-32602, gson.New(res.Body).Get("code").Int())
}

func (t T) MonitorErr() {
	l := launcher.New()
	u := l.MustLaunch()
	defer l.Kill()

	t.Panic(func() {
		rod.New().Monitor("abc").ControlURL(u).MustConnect()
	})
}

func (t T) Trace() {
	t.Eq(rod.TraceTypeInput.String(), "[input]")

	var msg []interface{}
	t.browser.Logger(utils.Log(func(list ...interface{}) { msg = list }))
	t.browser.Trace(true).SlowMotion(time.Microsecond)
	defer func() {
		t.browser.Logger(rod.DefaultLogger)
		t.browser.Trace(defaults.Trace).SlowMotion(defaults.Slow)
	}()

	p := t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustWaitLoad()

	t.Eq(rod.TraceTypeWait, msg[0])
	t.Eq("load", msg[1])
	t.Eq(p, msg[2])

	el := p.MustElement("button")
	el.MustClick()

	t.Eq(rod.TraceTypeInput, msg[0])
	t.Eq("left click", msg[1])
	t.Eq(el, msg[2])

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	_ = p.Mouse.Move(10, 10, 1)
}

func (t T) TraceLogs() {
	t.browser.Logger(utils.LoggerQuiet)
	t.browser.Trace(true)
	defer func() {
		t.browser.Logger(rod.DefaultLogger)
		t.browser.Trace(defaults.Trace)
	}()

	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	p.Overlay(0, 0, 100, 30, "")
}

func (t T) ExposeHelpers() {
	p := t.newPage(t.srcFile("fixtures/click.html"))
	p.ExposeHelpers(js.ElementR)

	t.Eq(p.MustElementByJS(`rod.elementR('button', 'click me')`).MustText(), "click me")
}
