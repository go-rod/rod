// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// ServeMonitor starts the monitor server.
// The reason why not to use "chrome://inspect/#devices" is one target cannot be driven by multiple controllers.
func (b *Browser) ServeMonitor(host string) string {
	url, mux, close := serve(host)
	go func() {
		<-b.ctx.Done()
		utils.E(close())
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

	return url
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

// Overlay a rectangle on the main frame with specified message
func (p *Page) Overlay(left, top, width, height float64, msg string) (remove func()) {
	id := utils.RandString(8)

	_, _ = p.root.Evaluate(jsHelper(js.Overlay,
		id,
		left,
		top,
		width,
		height,
		msg,
	).ByPromise())

	remove = func() {
		_, _ = p.root.Evaluate(jsHelper(js.RemoveOverlay, id))
	}

	return
}

// Trace with an overlay on the element
func (el *Element) Trace(msg string) (removeOverlay func()) {
	id := utils.RandString(8)

	_, _ = el.Evaluate(jsHelper(js.ElementOverlay,
		id,
		msg,
	).ByPromise())

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

	el.page.browser.logger.Println(msg)

	return el.Trace(details)
}

func (p *Page) tryTraceEval(opts *EvalOptions) func() {
	if !p.browser.trace {
		return func() {}
	}

	fn := ""

	if opts.jsHelper != nil {
		fn = "rod." + opts.jsHelper.Name
	}

	paramsStr := strings.Trim(mustToJSONForDev(opts.JSArgs), "[]\r\n")

	p.browser.logger.Println(&TraceMsg{
		TraceTypeEval,
		map[string]interface{}{
			"js":     fn,
			"params": opts.JSArgs,
		},
	})

	msg := fmt.Sprintf("js <code>%s(%s)</code>", fn, html.EscapeString(paramsStr))
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
	p.browser.logger.Println(msg)
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
				p.browser.logger.Println(&TraceMsg{
					TraceTypeWaitRequests,
					waitlist,
				})
			}
		}
	}()

	return update
}

func (m *Mouse) initMouseTracer() {
	_, _ = m.page.Evaluate(jsHelper(js.InitMouseTracer, m.id, assets.MousePointer).ByPromise())
}

func (m *Mouse) updateMouseTracer() bool {
	res, err := m.page.Evaluate(jsHelper(js.UpdateMouseTracer, m.id, m.x, m.y))
	if err != nil {
		return true
	}
	return res.Value.Bool()
}

// Serve a port, if host is empty a random port will be used.
func serve(host string) (string, *http.ServeMux, func() error) {
	if host == "" {
		host = "127.0.0.1:0"
	}

	mux := http.NewServeMux()
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				utils.E(json.NewEncoder(w).Encode(err))
			}
		}()

		mux.ServeHTTP(w, r)
	})}

	l, err := net.Listen("tcp", host)
	utils.E(err)

	go func() { _ = srv.Serve(l) }()

	url := "http://" + l.Addr().String()

	return url, mux, srv.Close
}
