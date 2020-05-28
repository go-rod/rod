package rod

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/assets"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/proto"
)

// Page implements the proto.Caller interface
var _ proto.Caller = &Page{}

// Page represents the webpage
type Page struct {
	// these are the handler for ctx
	ctx           context.Context
	timeoutCancel func()

	browser *Browser

	TargetID  proto.TargetTargetID
	SessionID proto.TargetSessionID
	FrameID   proto.PageFrameID

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard

	element             *Element                    // iframe only
	windowObjectID      proto.RuntimeRemoteObjectID // used as the thisObject when eval js
	getDownloadFileLock *sync.Mutex
	viewport            *proto.EmulationSetDeviceMetricsOverride

	event *kit.Observable
}

// IsIframe tells if it's iframe
func (p *Page) IsIframe() bool {
	return p.element != nil
}

// Event returns the observable for page events
func (p *Page) Event() *kit.Observable {
	return p.event
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
func (p *Page) SetExtraHeadersE(dict []string) error {
	headers := proto.NetworkHeaders{}

	for i := 0; i < len(dict); i += 2 {
		headers[dict[i]] = proto.NewJSON(dict[i+1])
	}

	return proto.NetworkSetExtraHTTPHeaders{Headers: headers}.Call(p)
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

// NavigateE doc is the same as the method Navigate
func (p *Page) NavigateE(url string) error {
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

// GetWindowE doc is the same as the method GetWindow
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

// ViewportE doc is the same as the method Viewport
func (p *Page) ViewportE(params *proto.EmulationSetDeviceMetricsOverride) error {
	p.viewport = params
	err := params.Call(p)
	return err
}

// StopLoadingE forces the page stop all navigations and pending resource fetches.
func (p *Page) StopLoadingE() error {
	return proto.PageStopLoading{}.Call(p)
}

// CloseE page
func (p *Page) CloseE() error {
	err := p.StopLoadingE()
	if err != nil {
		return err
	}
	return proto.PageClose{}.Call(p)
}

// HandleDialogE doc is the same as the method HandleDialog
func (p *Page) HandleDialogE(accept bool, promptText string) func() error {
	wait := p.WaitEventE(NewEventFilter(&proto.PageJavascriptDialogOpening{}))

	return func() error {
		err := <-wait
		if err != nil {
			return err
		}
		err = proto.PageHandleJavaScriptDialog{
			Accept:     accept,
			PromptText: promptText,
		}.Call(p)
		return err
	}
}

// GetDownloadFileE how it works is to proxy the request, the dir is the dir to save the file.
func (p *Page) GetDownloadFileE(dir, pattern string) (func() (http.Header, []byte, error), error) {
	var fetchEnable *proto.FetchEnable
	if pattern != "" {
		fetchEnable = &proto.FetchEnable{
			Patterns: []*proto.FetchRequestPattern{
				{URLPattern: pattern},
			},
		}
	}

	// both Page.setDownloadBehavior and Fetch.enable will pollute the global status,
	// we have to prevent race condition here
	p.getDownloadFileLock.Lock()

	err := proto.PageSetDownloadBehavior{
		Behavior:     proto.PageSetDownloadBehaviorBehaviorAllow,
		DownloadPath: dir,
	}.Call(p)
	if err != nil {
		return nil, err
	}

	err = fetchEnable.Call(p)
	if err != nil {
		return nil, err
	}

	msgReq := &proto.FetchRequestPaused{}
	wait := p.WaitEventE(NewEventFilter(msgReq))

	return func() (http.Header, []byte, error) {
		defer p.getDownloadFileLock.Unlock()

		var err error
		defer func() {
			e := proto.FetchDisable{}.Call(p)
			if err == nil {
				err = e
			}
		}()

		err = <-wait
		if err != nil {
			return nil, nil, err
		}

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
	}, err
}

// ScreenshotE options: https://chromedevtools.github.io/devtools-protocol/tot/Page#method-captureScreenshot
func (p *Page) ScreenshotE(fullpage bool, req *proto.PageCaptureScreenshot) ([]byte, error) {
	if fullpage {
		metrics, err := proto.PageGetLayoutMetrics{}.Call(p)
		if err != nil {
			return nil, err
		}

		oldView := p.viewport
		view := *oldView
		view.Width = int64(metrics.ContentSize.Width)
		view.Height = int64(metrics.ContentSize.Height)

		err = p.ViewportE(&view)
		if err != nil {
			return nil, err
		}
		defer func() {
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

// WaitPageE doc is the same as the method WaitPage
func (p *Page) WaitPageE() func() (*Page, error) {
	var targetInfo *proto.TargetTargetInfo

	wait := p.browser.Context(p.ctx).WaitEventE(func(e *cdp.Event) bool {
		event := &proto.TargetTargetCreated{}
		if e.Method == event.MethodName() {
			kit.E(json.Unmarshal(e.Params, event))
			targetInfo = event.TargetInfo

			if targetInfo.OpenerID == p.TargetID {
				return true
			}
		}
		return false
	})

	return func() (*Page, error) {
		err := <-wait
		if err != nil {
			return nil, err
		}
		return p.browser.Context(p.ctx).PageFromTargetIDE(targetInfo.TargetID)
	}
}

// PauseE doc is the same as the method Pause
func (p *Page) PauseE() error {
	_, err := proto.DebuggerEnable{}.Call(p)
	if err != nil {
		return err
	}
	err = proto.DebuggerPause{}.Call(p)
	if err != nil {
		return err
	}
	wait := p.WaitEventE(NewEventFilter(&proto.DebuggerResumed{}))
	return <-wait
}

// WaitRequestIdleE returns a wait function that waits until no request for d duration.
// Use the includes and excludes regexp list to filter the requests by their url.
// Such as set n to 1 if there's a polling request.
func (p *Page) WaitRequestIdleE(d time.Duration, includes, excludes []string) func() error {
	s := p.event.Subscribe()
	done := make(chan error)

	go func() {
		defer p.browser.event.Unsubscribe(s)

		reqList := map[proto.NetworkRequestID]kit.Nil{}
		timeout := time.NewTimer(d)

		reset := func(id proto.NetworkRequestID) {
			delete(reqList, id)
			if len(reqList) == 0 {
				timeout.Reset(d)
			}
		}

		for {
			select {
			case <-p.ctx.Done():
				done <- p.ctx.Err()
				return
			case <-timeout.C:
				done <- nil
				return
			case msg, ok := <-s.C:
				if !ok {
					done <- nil
					return
				}

				e := msg.(*cdp.Event)
				sent := &proto.NetworkRequestWillBeSent{}
				finished := &proto.NetworkLoadingFinished{}
				failed := &proto.NetworkLoadingFailed{}
				received := &proto.NetworkResponseReceived{}

				if Event(e, sent) {
					timeout.Stop()
					kit.E(json.Unmarshal(e.Params, sent))
					url := sent.Request.URL
					id := sent.RequestID
					if matchWithFilter(url, includes, excludes) {
						reqList[id] = kit.Nil{}
					}
				} else if Event(e, finished) {
					kit.E(json.Unmarshal(e.Params, finished))
					reset(finished.RequestID)
				} else if Event(e, failed) {
					kit.E(json.Unmarshal(e.Params, failed))
					reset(failed.RequestID)
				} else if Event(e, received) {
					kit.E(json.Unmarshal(e.Params, received))
					reset(received.RequestID)
				}
			}
		}
	}()

	return func() (err error) {
		defer func() { done = nil }()
		if done == nil {
			panic("can't use wait function twice")
		}

		if p.browser.trace {
			defer p.Overlay(0, 0, 300, 0, "waiting for request idle "+strings.Join(includes, " "))()
		}

		return <-done
	}
}

// WaitIdleE doc is the same as the method WaitIdle
func (p *Page) WaitIdleE(timeout time.Duration) (err error) {
	_, err = p.EvalE(true, "", p.jsFn("waitIdle"), Array{timeout.Seconds()})
	return err
}

// WaitLoadE doc is the same as the method WaitLoad
func (p *Page) WaitLoadE() error {
	_, err := p.EvalE(true, "", p.jsFn("waitLoad"), nil)
	return err
}

// WaitEventE doc is the same as the method WaitEvent
func (p *Page) WaitEventE(filter EventFilter) <-chan error {
	return p.browser.Context(p.ctx).WaitEventE(func(e *cdp.Event) bool {
		return e.SessionID == string(p.SessionID) && filter(e)
	})
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

// Sleeper returns the default sleeper for retry, it uses backoff and requestIdleCallback to wait
func (p *Page) Sleeper() kit.Sleeper {
	return kit.BackoffSleeper(100*time.Millisecond, time.Second, nil)
}

// ElementFromObjectID creates an Element from the remote object id.
func (p *Page) ElementFromObjectID(id proto.RuntimeRemoteObjectID) *Element {
	return &Element{
		page:     p,
		ctx:      p.ctx,
		ObjectID: id,
	}
}

// ReleaseE doc is the same as the method Release
func (p *Page) ReleaseE(objectID proto.RuntimeRemoteObjectID) error {
	err := proto.RuntimeReleaseObject{ObjectID: objectID}.Call(p)
	return err
}

// CallContext parameters for proto
func (p *Page) CallContext() (context.Context, proto.Client, string) {
	return p.ctx, p.browser.client, string(p.SessionID)
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

	err = p.initEvents()
	if err != nil {
		return err
	}

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

func (p *Page) initEvents() error {
	p.event = kit.NewObservable()

	go func() {
		for msg := range p.browser.event.Subscribe().C {
			if msg.(*cdp.Event).SessionID == string(p.SessionID) {
				p.event.Publish(msg)
			}
		}
	}()

	err := proto.PageEnable{}.Call(p)
	if err != nil {
		return err
	}

	err = proto.NetworkEnable{}.Call(p)
	if err != nil {
		return err
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

	return nil
}

func (p *Page) jsFnPrefix() string {
	return "rod" + string(p.FrameID) + "."
}

func (p *Page) jsFn(fnName string) string {
	return p.jsFnPrefix() + fnName
}
