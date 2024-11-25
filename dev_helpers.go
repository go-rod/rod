// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

// TraceType for logger.
type TraceType string

// String interface.
func (t TraceType) String() string {
	return fmt.Sprintf("[%s]", string(t))
}

const (
	// TraceTypeWaitRequestsIdle type.
	TraceTypeWaitRequestsIdle TraceType = "wait requests idle"

	// TraceTypeWaitRequests type.
	TraceTypeWaitRequests TraceType = "wait requests"

	// TraceTypeQuery type.
	TraceTypeQuery TraceType = "query"

	// TraceTypeWait type.
	TraceTypeWait TraceType = "wait"

	// TraceTypeInput type.
	TraceTypeInput TraceType = "input"
)

type screencastPage struct {
	frames chan []byte
	stop   func()
	once   sync.Once
}

// ServeMonitor starts the monitor server.
// The reason why not to use "chrome://inspect/#devices" is one target cannot be driven by multiple controllers.
func (b *Browser) ServeMonitor(host string) string {
	u, mux, closeSvr := serve(host)
	go func() {
		<-b.ctx.Done()
		utils.E(closeSvr())
	}()

	activeScreencasts := make(map[proto.TargetTargetID]*screencastPage)
	screencastMux := sync.RWMutex{}

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		httHTML(w, assets.Monitor)
	})
	mux.HandleFunc("/api/pages", func(w http.ResponseWriter, _ *http.Request) {
		res, err := proto.TargetGetTargets{}.Call(b) //nolint: contextcheck
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
	mux.HandleFunc("/page/", func(w http.ResponseWriter, _ *http.Request) {
		httHTML(w, assets.MonitorPage)
	})
	mux.HandleFunc("/api/page/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		info, err := b.pageInfo(proto.TargetTargetID(id)) //nolint: contextcheck
		utils.E(err)
		w.WriteHeader(http.StatusOK)
		utils.E(w.Write(utils.MustToJSONBytes(info)))
	})
	mux.HandleFunc("/screenshot/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		target := proto.TargetTargetID(id)
		p := b.MustPageFromTargetID(target)

		w.Header().Add("Content-Type", "image/png;")
		utils.E(w.Write(p.MustScreenshot())) //nolint: contextcheck
	})
	mux.HandleFunc("/screencast/", func(w http.ResponseWriter, r *http.Request) {
		b.handleScreencast(w, r, activeScreencasts, &screencastMux)
	})

	return u
}

// handleScreencast handles the ServeMonitor screencasting functionality for a target.
func (b *Browser) handleScreencast(
	w http.ResponseWriter,
	r *http.Request,
	activeScreencasts map[proto.TargetTargetID]*screencastPage,
	screencastMux *sync.RWMutex,
) {
	id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	target := proto.TargetTargetID(id)

	// Set headers for MJPEG stream
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Pragma", "no-cache")

	screencastMux.RLock()
	sc, exists := activeScreencasts[target]
	screencastMux.RUnlock()

	if !exists {
		p := b.MustPageFromTargetID(target)
		frames := make(chan []byte, 100)

		stopCtx, cancelFunc := context.WithCancel(r.Context())

		sc = &screencastPage{
			frames: frames,
			once:   sync.Once{},
			stop: func() {
				cancelFunc()
			},
		}

		screencastMux.Lock()
		activeScreencasts[target] = sc
		screencastMux.Unlock()

		go func() {
			defer sc.once.Do(func() {
				p.MustStopScreencast()
				close(sc.frames)
				screencastMux.Lock()
				delete(activeScreencasts, target)
				screencastMux.Unlock()
			})

			opts := &ScreencastOptions{
				Format:        "jpeg",
				Quality:       gson.Int(75),
				EveryNthFrame: gson.Int(1),
				BufferSize:    100,
			}

			frames, err := p.StartScreencast(opts)
			if err != nil {
				return
			}

			for {
				select {
				case <-stopCtx.Done():
					return
				case frame, ok := <-frames:
					if !ok {
						return
					}
					select {
					case sc.frames <- frame:
					default:
						// Skip frame if buffer is full
					}
				}
			}
		}()
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case <-r.Context().Done():
			sc.stop()
			return
		case frame, ok := <-sc.frames:
			if !ok {
				return
			}

			utils.MustWriteMJPEGFrame(w, frame, flusher)
		}
	}
}

// check method and sleep if needed.
func (b *Browser) trySlowMotion() {
	if b.slowMotion == 0 {
		return
	}

	time.Sleep(b.slowMotion)
}

// ExposeHelpers helper functions to page's js context so that we can use the Devtools' console to debug them.
func (p *Page) ExposeHelpers(list ...*js.Function) {
	p.MustEvaluate(evalHelper(&js.Function{
		Name:         "_" + utils.RandString(8), // use a random name so it won't hit the cache
		Definition:   "() => { window.rod = functions }",
		Dependencies: list,
	}))
}

// Overlay a rectangle on the main frame with specified message.
func (p *Page) Overlay(left, top, width, height float64, msg string) (remove func()) {
	id := utils.RandString(8)

	_, _ = p.root.Evaluate(evalHelper(js.Overlay,
		id,
		left,
		top,
		width,
		height,
		msg,
	).ByPromise())

	remove = func() {
		_, _ = p.root.Evaluate(evalHelper(js.RemoveOverlay, id))
	}

	return
}

func (p *Page) tryTrace(typ TraceType, msg ...interface{}) func() {
	if !p.browser.trace {
		return func() {}
	}

	msg = append([]interface{}{typ}, msg...)
	msg = append(msg, p)

	p.browser.logger.Println(msg...)

	return p.Overlay(0, 0, 500, 0, fmt.Sprint(msg))
}

func (p *Page) tryTraceQuery(opts *EvalOptions) func() {
	if !p.browser.trace {
		return func() {}
	}

	p.browser.logger.Println(TraceTypeQuery, opts, p)

	msg := fmt.Sprintf("<code>%s</code>", html.EscapeString(opts.String()))
	return p.Overlay(0, 0, 500, 0, msg)
}

func (p *Page) tryTraceReq(includes, excludes []string) func(map[proto.NetworkRequestID]string) {
	if !p.browser.trace {
		return func(map[proto.NetworkRequestID]string) {}
	}

	msg := map[string][]string{
		"includes": includes,
		"excludes": excludes,
	}
	p.browser.logger.Println(TraceTypeWaitRequestsIdle, msg, p)
	cleanup := p.Overlay(0, 0, 500, 0, utils.MustToJSON(msg))

	ch := make(chan map[string]string)
	update := func(list map[proto.NetworkRequestID]string) {
		clone := map[string]string{}
		for k, v := range list {
			clone[string(k)] = v
		}
		ch <- clone
	}

	go func() {
		var waitList map[string]string
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-p.ctx.Done():
				t.Stop()
				cleanup()
				return
			case waitList = <-ch:
			case <-t.C:
				p.browser.logger.Println(TraceTypeWaitRequests, p, waitList)
			}
		}
	}()

	return update
}

// Overlay msg on the element.
func (el *Element) Overlay(msg string) (removeOverlay func()) {
	id := utils.RandString(8)

	_, _ = el.Evaluate(evalHelper(js.ElementOverlay,
		id,
		msg,
	).ByPromise())

	removeOverlay = func() {
		_, _ = el.Evaluate(evalHelper(js.RemoveOverlay, id))
	}

	return
}

func (el *Element) tryTrace(typ TraceType, msg ...interface{}) func() {
	if !el.page.browser.trace {
		return func() {}
	}

	msg = append([]interface{}{typ}, msg...)
	msg = append(msg, el)

	el.page.browser.logger.Println(msg...)

	return el.Overlay(fmt.Sprint(msg))
}

func (m *Mouse) initMouseTracer() {
	_, _ = m.page.Evaluate(evalHelper(js.InitMouseTracer, m.id, assets.MousePointer).ByPromise())
}

func (m *Mouse) updateMouseTracer() bool {
	res, err := m.page.Evaluate(evalHelper(js.UpdateMouseTracer, m.id, m.pos.X, m.pos.Y))
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
				if nerr, ok := err.(*net.OpError); ok && strings.Contains(nerr.Err.Error(), "broken pipe") {
					return
				}
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
