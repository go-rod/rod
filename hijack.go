package rod

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

// HijackRequests same as Page.HijackRequests, but can intercept requests of the entire browser.
func (b *Browser) HijackRequests() *HijackRouter {
	return newHijackRouter(b, b).initEvents()
}

// HijackRequests creates a new router instance for requests hijacking.
// When use Fetch domain outside the router should be stopped. Enabling hijacking disables page caching,
// but such as 304 Not Modified will still work as expected.
// The entire process of hijacking one request:
//
//	browser --req-> rod ---> server ---> rod --res-> browser
//
// The --req-> and --res-> are the parts that can be modified.
func (p *Page) HijackRequests() *HijackRouter {
	return newHijackRouter(p.browser, p).initEvents()
}

// HijackRouter context
type HijackRouter struct {
	run      func()
	stop     func()
	handlers []*hijackHandler
	enable   *proto.FetchEnable
	client   proto.Client
	browser  *Browser
}

func newHijackRouter(browser *Browser, client proto.Client) *HijackRouter {
	return &HijackRouter{
		enable:   &proto.FetchEnable{},
		browser:  browser,
		client:   client,
		handlers: []*hijackHandler{},
	}
}

func (r *HijackRouter) initEvents() *HijackRouter {
	ctx := r.browser.ctx
	if cta, ok := r.client.(proto.Contextable); ok {
		ctx = cta.GetContext()
	}

	var sessionID proto.TargetSessionID
	if tsa, ok := r.client.(proto.Sessionable); ok {
		sessionID = tsa.GetSessionID()
	}

	eventCtx, cancel := context.WithCancel(ctx)
	r.stop = cancel

	_ = r.enable.Call(r.client)

	r.run = r.browser.Context(eventCtx).eachEvent(sessionID, func(e *proto.FetchRequestPaused) bool {
		go func() {
			ctx := r.new(eventCtx, e)
			for _, h := range r.handlers {
				if !h.regexp.MatchString(e.Request.URL) {
					continue
				}

				h.handler(ctx)

				if ctx.continueRequest != nil {
					ctx.continueRequest.RequestID = e.RequestID
					err := ctx.continueRequest.Call(r.client)
					if err != nil {
						ctx.OnError(err)
					}
					return
				}

				if ctx.Skip {
					continue
				}

				if ctx.Response.fail.ErrorReason != "" {
					err := ctx.Response.fail.Call(r.client)
					if err != nil {
						ctx.OnError(err)
					}
					return
				}

				err := ctx.Response.payload.Call(r.client)
				if err != nil {
					ctx.OnError(err)
					return
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

	return r.enable.Call(r.client)
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

	return r.enable.Call(r.client)
}

// new context
func (r *HijackRouter) new(ctx context.Context, e *proto.FetchRequestPaused) *Hijack {
	headers := http.Header{}
	for k, v := range e.Request.Headers {
		headers[k] = []string{v.String()}
	}

	u, _ := url.Parse(e.Request.URL)

	req := &http.Request{
		Method: e.Request.Method,
		URL:    u,
		Body:   ioutil.NopCloser(strings.NewReader(e.Request.PostData)),
		Header: headers,
	}

	return &Hijack{
		Request: &HijackRequest{
			event: e,
			req:   req.WithContext(ctx),
		},
		Response: &HijackResponse{
			payload: &proto.FetchFulfillRequest{
				ResponseCode: 200,
				RequestID:    e.RequestID,
			},
			fail: &proto.FetchFailRequest{
				RequestID: e.RequestID,
			},
		},
		OnError: func(err error) {},

		browser: r.browser,
	}
}

// Run the router, after you call it, you shouldn't add new handler to it.
func (r *HijackRouter) Run() {
	r.run()
}

// Stop the router
func (r *HijackRouter) Stop() error {
	r.stop()
	return proto.FetchDisable{}.Call(r.client)
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

	// CustomState is used to store things for this context
	CustomState interface{}

	browser *Browser
}

// ContinueRequest without hijacking. The RequestID will be set by the router, you don't have to set it.
func (h *Hijack) ContinueRequest(cq *proto.FetchContinueRequest) {
	h.continueRequest = cq
}

// LoadResponse will send request to the real destination and load the response as default response to override.
func (h *Hijack) LoadResponse(client *http.Client, loadBody bool) error {
	res, err := client.Do(h.Request.req)
	if err != nil {
		return err
	}

	defer func() { _ = res.Body.Close() }()

	h.Response.payload.ResponseCode = res.StatusCode

	for k, vs := range res.Header {
		for _, v := range vs {
			h.Response.SetHeader(k, v)
		}
	}

	if loadBody {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		h.Response.payload.Body = b
	}

	return nil
}

// HijackRequest context
type HijackRequest struct {
	event *proto.FetchRequestPaused
	req   *http.Request
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
	u, _ := url.Parse(ctx.event.Request.URL)
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
func (ctx *HijackRequest) JSONBody() gson.JSON {
	return gson.NewFrom(ctx.Body())
}

// Req returns the underlying http.Request instance that will be used to send the request.
func (ctx *HijackRequest) Req() *http.Request {
	return ctx.req
}

// SetContext of the underlying http.Request instance
func (ctx *HijackRequest) SetContext(c context.Context) *HijackRequest {
	ctx.req = ctx.req.WithContext(c)
	return ctx
}

// SetBody of the request, if obj is []byte or string, raw body will be used, else it will be encoded as json.
func (ctx *HijackRequest) SetBody(obj interface{}) *HijackRequest {
	var b []byte

	switch body := obj.(type) {
	case []byte:
		b = body
	case string:
		b = []byte(body)
	default:
		b = utils.MustToJSONBytes(body)
	}

	ctx.req.Body = ioutil.NopCloser(bytes.NewBuffer(b))

	return ctx
}

// IsNavigation determines whether the request is a navigation request
func (ctx *HijackRequest) IsNavigation() bool {
	return ctx.Type() == proto.NetworkResourceTypeDocument
}

// HijackResponse context
type HijackResponse struct {
	payload *proto.FetchFulfillRequest
	fail    *proto.FetchFailRequest
}

// Payload to respond the request from the browser.
func (ctx *HijackResponse) Payload() *proto.FetchFulfillRequest {
	return ctx.payload
}

// Body of the payload
func (ctx *HijackResponse) Body() string {
	return string(ctx.payload.Body)
}

// Headers returns the clone of response headers.
// If you want to modify the response headers use HijackResponse.SetHeader .
func (ctx *HijackResponse) Headers() http.Header {
	header := http.Header{}

	for _, h := range ctx.payload.ResponseHeaders {
		header.Add(h.Name, h.Value)
	}

	return header
}

// SetHeader of the payload via key-value pairs
func (ctx *HijackResponse) SetHeader(pairs ...string) *HijackResponse {
	for i := 0; i < len(pairs); i += 2 {
		ctx.payload.ResponseHeaders = append(ctx.payload.ResponseHeaders, &proto.FetchHeaderEntry{
			Name:  pairs[i],
			Value: pairs[i+1],
		})
	}
	return ctx
}

// SetBody of the payload, if obj is []byte or string, raw body will be used, else it will be encoded as json.
func (ctx *HijackResponse) SetBody(obj interface{}) *HijackResponse {
	switch body := obj.(type) {
	case []byte:
		ctx.payload.Body = body
	case string:
		ctx.payload.Body = []byte(body)
	default:
		ctx.payload.Body = utils.MustToJSONBytes(body)
	}
	return ctx
}

// Fail request
func (ctx *HijackResponse) Fail(reason proto.NetworkErrorReason) *HijackResponse {
	ctx.fail.ErrorReason = reason
	return ctx
}

// HandleAuth for the next basic HTTP authentication.
// It will prevent the popup that requires user to input user name and password.
// Ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
func (b *Browser) HandleAuth(username, password string) func() error {
	enable := b.DisableDomain("", &proto.FetchEnable{})
	disable := b.EnableDomain("", &proto.FetchEnable{
		HandleAuthRequests: true,
	})

	paused := &proto.FetchRequestPaused{}
	auth := &proto.FetchAuthRequired{}

	ctx, cancel := context.WithCancel(b.ctx)
	waitPaused := b.Context(ctx).WaitEvent(paused)
	waitAuth := b.Context(ctx).WaitEvent(auth)

	return func() (err error) {
		defer enable()
		defer disable()
		defer cancel()

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
