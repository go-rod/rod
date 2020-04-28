// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/assets"
	"github.com/ysmood/rod/lib/cdp"
)

// check method and sleep if needed
func (b *Browser) trySlowmotion(method string) {
	if b.slowmotion == 0 {
		return
	}

	if strings.HasPrefix(method, "Input.") {
		time.Sleep(b.slowmotion)
	}
}

// ServeMonitor starts the monitor server
// The reason why not to use "chrome://inspect/#devices" is one target cannot be driven by multiple controllers.
func (b *Browser) ServeMonitor(host string) *kit.ServerContext {
	if host == "" {
		return nil
	}

	srv := kit.MustServer(host)
	srv.Engine.GET("/", func(ctx kit.GinContext) {
		infos := b.Call("Target.getTargets", nil).Get("targetInfos")
		var list interface{}
		kit.E(json.Unmarshal([]byte(infos.Raw), &list))

		ctx.Header("Content-Type", "text/html; charset=utf-8")
		kit.E(ctx.Writer.WriteString(kit.S(assets.Monitor, "list", list)))
	})
	srv.Engine.GET("/page/:id", func(ctx kit.GinContext) {
		ctx.Header("Content-Type", "text/html; charset=utf-8")
		kit.E(ctx.Writer.WriteString(kit.S(
			assets.MonitorPage,
			"id", ctx.Param("id"),
			"rate", ctx.Query("rate"),
		)))
	})
	srv.Engine.GET("/screenshot/:id", func(ctx kit.GinContext) {
		p, err := b.page(ctx.Param("id"))
		kit.E(err)

		ctx.Header("Content-Type", "image/png;")
		kit.E(ctx.Writer.Write(p.Screenshot()))
	})

	go func() { _ = srv.Do() }()

	url := "http://" + srv.Listener.Addr().String()
	kit.Log("[rod] monitor server on", url, "(open it with your browser)")

	return srv
}

// Overlay a rectangle on the main frame with specified message
func (p *Page) Overlay(left, top, width, height float64, msg string) (remove func()) {
	root := p.Root()
	id := "rod-" + kit.RandString(8)

	_, err := root.EvalE(true, "", root.jsFn("overlay"), cdp.Array{
		id,
		left,
		top,
		width,
		height,
		msg,
	})
	CancelPanic(err)

	remove = func() {
		_, _ = root.EvalE(true, "", root.jsFn("removeOverlay"), cdp.Array{id})
	}

	return
}

// Trace with an overlay on the element
func (el *Element) Trace(htmlMessage string) (removeOverlay func()) {
	id := "rod-" + kit.RandString(8)

	_, err := el.EvalE(true, el.page.jsFn("elementOverlay"), cdp.Array{
		id,
		htmlMessage,
	})
	CancelPanic(err)

	removeOverlay = func() {
		_, _ = el.EvalE(true, el.page.jsFn("removeOverlay"), cdp.Array{id})
	}

	return
}

func (p *Page) stripHTML(str string) string {
	return p.Eval(p.jsFn("stripHTML"), str).String()
}

func (p *Page) traceFn(js string, params cdp.Array) func() {
	fnName := strings.Replace(js, p.jsFnPrefix(), "rod.", 1)
	paramsStr := p.stripHTML(kit.MustToJSON(params))
	msg := fmt.Sprintf("retry <code>%s(%s)</code>", fnName, paramsStr[1:len(paramsStr)-1])
	return p.Overlay(0, 0, 500, 0, msg)
}
