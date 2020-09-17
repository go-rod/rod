// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// ServeMonitor starts the monitor server.
// The reason why not to use "chrome://inspect/#devices" is one target cannot be driven by multiple controllers.
func (b *Browser) ServeMonitor(host string) string {
	u, mux, close := utils.Serve(host)
	go func() {
		<-b.ctx.Done()
		close()
	}()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httHTML(w, assets.Monitor)
	})
	mux.HandleFunc("/api/pages", func(w http.ResponseWriter, r *http.Request) {
		res, err := proto.TargetGetTargets{}.Call(b)
		utils.E(err)
		w.WriteHeader(http.StatusOK)
		utils.E(w.Write(utils.MustToJSONBytes(res.TargetInfos)))
	})
	mux.HandleFunc("/page/", func(w http.ResponseWriter, r *http.Request) {
		httHTML(w, assets.MonitorPage)
	})
	mux.HandleFunc("/api/page/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		info, err := b.pageInfo(proto.TargetTargetID(id))
		utils.E(err)
		w.WriteHeader(http.StatusOK)
		utils.E(w.Write(utils.MustToJSONBytes(info)))
	})
	mux.HandleFunc("/screenshot/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		target := proto.TargetTargetID(id)
		p := b.MustPageFromTargetID(target)

		w.Header().Add("Content-Type", "image/png;")
		utils.E(w.Write(p.MustScreenshot()))
	})

	return u
}

// TraceType for logger
type TraceType string

const (
	// TraceTypeWaitRequestsIdle type
	TraceTypeWaitRequestsIdle TraceType = "wait requests idle"

	// TraceTypeWaitRequests type
	TraceTypeWaitRequests TraceType = "wait requests"

	// TraceTypeJS type
	TraceTypeJS TraceType = "js"

	// TraceTypeAction type
	TraceTypeAction TraceType = "act"
)

// TraceMsg for logger
type TraceMsg struct {
	// Type of the message
	Type TraceType

	// Details is a json object
	Details interface{}
}

func (msg *TraceMsg) String() string {
	return utils.MustToJSON(msg)
}

// TraceLog handler
type TraceLog func(*TraceMsg)

// Overlay a rectangle on the main frame with specified message
func (p *Page) Overlay(left, top, width, height float64, msg string) (remove func()) {
	root := p.Root()
	id := utils.RandString(8)

	_, _ = root.EvalWithOptions(jsHelper(js.Overlay, JSArgs{
		id,
		left,
		top,
		width,
		height,
		msg,
	}))

	remove = func() {
		_, _ = root.EvalWithOptions(jsHelper(js.RemoveOverlay, JSArgs{id}))
	}

	return
}

// ExposeJSHelper to page's window object, so you can debug helper.js in the browser console.
// Such as run `rod.elementR("div", "ok")` in the browser console to test the Page.ElementR.
func (p *Page) ExposeJSHelper() *Page {
	p.MustEval(`rod => window.rod = rod`, proto.RuntimeRemoteObjectID(""))
	return p
}

// Trace with an overlay on the element
func (el *Element) Trace(msg string) (removeOverlay func()) {
	id := utils.RandString(8)

	_, _ = el.EvalWithOptions(jsHelper(js.ElementOverlay, JSArgs{
		id,
		msg,
	}))

	removeOverlay = func() {
		_, _ = el.EvalWithOptions(jsHelper(js.RemoveOverlay, JSArgs{id}))
	}

	return
}

// check method and sleep if needed
func (b *Browser) trySlowmotion() {
	if b.slowmotion == 0 {
		return
	}

	time.Sleep(b.slowmotion)
}

func (el *Element) traceAction(details string) func() {
	if !el.page.browser.trace {
		return func() {}
	}

	msg := &TraceMsg{TraceTypeAction, details}

	if !el.page.browser.quiet {
		el.page.browser.traceLog(msg)
	}

	return el.Trace(msg.String())
}

var regHelperJS = regexp.MustCompile(`\A\(rod, \.\.\.args\) => (rod\..+)\.apply\(this, `)

func (p *Page) tryTraceFn(js string, params JSArgs) func() {
	if !p.browser.trace {
		return func() {}
	}

	matches := regHelperJS.FindStringSubmatch(js)
	if matches != nil {
		js = matches[1]
		params = params[1:]
	}
	paramsStr := strings.Trim(mustToJSONForDev(params), "[]\r\n")

	if !p.browser.quiet {
		p.browser.traceLog(&TraceMsg{
			TraceTypeJS,
			map[string]interface{}{
				"js":     js,
				"params": params,
			},
		})
	}

	msg := fmt.Sprintf("js <code>%s(%s)</code>", js, html.EscapeString(paramsStr))
	return p.Overlay(0, 0, 500, 0, msg)
}

func (p *Page) traceReq(ctx context.Context, reqList *sync.Map, includes, excludes []string) {
	if !p.browser.trace {
		return
	}

	msg := &TraceMsg{TraceTypeWaitRequestsIdle, map[string][]string{
		"includes": includes,
		"excludes": excludes,
	}}
	p.browser.traceLog(msg)
	cleanup := p.Overlay(0, 0, 300, 0, msg.String())

	go func() {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				cleanup()
				return
			case <-t.C:
				p.browser.traceLog(&TraceMsg{
					TraceTypeWaitRequests,
					utils.SyncMapToMap(reqList),
				})
			}
		}
	}()
}

func defaultTraceLog(msg *TraceMsg) {
	log.Println(msg)
}

func (m *Mouse) initMouseTracer() {
	_, _ = m.page.EvalWithOptions(jsHelper(js.InitMouseTracer, JSArgs{m.id, assets.MousePointer}))
}

func (m *Mouse) updateMouseTracer() bool {
	res, err := m.page.EvalWithOptions(jsHelper(js.UpdateMouseTracer, JSArgs{m.id, m.x, m.y}))
	if err != nil {
		return true
	}
	return res.Value.Bool()
}
