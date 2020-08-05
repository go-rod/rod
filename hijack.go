package rod

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/tidwall/gjson"
	"github.com/ysmood/kit"
)

// HijackRequests creates a new router instance for requests hijacking.
// When use Fetch domain outside the router should be stopped. Enabling hijacking disables page caching,
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
			ctx := r.new(eventCtx, e)
			for _, h := range r.handlers {
				if h.regexp.MatchString(e.Request.URL) {
					h.handler(ctx)

					if ctx.continueRequest != nil {
						ctx.continueRequest.RequestID = e.RequestID
						err := ctx.continueRequest.Call(r.caller)
						if err != nil {
							ctx.OnError(err)
						}
						return
					}

					if ctx.Skip {
						continue
					}

					if ctx.Response.fail.ErrorReason != "" {
						err := ctx.Response.fail.Call(r.caller)
						if err != nil {
							ctx.OnError(err)
						}
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

// Add a hijack handler to router, the doc of the pattern is the same as "proto.FetchRequestPattern.URLPattern".
// You can add new handler even after the "Run" is called.
func (r *HijackRouter) Add(pattern string, resourceType proto.NetworkResourceType, handler func(*Hijack)) error {
	r.enable.Patterns = append(r.enable.Patterns, &proto.FetchRequestPattern{
		URLPattern:   pattern,
		ResourceType: resourceType,
	})

	reg := regexp.MustCompile(proto.PatternToReg(pattern))

	r.handlers = append(r.handlers, &hijackHandler{
		pattern: pattern,
		regexp:  reg,
		handler: handler,
	})

	return r.enable.Call(r.caller)
}

// Remove handler via the pattern
func (r *HijackRouter) Remove(pattern string) error {
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

// new context
func (r *HijackRouter) new(ctx context.Context, e *proto.FetchRequestPaused) *Hijack {
	headers := http.Header{}
	for k, v := range e.Request.Headers {
		headers[k] = []string{v.String()}
	}

	req := kit.Req(e.Request.URL).
		Method(e.Request.Method).
		Headers(headers).
		StringBody(e.Request.PostData).
		Context(ctx)

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
			fail: &proto.FetchFailRequest{
				RequestID: e.RequestID,
			},
		},
		OnError: func(err error) {
			if err != context.Canceled {
				log.Println(utils.C("[rod hijack err]", "yellow"), err)
			}
		},
	}
}

// Run the router, after you call it, you shouldn't add new handler to it.
func (r *HijackRouter) Run() {
	r.run()
}

// Stop the router
func (r *HijackRouter) Stop() error {
	r.stopEvents()
	return proto.FetchDisable{}.Call(r.caller)
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

	continueRequest *proto.FetchContinueRequest
}

// ContinueRequest without hijacking
func (h *Hijack) ContinueRequest(cq *proto.FetchContinueRequest) {
	h.continueRequest = cq
}

// LoadResponse will send request to the real destination and load the response as default response to override.
func (h *Hijack) LoadResponse(loadBody bool) error {
	code, err := h.Response.StatusCode()
	if err != nil {
		return err
	}
	h.Response.SetStatusCode(code)

	headers, err := h.Response.Headers()
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

	if loadBody {
		body, err := h.Response.Body()
		if err != nil {
			return err
		}
		h.Response.SetBody(body)
	}

	return nil
}

// HijackRequest context
type HijackRequest struct {
	event *proto.FetchRequestPaused
	req   *kit.ReqContext
}

// Type of the resource
func (ctx *HijackRequest) Type() proto.NetworkResourceType {
	return ctx.event.ResourceType
}

// Method of the request
func (ctx *HijackRequest) Method() string {
	return ctx.event.Request.Method
}

// URL of the request
func (ctx *HijackRequest) URL() *url.URL {
	u, err := url.Parse(ctx.event.Request.URL)
	utils.E(err) // no way this will happen, if it happens it's fatal
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
func (ctx *HijackRequest) SetMethod(name string) *HijackRequest {
	ctx.req.Method(name)
	return ctx
}

// SetHeader via key-value pairs
func (ctx *HijackRequest) SetHeader(pairs ...string) *HijackRequest {
	ctx.req.Header(pairs...)
	return ctx
}

// SetQuery of the request, example Query(k, v, k, v ...)
func (ctx *HijackRequest) SetQuery(pairs ...interface{}) *HijackRequest {
	ctx.req.Query(pairs...)
	return ctx
}

// SetURL of the request
func (ctx *HijackRequest) SetURL(url string) *HijackRequest {
	ctx.req.URL(url)
	return ctx
}

// SetBody of the request, if obj is []byte or string, raw body will be used, else it will be encoded as json.
func (ctx *HijackRequest) SetBody(obj interface{}) *HijackRequest {
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
	return ctx
}

// SetClient for http
func (ctx *HijackRequest) SetClient(client *http.Client) *HijackRequest {
	ctx.req.Client(client)
	return ctx
}

// HijackResponse context
type HijackResponse struct {
	req     *kit.ReqContext
	payload *proto.FetchFulfillRequest
	fail    *proto.FetchFailRequest
}

// StatusCode of response
func (ctx *HijackResponse) StatusCode() (int, error) {
	res, err := ctx.req.Response()
	if err != nil {
		return 0, err
	}

	return res.StatusCode, nil
}

// SetStatusCode of response
func (ctx *HijackResponse) SetStatusCode(code int) {
	ctx.payload.ResponseCode = int64(code)
}

// Header via key
func (ctx *HijackResponse) Header(key string) (string, error) {
	res, err := ctx.req.Response()
	if err != nil {
		return "", err
	}

	return res.Header.Get(key), nil
}

// Headers of request
func (ctx *HijackResponse) Headers() (http.Header, error) {
	res, err := ctx.req.Response()
	if err != nil {
		return nil, err
	}

	return res.Header, nil
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

// Body of response
func (ctx *HijackResponse) Body() ([]byte, error) {
	b, err := ctx.req.Bytes()
	if err != nil {
		return nil, err
	}

	return b, nil
}

// BodyStream returns the stream of the body
func (ctx *HijackResponse) BodyStream() (io.Reader, error) {
	res, err := ctx.req.Response()
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

// StringBody of response
func (ctx *HijackResponse) StringBody() string {
	return string(ctx.MustBody())
}

// JSONBody of response
func (ctx *HijackResponse) JSONBody() gjson.Result {
	return gjson.ParseBytes(ctx.MustBody())
}

// SetBody of response, if obj is []byte, raw body will be used, else it will be encoded as json
func (ctx *HijackResponse) SetBody(obj interface{}) *HijackResponse {
	switch body := obj.(type) {
	case []byte:
		ctx.payload.Body = body
	case string:
		ctx.payload.Body = []byte(body)
	default:
		ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
		var err error
		ctx.payload.Body, err = json.Marshal(obj)
		utils.E(err)
	}
	return ctx
}

// Fail request
func (ctx *HijackResponse) Fail(reason proto.NetworkErrorReason) *HijackResponse {
	ctx.fail.ErrorReason = reason
	return ctx
}

// GetDownloadFile of the next download url that matches the pattern, returns the file content.
// The handler will be used once and removed.
func (p *Page) GetDownloadFile(pattern string, resourceType proto.NetworkResourceType) func() (http.Header, io.Reader, error) {
	enable := p.DisableDomain(&proto.FetchEnable{})

	_ = proto.BrowserSetDownloadBehavior{
		Behavior:         proto.BrowserSetDownloadBehaviorBehaviorDeny,
		BrowserContextID: p.browser.BrowserContextID,
	}.Call(p)

	r := p.HijackRequests()

	ctx, cancel := context.WithCancel(p.ctx)
	downloading := &proto.PageDownloadWillBegin{}
	waitDownload := p.Context(ctx, cancel).WaitEvent(downloading)

	return func() (http.Header, io.Reader, error) {
		defer enable()
		defer cancel()

		defer func() {
			_ = proto.BrowserSetDownloadBehavior{
				Behavior:         proto.BrowserSetDownloadBehaviorBehaviorDefault,
				BrowserContextID: r.browser.BrowserContextID,
			}.Call(r.caller)
		}()

		var body io.Reader
		var header http.Header
		wg := &sync.WaitGroup{}
		wg.Add(1)

		var err error
		err = r.Add(pattern, resourceType, func(ctx *Hijack) {
			defer wg.Done()

			ctx.Skip = true

			ctx.OnError = func(e error) {
				err = e
			}

			err = ctx.LoadResponse(false)
			if err != nil {
				return
			}

			header, err = ctx.Response.Headers()
			if err != nil {
				return
			}

			body, err = ctx.Response.BodyStream()
		})
		if err != nil {
			return nil, nil, err
		}

		go r.Run()
		go func() {
			waitDownload()

			u := downloading.URL
			if strings.HasPrefix(u, "blob:") {
				res, e := p.EvalWithOptions(jsHelper(js.FetchAsDataURL, Array{u}))
				if e != nil {
					err = e
					wg.Done()
					return
				}
				u = res.Value.Str
			}

			if strings.HasPrefix(u, "data:") {
				t, d := parseDataURI(u)
				header = http.Header{"Content-Type": []string{t}}
				body = bytes.NewBuffer(d)
			} else {
				return
			}

			wg.Done()
		}()

		wg.Wait()
		r.MustStop()

		if err != nil {
			return nil, nil, err
		}

		return header, body, nil
	}
}

// HandleAuth for the next basic HTTP authentication.
// It will prevent the popup that requires user to input user name and password.
// Ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
func (b *Browser) HandleAuth(username, password string) func() error {
	enable := b.DisableDomain(b.ctx, "", &proto.FetchEnable{})
	disable := b.EnableDomain(b.ctx, "", &proto.FetchEnable{
		HandleAuthRequests: true,
	})

	paused := &proto.FetchRequestPaused{}
	auth := &proto.FetchAuthRequired{}

	waitPaused := b.WaitEvent(paused)
	waitAuth := b.WaitEvent(auth)

	return func() (err error) {
		defer enable()
		defer disable()

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
