// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strings"
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

		list := []*proto.TargetTargetInfo{}
		for _, info := range res.TargetInfos {
			if info.Type == proto.TargetTargetInfoTypePage {
				list = append(list, info)
			}
		}

		w.WriteHeader(http.StatusOK)
		utils.E(w.Write(utils.MustToJSONBytes(list)))
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

	// TraceTypeEval type
	TraceTypeEval TraceType = "eval"

	// TraceTypeAction type
	TraceTypeAction TraceType = "act"

	// TraceTypeInput type
	TraceTypeInput TraceType = "input"
)

// TraceMsg for logger
type TraceMsg struct {
	// Type of the message
	Type TraceType

	// Details is a json object
	Details interface{}
}

func (msg *TraceMsg) String() string {
	return fmt.Sprintf("[%s] %v", msg.Type, utils.MustToJSON(msg.Details))
}

// TraceLog handler
type TraceLog func(*TraceMsg)

// Overlay a rectangle on the main frame with specified message
func (p *Page) Overlay(left, top, width, height float64, msg string) (remove func()) {
	root := p.Root()
	id := utils.RandString(8)

	_, _ = root.Evaluate(jsHelper(js.Overlay,
		id,
		left,
		top,
		width,
		height,
		msg,
	))

	remove = func() {
		_, _ = root.Evaluate(jsHelper(js.RemoveOverlay, id))
	}

	return
}

// ExposeJSHelper to page's window object, so you can debug helper.js in the browser console.
// Such as run `rod.elementR("div", "ok")` in the browser console to test the Page.ElementR.
func (p *Page) ExposeJSHelper() *Page {
	p.MustEval(`rod => window.rod = rod`, p.jsHelperObj)
	return p
}

// Trace with an overlay on the element
func (el *Element) Trace(msg string) (removeOverlay func()) {
	id := utils.RandString(8)

	_, _ = el.Evaluate(jsHelper(js.ElementOverlay,
		id,
		msg,
	))

	removeOverlay = func() {
		_, _ = el.Evaluate(jsHelper(js.RemoveOverlay, id))
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

func (el *Element) tryTraceInput(details string) func() {
	if !el.page.browser.trace {
		return func() {}
	}

	msg := &TraceMsg{TraceTypeInput, details}

	el.page.browser.traceLog(msg)

	return el.Trace(details)
}

var regHelperJS = regexp.MustCompile(`\A\(rod, \.\.\.args\) => (rod\..+)\.apply\(this, `)

func (p *Page) tryTraceEval(js string, params []interface{}) func() {
	if !p.browser.trace {
		return func() {}
	}

	matches := regHelperJS.FindStringSubmatch(js)
	if matches != nil {
		js = matches[1]
	}
	paramsStr := strings.Trim(mustToJSONForDev(params), "[]\r\n")

	p.browser.traceLog(&TraceMsg{
		TraceTypeEval,
		map[string]interface{}{
			"js":     js,
			"params": params,
		},
	})

	msg := fmt.Sprintf("js <code>%s(%s)</code>", js, html.EscapeString(paramsStr))
	return p.Overlay(0, 0, 500, 0, msg)
}

func (p *Page) tryTraceReq(includes, excludes []string) func(map[proto.NetworkRequestID]string) {
	if !p.browser.trace {
		return func(map[proto.NetworkRequestID]string) {}
	}

	msg := &TraceMsg{TraceTypeWaitRequestsIdle, map[string][]string{
		"includes": includes,
		"excludes": excludes,
	}}
	p.browser.traceLog(msg)
	cleanup := p.Overlay(0, 0, 300, 0, msg.String())

	ch := make(chan map[string]string)
	update := func(list map[proto.NetworkRequestID]string) {
		clone := map[string]string{}
		for k, v := range list {
			clone[string(k)] = v
		}
		ch <- clone
	}

	go func() {
		var waitlist map[string]string
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-p.ctx.Done():
				t.Stop()
				cleanup()
				return
			case waitlist = <-ch:
			case <-t.C:
				p.browser.traceLog(&TraceMsg{
					TraceTypeWaitRequests,
					waitlist,
				})
			}
		}
	}()

	return update
}

func defaultTraceLog(msg *TraceMsg) {
	log.Println(msg)
}

func (m *Mouse) initMouseTracer() {
	_, _ = m.page.Evaluate(jsHelper(js.InitMouseTracer, m.id, assets.MousePointer))
}

func (m *Mouse) updateMouseTracer() bool {
	res, err := m.page.Evaluate(jsHelper(js.UpdateMouseTracer, m.id, m.x, m.y))
	if err != nil {
		return true
	}
	return res.Value.Bool()
}
