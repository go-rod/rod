package rod

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/proto"
)

// HijackRequests creates a new router instance for requests hijacking.
// A router must be singleton for a page. Enabling hijacking disables page caching,
// but such as 304 Not Modified will still work as expected.
func (b *Browser) HijackRequests() *HijackRouter {
	return newHijackRouter(b, b).initEvents()
}

// HijackRequests same as Browser.HijackRequests, but scoped with the page
func (p *Page) HijackRequests() *HijackRouter {
	return newHijackRouter(p.browser, p).initEvents()
}

// HijackRouter context
type HijackRouter struct {
	run        func()
	stopEvents func()
	handlers   []*hijackHandler
	enable     *proto.FetchEnable
	caller     proto.Caller
	browser    *Browser
}

func newHijackRouter(browser *Browser, caller proto.Caller) *HijackRouter {
	return &HijackRouter{
		enable:   &proto.FetchEnable{},
		browser:  browser,
		caller:   caller,
		handlers: []*hijackHandler{},
	}
}

func (r *HijackRouter) initEvents() *HijackRouter {
	ctx, _, sessionID := r.caller.CallContext()
	eventCtx, cancel := context.WithCancel(ctx)
	r.stopEvents = cancel

	_ = r.enable.Call(r.caller)

	r.run = r.browser.eachEvent(eventCtx, proto.TargetSessionID(sessionID), func(e *proto.FetchRequestPaused) bool {
		go func() {
			ctx := r.new(e)
			for _, h := range r.handlers {
				if h.regexp.MatchString(e.Request.URL) {
					h.handler(ctx)

					if ctx.Skip {
						return
					}

					err := ctx.Response.payload.Call(r.caller)
					if err != nil {
						ctx.OnError(err)
						return
					}
				}
			}
		}()

		return false
	})
	return r
}

// AddE a hijack handler to router, the doc of the pattern is the same as "proto.FetchRequestPattern.URLPattern".
// You can add new handler even after the "Run" is called.
func (r *HijackRouter) AddE(pattern string, handler func(*Hijack)) error {
	r.enable.Patterns = append(r.enable.Patterns, &proto.FetchRequestPattern{
		URLPattern: pattern,
	})

	reg := regexp.MustCompile(proto.PatternToReg(pattern))

	r.handlers = append(r.handlers, &hijackHandler{
		pattern: pattern,
		regexp:  reg,
		handler: handler,
	})

	return r.enable.Call(r.caller)
}

// Add a hijack handler to router, the doc of the pattern is the same as "proto.FetchRequestPattern.URLPattern".
// You can add new handler even after the "Run" is called.
func (r *HijackRouter) Add(pattern string, handler func(*Hijack)) {
	kit.E(r.AddE(pattern, handler))
}

// RemoveE handler via the pattern
func (r *HijackRouter) RemoveE(pattern string) error {
	patterns := []*proto.FetchRequestPattern{}
	handlers := []*hijackHandler{}
	for _, h := range r.handlers {
		if h.pattern != pattern {
			patterns = append(patterns, &proto.FetchRequestPattern{URLPattern: h.pattern})
			handlers = append(handlers, h)
		}
	}
	r.enable.Patterns = patterns
	r.handlers = handlers

	return r.enable.Call(r.caller)
}

// Remove handler via the pattern
func (r *HijackRouter) Remove(pattern string) {
	kit.E(r.RemoveE(pattern))
}

// new context
func (r *HijackRouter) new(e *proto.FetchRequestPaused) *Hijack {
	headers := http.Header{}
	for k, v := range e.Request.Headers {
		headers[k] = []string{v.String()}
	}

	req := kit.Req(e.Request.URL).
		Method(e.Request.Method).
		Headers(headers).
		StringBody(e.Request.PostData)

	return &Hijack{
		Request: &HijackRequest{
			event: e,
			req:   req,
		},
		Response: &HijackResponse{
			req: req,
			payload: &proto.FetchFulfillRequest{
				ResponseCode: 200,
				RequestID:    e.RequestID,
			},
		},
		OnError: func(err error) { kit.Err(err) },
	}
}

// RunE the router, after you call it, you shouldn't add new handler to it.
func (r *HijackRouter) RunE() error {
	r.run()
	return r.enable.Call(r.caller)
}

// Run the router, after you call it, you shouldn't add new handler to it.
// You can stop and run the same router without limitation.
func (r *HijackRouter) Run() {
	kit.E(r.RunE())
}

// StopE the router
func (r *HijackRouter) StopE() error {
	r.stopEvents()
	return proto.FetchDisable{}.Call(r.caller)
}

// Stop the router
func (r *HijackRouter) Stop() {
	kit.E(r.StopE())
}

// hijackHandler to handle each request that match the regexp
type hijackHandler struct {
	pattern string
	regexp  *regexp.Regexp
	handler func(*Hijack)
}

// Hijack context
type Hijack struct {
	Request  *HijackRequest
	Response *HijackResponse
	OnError  func(error)

	// Skip to next handler
	Skip bool
}

// LoadResponseE will send request to the real destination and load the response as default response to override.
func (h *Hijack) LoadResponseE() error {
	code, err := h.Response.StatusCodeE()
	if err != nil {
		return err
	}
	h.Response.SetStatusCode(code)

	headers, err := h.Response.HeadersE()
	if err != nil {
		return err
	}
	list := []string{}
	for k, vs := range headers {
		for _, v := range vs {
			list = append(list, k, v)
		}
	}
	h.Response.SetHeader(list...)

	body, err := h.Response.BodyE()
	if err != nil {
		return err
	}
	h.Response.SetBody(body)

	return nil
}

// LoadResponse will send request to the real destination and load the response as default response to override.
func (h *Hijack) LoadResponse() {
	kit.E(h.LoadResponseE())
}

// HijackRequest context
type HijackRequest struct {
	event *proto.FetchRequestPaused
	req   *kit.ReqContext
}

// Method of the request
func (ctx *HijackRequest) Method() string {
	return ctx.event.Request.Method
}

// URL of the request
func (ctx *HijackRequest) URL() *url.URL {
	u, err := url.Parse(ctx.event.Request.URL)
	kit.E(err) // no way this will happen, if it happens it's fatal
	return u
}

// Header via a key
func (ctx *HijackRequest) Header(key string) string {
	return ctx.event.Request.Headers[key].String()
}

// Headers of request
func (ctx *HijackRequest) Headers() proto.NetworkHeaders {
	return ctx.event.Request.Headers
}

// Body of the request, devtools API doesn't support binary data yet, only string can be captured.
func (ctx *HijackRequest) Body() string {
	return ctx.event.Request.PostData
}

// JSONBody of the request
func (ctx *HijackRequest) JSONBody() gjson.Result {
	return gjson.Parse(ctx.Body())
}

// SetMethod of request
func (ctx *HijackRequest) SetMethod(name string) {
	ctx.req.Method(name)
}

// SetHeader via key-value pairs
func (ctx *HijackRequest) SetHeader(pairs ...string) {
	ctx.req.Header(pairs...)
}

// SetQuery of the request, example Query(k, v, k, v ...)
func (ctx *HijackRequest) SetQuery(pairs ...interface{}) {
	ctx.req.Query(pairs...)
}

// SetURL of the request
func (ctx *HijackRequest) SetURL(url string) {
	ctx.req.URL(url)
}

// SetBody of the request, if obj is []byte or string, raw body will be used, else it will be encoded as json.
func (ctx *HijackRequest) SetBody(obj interface{}) {
	// reset to empty
	ctx.req.StringBody("")
	ctx.req.JSONBody(nil)

	switch body := obj.(type) {
	case []byte:
		buf := bytes.NewBuffer(body)
		ctx.req.Body(buf)
	case string:
		ctx.req.StringBody(body)
	default:
		ctx.req.JSONBody(obj)
	}
}

// HijackResponse context
type HijackResponse struct {
	req     *kit.ReqContext
	payload *proto.FetchFulfillRequest
}

// StatusCodeE of response
func (ctx *HijackResponse) StatusCodeE() (int, error) {
	res, err := ctx.req.Response()
	if err != nil {
		return 0, err
	}

	return res.StatusCode, nil
}

// StatusCode of response
func (ctx *HijackResponse) StatusCode() int {
	code, err := ctx.StatusCodeE()
	kit.E(err)
	return code
}

// SetStatusCode of response
func (ctx *HijackResponse) SetStatusCode(code int) {
	ctx.payload.ResponseCode = int64(code)
}

// HeaderE via key
func (ctx *HijackResponse) HeaderE(key string) (string, error) {
	res, err := ctx.req.Response()
	if err != nil {
		return "", err
	}

	return res.Header.Get(key), nil
}

// Header via key
func (ctx *HijackResponse) Header(key string) string {
	val, err := ctx.HeaderE(key)
	kit.E(err)
	return val
}

// HeadersE of request
func (ctx *HijackResponse) HeadersE() (http.Header, error) {
	res, err := ctx.req.Response()
	if err != nil {
		return nil, err
	}

	return res.Header, nil
}

// Headers of request
func (ctx *HijackResponse) Headers() http.Header {
	val, err := ctx.HeadersE()
	kit.E(err)
	return val
}

// SetHeader via key-value pairs
func (ctx *HijackResponse) SetHeader(pairs ...string) {
	for i := 0; i < len(pairs); i += 2 {
		ctx.payload.ResponseHeaders = append(ctx.payload.ResponseHeaders, &proto.FetchHeaderEntry{
			Name:  pairs[i],
			Value: pairs[i+1],
		})
	}
}

// BodyE of response
func (ctx *HijackResponse) BodyE() ([]byte, error) {
	b, err := ctx.req.Bytes()
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Body of response
func (ctx *HijackResponse) Body() []byte {
	b, err := ctx.BodyE()
	kit.E(err)
	return b
}

// StringBody of response
func (ctx *HijackResponse) StringBody() string {
	return string(ctx.Body())
}

// JSONBody of response
func (ctx *HijackResponse) JSONBody() gjson.Result {
	return gjson.ParseBytes(ctx.Body())
}

// SetBody of response, if obj is []byte, raw body will be used, else it will be encoded as json
func (ctx *HijackResponse) SetBody(obj interface{}) {
	switch body := obj.(type) {
	case []byte:
		ctx.payload.Body = body
	case string:
		ctx.payload.Body = []byte(body)
	default:
		ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
		var err error
		ctx.payload.Body, err = json.Marshal(obj)
		kit.E(err)
	}
}

// GetDownloadFileE of the next download url that matches the pattern, returns the file content.
// The handler will be used once and removed.
func (p *Page) GetDownloadFileE(pattern string) func() ([]byte, error) {
	_ = proto.BrowserSetDownloadBehavior{
		Behavior:         proto.BrowserSetDownloadBehaviorBehaviorDeny,
		BrowserContextID: p.browser.BrowserContextID,
	}.Call(p)

	r := p.HijackRequests()

	return func() ([]byte, error) {
		defer func() {
			_ = proto.BrowserSetDownloadBehavior{
				Behavior:         proto.BrowserSetDownloadBehaviorBehaviorDefault,
				BrowserContextID: r.browser.BrowserContextID,
			}.Call(r.caller)
		}()

		var data []byte
		wg := &sync.WaitGroup{}
		wg.Add(1)

		var err error
		err = r.AddE(pattern, func(ctx *Hijack) {
			defer wg.Done()

			ctx.OnError = func(e error) {
				err = e
			}

			err = ctx.LoadResponseE()
			if err != nil {
				return
			}

			data, err = ctx.Response.BodyE()
		})
		if err != nil {
			return nil, err
		}

		go r.Run()
		defer r.Stop()

		wg.Wait()

		if err != nil {
			return nil, err
		}

		return data, nil
	}
}

// GetDownloadFile of the next download url that matches the pattern, returns the file content.
func (p *Page) GetDownloadFile(pattern string) func() []byte {
	wait := p.GetDownloadFileE(pattern)
	return func() []byte {
		data, err := wait()
		kit.E(err)
		return data
	}
}

// HandleAuthE for the next basic HTTP authentication.
// It will prevent the popup that requires user to input user name and password.
// Ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
func (b *Browser) HandleAuthE(username, password string) func() error {
	recover := b.EnableDomain(b.ctx, "", &proto.FetchEnable{
		HandleAuthRequests: true,
	})

	paused := &proto.FetchRequestPaused{}
	auth := &proto.FetchAuthRequired{}

	waitPaused := b.WaitEvent(paused)
	waitAuth := b.WaitEvent(auth)

	return func() (err error) {
		defer recover()

		waitPaused()

		err = proto.FetchContinueRequest{
			RequestID: paused.RequestID,
		}.Call(b)
		if err != nil {
			return
		}

		waitAuth()

		err = proto.FetchContinueWithAuth{
			RequestID: auth.RequestID,
			AuthChallengeResponse: &proto.FetchAuthChallengeResponse{
				Response: proto.FetchAuthChallengeResponseResponseProvideCredentials,
				Username: username,
				Password: password,
			},
		}.Call(b)

		return
	}
}
