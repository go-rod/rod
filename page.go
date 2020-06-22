package rod

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/ysmood/goob"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/assets"
	"github.com/ysmood/rod/lib/proto"
)

// Page implements the proto.Caller interface
var _ proto.Caller = &Page{}

// Page represents the webpage
type Page struct {
	// these are the handler for ctx
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	browser *Browser

	TargetID  proto.TargetTargetID
	SessionID proto.TargetSessionID
	FrameID   proto.PageFrameID

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard

	element        *Element                    // iframe only
	windowObjectID proto.RuntimeRemoteObjectID // used as the thisObject when eval js

	event *goob.Observable
}

// IsIframe tells if it's iframe
func (p *Page) IsIframe() bool {
	return p.element != nil
}

// Root page of the iframe, if it's not a iframe returns itself
func (p *Page) Root() *Page {
	f := p

	for f.IsIframe() {
		f = f.element.page
	}

	return f
}

// CookiesE returns the page cookies. By default it will return the cookies for current page.
// The urls is the list of URLs for which applicable cookies will be fetched.
func (p *Page) CookiesE(urls []string) ([]*proto.NetworkCookie, error) {
	if len(urls) == 0 {
		info, err := proto.TargetGetTargetInfo{TargetID: p.TargetID}.Call(p)
		if err != nil {
			return nil, err
		}
		urls = []string{info.TargetInfo.URL}
	}

	res, err := proto.NetworkGetCookies{Urls: urls}.Call(p)
	if err != nil {
		return nil, err
	}
	return res.Cookies, nil
}

// SetCookiesE of the page.
// Cookie format: https://chromedevtools.github.io/devtools-protocol/tot/Network#method-setCookie
func (p *Page) SetCookiesE(cookies []*proto.NetworkCookieParam) error {
	err := proto.NetworkSetCookies{Cookies: cookies}.Call(p)
	return err
}

// SetExtraHeadersE whether to always send extra HTTP headers with the requests from this page.
func (p *Page) SetExtraHeadersE(dict []string) (func(), error) {
	headers := proto.NetworkHeaders{}

	for i := 0; i < len(dict); i += 2 {
		headers[dict[i]] = proto.NewJSON(dict[i+1])
	}

	return p.EnableDomain(&proto.NetworkEnable{}), proto.NetworkSetExtraHTTPHeaders{Headers: headers}.Call(p)
}

// SetUserAgentE Allows overriding user agent with the given string.
func (p *Page) SetUserAgentE(req *proto.NetworkSetUserAgentOverride) error {
	if req == nil {
		req = &proto.NetworkSetUserAgentOverride{
			UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36",
			AcceptLanguage: "en",
			Platform:       "MacIntel",
		}
	}
	return req.Call(p)
}

// NavigateE doc is similar to the method Navigate
func (p *Page) NavigateE(url string) error {
	if url == "" {
		url = "about:blank"
	}

	err := p.StopLoadingE()
	if err != nil {
		return err
	}
	res, err := proto.PageNavigate{URL: url}.Call(p)
	if err != nil {
		return err
	}
	if res.ErrorText != "" {
		return &Error{Code: ErrNavigation, Details: res.ErrorText}
	}
	return nil
}

func (p *Page) getWindowID() (proto.BrowserWindowID, error) {
	res, err := proto.BrowserGetWindowForTarget{TargetID: p.TargetID}.Call(p)
	if err != nil {
		return 0, err
	}
	return res.WindowID, err
}

// GetWindowE doc is similar to the method GetWindow
func (p *Page) GetWindowE() (*proto.BrowserBounds, error) {
	id, err := p.getWindowID()
	if err != nil {
		return nil, err
	}

	res, err := proto.BrowserGetWindowBounds{WindowID: id}.Call(p)
	if err != nil {
		return nil, err
	}

	return res.Bounds, nil
}

// WindowE https://chromedevtools.github.io/devtools-protocol/tot/Browser#type-Bounds
func (p *Page) WindowE(bounds *proto.BrowserBounds) error {
	id, err := p.getWindowID()
	if err != nil {
		return err
	}

	err = proto.BrowserSetWindowBounds{WindowID: id, Bounds: bounds}.Call(p)
	return err
}

// ViewportE doc is similar to the method Viewport
func (p *Page) ViewportE(params *proto.EmulationSetDeviceMetricsOverride) error {
	err := params.Call(p)
	return err
}

// StopLoadingE forces the page stop navigation and pending resource fetches.
func (p *Page) StopLoadingE() error {
	return proto.PageStopLoading{}.Call(p)
}

// CloseE page
func (p *Page) CloseE() error {
	err := p.StopLoadingE()
	if err != nil {
		return err
	}
	err = proto.PageClose{}.Call(p)
	if err != nil {
		return err
	}

	p.ctxCancel()
	return nil
}

// HandleDialogE doc is similar to the method HandleDialog
func (p *Page) HandleDialogE(accept bool, promptText string) func() error {
	recover := p.EnableDomain(&proto.PageEnable{})

	wait := p.WaitEvent(&proto.PageJavascriptDialogOpening{})

	return func() error {
		defer recover()

		wait()
		return proto.PageHandleJavaScriptDialog{
			Accept:     accept,
			PromptText: promptText,
		}.Call(p)
	}
}

// GetDownloadFileE how it works is to proxy the request, the dir is the dir to save the file.
func (p *Page) GetDownloadFileE(pattern string) (func() (http.Header, []byte, error), error) {
	err := proto.BrowserSetDownloadBehavior{
		Behavior:         proto.BrowserSetDownloadBehaviorBehaviorDeny,
		BrowserContextID: p.browser.BrowserContextID,
	}.Call(p.browser)
	if err != nil {
		return nil, err
	}

	var fetchEnable *proto.FetchEnable
	if pattern != "" {
		fetchEnable = &proto.FetchEnable{
			Patterns: []*proto.FetchRequestPattern{
				{URLPattern: pattern},
			},
		}
	}
	recover := p.EnableDomain(fetchEnable)

	msgReq := &proto.FetchRequestPaused{}
	wait := p.WaitEvent(msgReq)

	return func() (http.Header, []byte, error) {
		defer recover()

		wait()

		req := kit.Req(msgReq.Request.URL).Context(p.ctx)

		for k, v := range msgReq.Request.Headers {
			req.Header(k, v.String())
		}

		res, err := req.Response()
		if err != nil {
			return nil, nil, err
		}

		body, err := req.Bytes()
		if err != nil {
			return nil, nil, err
		}

		headers := []*proto.FetchHeaderEntry{}
		for k, vs := range res.Header {
			for _, v := range vs {
				headers = append(headers, &proto.FetchHeaderEntry{Name: k, Value: v})
			}
		}

		err = proto.FetchFulfillRequest{
			RequestID:       msgReq.RequestID,
			ResponseCode:    int64(res.StatusCode),
			ResponseHeaders: headers,
			Body:            body,
		}.Call(p)
		if err != nil {
			return nil, nil, err
		}

		return res.Header, body, nil
	}, nil
}

// ScreenshotE options: https://chromedevtools.github.io/devtools-protocol/tot/Page#method-captureScreenshot
func (p *Page) ScreenshotE(fullpage bool, req *proto.PageCaptureScreenshot) ([]byte, error) {
	if fullpage {
		metrics, err := proto.PageGetLayoutMetrics{}.Call(p)
		if err != nil {
			return nil, err
		}

		oldView := &proto.EmulationSetDeviceMetricsOverride{}
		set := p.LoadState(oldView)
		view := *oldView
		view.Width = int64(metrics.ContentSize.Width)
		view.Height = int64(metrics.ContentSize.Height)

		err = p.ViewportE(&view)
		if err != nil {
			return nil, err
		}
		defer func() {
			if !set {
				e := proto.EmulationClearDeviceMetricsOverride{}.Call(p)
				if err == nil {
					err = e
				}
				return
			}

			e := p.ViewportE(oldView)
			if err == nil {
				err = e
			}
		}()
	}

	shot, err := req.Call(p)
	if err != nil {
		return nil, err
	}
	return shot.Data, nil
}

// PDFE prints page as PDF
func (p *Page) PDFE(req *proto.PagePrintToPDF) ([]byte, error) {
	res, err := req.Call(p)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// WaitOpenE doc is similar to the method WaitPage
func (p *Page) WaitOpenE() func() (*Page, error) {
	b := p.browser.Context(p.ctx)
	var targetID proto.TargetTargetID

	wait := b.EachEvent(func(e *proto.TargetTargetCreated) bool {
		if e.TargetInfo.OpenerID == p.TargetID {
			targetID = e.TargetInfo.TargetID
			return true
		}
		return false
	})

	return func() (*Page, error) {
		wait()
		return b.PageFromTargetIDE(targetID)
	}
}

// PauseE doc is similar to the method Pause
func (p *Page) PauseE() error {
	wait := p.WaitEvent(&proto.DebuggerResumed{})
	err := proto.DebuggerPause{}.Call(p)
	if err != nil {
		return err
	}
	wait()
	return nil
}

// EachEvent of the specified event type, if the fn returns true the event loop will stop.
// The fn can accpet multiple events, such as EachEventE(func(e1 *proto.PageLoadEventFired, e2 *proto.PageLifecycleEvent) {}),
// only one argument will be non-null, others will null.
func (p *Page) EachEvent(fn interface{}) (wait func()) {
	return p.browser.eachEvent(p.ctx, p.SessionID, fn)
}

// WaitEvent waits for the next event for one time. It will also load the data into the event object.
func (p *Page) WaitEvent(e proto.Payload) (wait func()) {
	return p.browser.waitEvent(p.ctx, p.SessionID, e)
}

// WaitRequestIdleE returns a wait function that waits until no request for d duration.
// Use the includes and excludes regexp list to filter the requests by their url.
// Such as set n to 1 if there's a polling request.
func (p *Page) WaitRequestIdleE(d time.Duration, includes, excludes []string) func() {
	ctx, cancel := context.WithCancel(p.ctx)

	reqList := map[proto.NetworkRequestID]kit.Nil{}
	timeout := time.NewTimer(d)
	timeout.Stop()

	reset := func(id proto.NetworkRequestID) {
		delete(reqList, id)
		if len(reqList) == 0 {
			timeout.Reset(d)
		}
	}

	go func() {
		<-timeout.C
		cancel()
	}()

	wait := p.browser.eachEvent(ctx, p.SessionID, func(
		sent *proto.NetworkRequestWillBeSent,
		finished *proto.NetworkLoadingFinished,
		failed *proto.NetworkLoadingFailed,
	) {
		if sent != nil {
			timeout.Stop()
			url := sent.Request.URL
			id := sent.RequestID
			if matchWithFilter(url, includes, excludes) {
				reqList[id] = kit.Nil{}
			}
		} else if finished != nil {
			reset(finished.RequestID)
		} else if failed != nil {
			reset(failed.RequestID)
		}
	})

	return func() {
		if p.browser.trace {
			defer p.Overlay(0, 0, 300, 0, "waiting for request idle "+strings.Join(includes, " "))()
		}
		timeout.Reset(d)
		wait()
	}
}

// WaitIdleE doc is similar to the method WaitIdle
func (p *Page) WaitIdleE(timeout time.Duration) (err error) {
	_, err = p.EvalE(true, "", p.jsFn("waitIdle"), Array{timeout.Seconds()})
	return err
}

// WaitLoadE doc is similar to the method WaitLoad
func (p *Page) WaitLoadE() error {
	_, err := p.EvalE(true, "", p.jsFn("waitLoad"), nil)
	return err
}

// AddScriptTagE to page. If url is empty, content will be used.
func (p *Page) AddScriptTagE(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.EvalE(true, "", p.jsFn("addScriptTag"), Array{id, url, content})
	return err
}

// AddStyleTagE to page. If url is empty, content will be used.
func (p *Page) AddStyleTagE(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.EvalE(true, "", p.jsFn("addStyleTag"), Array{id, url, content})
	return err
}

// EvalE thisID is the remote objectID that will be the this of the js function, if it's empty "window" will be used.
// Set the byValue to true to reduce memory occupation.
func (p *Page) EvalE(byValue bool, thisID proto.RuntimeRemoteObjectID, js string, jsArgs Array) (*proto.RuntimeRemoteObject, error) {
	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	objectID := thisID
	var err error
	var res *proto.RuntimeCallFunctionOnResult

	// js context will be invalid if a frame is reloaded
	err = kit.Retry(p.ctx, backoff, func() (bool, error) {
		if thisID == "" {
			if p.windowObjectID == "" {
				err := p.initJS()
				if err != nil {
					if isNilContextErr(err) {
						return false, nil
					}
					return true, err
				}
			}
			objectID = p.windowObjectID
		}

		args := []*proto.RuntimeCallArgument{}
		for _, p := range jsArgs {
			args = append(args, &proto.RuntimeCallArgument{Value: proto.NewJSON(p)})
		}

		res, err = proto.RuntimeCallFunctionOn{
			ObjectID:            objectID,
			AwaitPromise:        true,
			ReturnByValue:       byValue,
			FunctionDeclaration: SprintFnThis(js),
			Arguments:           args,
		}.Call(p)

		if thisID == "" {
			if isNilContextErr(err) {
				_ = p.initJS()
				return false, nil
			}
		}

		return true, err
	})

	if err != nil {
		return nil, err
	}

	if res.ExceptionDetails != nil {
		return nil, &Error{nil, ErrEval, res.ExceptionDetails.Exception.Description}
	}

	return res.Result, nil
}

// ObjectToJSONE by object id
func (p *Page) ObjectToJSONE(obj *proto.RuntimeRemoteObject) (proto.JSON, error) {
	if obj.ObjectID == "" {
		return obj.Value, nil
	}

	res, err := proto.RuntimeCallFunctionOn{
		ObjectID:            obj.ObjectID,
		FunctionDeclaration: `function() { return this }`,
		ReturnByValue:       true,
	}.Call(p)
	if err != nil {
		return proto.JSON{}, err
	}
	return res.Result.Value, nil
}

// Sleeper returns the default sleeper for retry, it uses backoff and requestIdleCallback to wait
func (p *Page) Sleeper() kit.Sleeper {
	return kit.BackoffSleeper(100*time.Millisecond, time.Second, nil)
}

// ElementFromObjectID creates an Element from the remote object id.
func (p *Page) ElementFromObjectID(id proto.RuntimeRemoteObjectID) *Element {
	return (&Element{
		page:     p,
		ObjectID: id,
	}).Context(p.ctx)
}

// ReleaseE doc is similar to the method Release
func (p *Page) ReleaseE(objectID proto.RuntimeRemoteObjectID) error {
	err := proto.RuntimeReleaseObject{ObjectID: objectID}.Call(p)
	return err
}

// CallContext parameters for proto
func (p *Page) CallContext() (context.Context, proto.Client, string) {
	return p.ctx, p.browser, string(p.SessionID)
}

func (p *Page) initSession() error {
	obj, err := proto.TargetAttachToTarget{
		TargetID: p.TargetID,
		Flatten:  true, // if it's not set no response will return
	}.Call(p)
	if err != nil {
		return err
	}
	p.SessionID = obj.SessionID

	res, err := proto.DOMGetDocument{}.Call(p)
	if err != nil {
		return err
	}

	for _, child := range res.Root.Children {
		frameID := child.FrameID
		if frameID != "" {
			p.FrameID = frameID
		}
	}

	return nil
}

func (p *Page) initJS() error {
	scriptURL := "\n//# sourceURL=__rod_helper__"

	params := &proto.RuntimeEvaluate{
		Expression: sprintFnApply(assets.Helper, Array{p.FrameID}) + scriptURL,
	}

	if p.IsIframe() {
		res, err := proto.PageCreateIsolatedWorld{
			FrameID: p.FrameID,
		}.Call(p)
		if err != nil {
			return err
		}

		params.ContextID = res.ExecutionContextID
	}

	res, err := params.Call(p)
	if err != nil {
		return err
	}

	p.windowObjectID = res.Result.ObjectID

	if p.browser.trace {
		_, err := p.EvalE(true, "", p.jsFn("initMouseTracer"), Array{p.Mouse.id, assets.MousePointer})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Page) jsFnPrefix() string {
	return "rod" + string(p.FrameID) + "."
}

func (p *Page) jsFn(fnName string) string {
	return p.jsFnPrefix() + fnName
}
