// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/assets"
	"github.com/ysmood/rod/lib/proto"
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
		res, err := proto.TargetGetTargets{}.Call(b)
		kit.E(err)

		ginHTML(ctx, kit.S(assets.Monitor, "list", res.TargetInfos))
	})
	srv.Engine.GET("/page/:id", func(ctx kit.GinContext) {
		ginHTML(ctx, kit.S(
			assets.MonitorPage,
			"id", ctx.Param("id"),
		))
	})
	srv.Engine.GET("/api/page/:id", func(ctx kit.GinContext) {
		info, err := proto.TargetGetTargetInfo{
			TargetID: proto.TargetTargetID(ctx.Param("id")),
		}.Call(b)
		kit.E(err)
		ctx.PureJSON(http.StatusOK, info.TargetInfo)
	})
	srv.Engine.GET("/screenshot/:id", func(ctx kit.GinContext) {
		id := proto.TargetTargetID(ctx.Param("id"))
		p := b.PageFromTargetID(id)

		ctx.Header("Content-Type", "image/png;")
		_, _ = ctx.Writer.Write(p.Screenshot())
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

	_, err := root.EvalE(true, "", root.jsFn("overlay"), Array{
		id,
		left,
		top,
		width,
		height,
		msg,
	})
	CancelPanic(err)

	remove = func() {
		_, _ = root.EvalE(true, "", root.jsFn("removeOverlay"), Array{id})
	}

	return
}

// Trace with an overlay on the element
func (el *Element) Trace(htmlMessage string) (removeOverlay func()) {
	id := "rod-" + kit.RandString(8)

	_, err := el.EvalE(true, el.page.jsFn("elementOverlay"), Array{
		id,
		htmlMessage,
	})
	CancelPanic(err)

	removeOverlay = func() {
		_, _ = el.EvalE(true, el.page.jsFn("removeOverlay"), Array{id})
	}

	return
}

func (p *Page) traceFn(js string, params Array) func() {
	fnName := strings.Replace(js, p.jsFnPrefix(), "rod.", 1)
	paramsStr := html.EscapeString(strings.Trim(kit.MustToJSON(params), "[]"))
	msg := fmt.Sprintf("retry <code>%s(%s)</code>", fnName, paramsStr)
	return p.Overlay(0, 0, 500, 0, msg)
}
