package rod

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/goob"
	"github.com/ysmood/kit"
)

// Page implements the proto.Caller interface
var _ proto.Caller = &Page{}

// Page represents the webpage
// We try to hold as less states as possible
type Page struct {
	lock *sync.Mutex

	// these are the handler for ctx
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	browser *Browser

	TargetID  proto.TargetTargetID
	SessionID proto.TargetSessionID

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard

	element          *Element                    // iframe only
	windowObjectID   proto.RuntimeRemoteObjectID // used as the thisObject when eval js
	jsHelperObjectID proto.RuntimeRemoteObjectID
	executionIDs     map[proto.PageFrameID]proto.RuntimeExecutionContextID

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

// InfoE of the page, such as the URL or title of the page
func (p *Page) InfoE() (*proto.TargetTargetInfo, error) {
	return p.browser.pageInfo(p.TargetID)
}

// CookiesE returns the page cookies. By default it will return the cookies for current page.
// The urls is the list of URLs for which applicable cookies will be fetched.
func (p *Page) CookiesE(urls []string) ([]*proto.NetworkCookie, error) {
	if len(urls) == 0 {
		info, err := p.InfoE()
		if err != nil {
			return nil, err
		}
		urls = []string{info.URL}
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
// If req is nil, the default user agent will be the same as a mac chrome.
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
// If url is empty, it will navigate to "about:blank".
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
		return fmt.Errorf("%w: %s", newErr(ErrNavigation, res.ErrorText), res.ErrorText)
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

// ViewportE doc is similar to the method Viewport. If params is nil, it will clear the override.
func (p *Page) ViewportE(params *proto.EmulationSetDeviceMetricsOverride) error {
	if params == nil {
		return proto.EmulationClearDeviceMetricsOverride{}.Call(p)
	}
	return params.Call(p)
}

// EmulateE the device, such as iPhone9. If device is empty, it will clear the override.
func (p *Page) EmulateE(device devices.DeviceType, landscape bool) error {
	v := devices.GetViewport(device, landscape)
	u := devices.GetUserAgent(device)

	err := p.ViewportE(v)
	if err != nil {
		return err
	}

	return p.SetUserAgentE(u)
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

	p.cleanupStates()

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
	b := p.browser.Context(p.ctx, p.ctxCancel)
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
		finished *proto.NetworkLoadingFinished, // not use responseReceived because https://crbug.com/883475
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
	js, jsArgs := jsHelper("waitIdle", Array{timeout.Seconds()})
	_, err = p.EvalE(true, "", js, jsArgs)
	return err
}

// WaitLoadE doc is similar to the method WaitLoad
func (p *Page) WaitLoadE() error {
	js, jsArgs := jsHelper("waitLoad", nil)
	_, err := p.EvalE(true, "", js, jsArgs)
	return err
}

// AddScriptTagE to page. If url is empty, content will be used.
func (p *Page) AddScriptTagE(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	js, jsArgs := jsHelper("addScriptTag", Array{id, url, content})
	_, err := p.EvalE(true, "", js, jsArgs)
	return err
}

// AddStyleTagE to page. If url is empty, content will be used.
func (p *Page) AddStyleTagE(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	js, jsArgs := jsHelper("addStyleTag", Array{id, url, content})
	_, err := p.EvalE(true, "", js, jsArgs)
	return err
}

// EvalOnNewDocumentE Evaluates given script in every frame upon creation (before loading frame's scripts).
func (p *Page) EvalOnNewDocumentE(js string) (proto.PageScriptIdentifier, error) {
	res, err := proto.PageAddScriptToEvaluateOnNewDocument{Source: js}.Call(p)
	if err != nil {
		return "", err
	}

	return res.Identifier, nil
}

// EvalE thisID is the remote objectID that will be the this of the js function, if it's empty "window" will be used.
// Set the byValue to true to reduce memory occupation.
// If the item in jsArgs is proto.RuntimeRemoteObjectID, the remote object will be used, else the item will be treated as JSON value.
func (p *Page) EvalE(byValue bool, thisID proto.RuntimeRemoteObjectID, js string, jsArgs Array) (*proto.RuntimeRemoteObject, error) {
	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	objectID := thisID
	var err error
	var res *proto.RuntimeCallFunctionOnResult

	// js context will be invalid if a frame is reloaded
	err = kit.Retry(p.ctx, backoff, func() (bool, error) {
		if p.getWindowObjectID() == "" || thisID == "" {
			err := p.initJS(false)
			if err != nil {
				if isNilContextErr(err) {
					return false, nil
				}
				return true, err
			}
		}
		if thisID == "" {
			objectID = p.getWindowObjectID()
		}

		args := []*proto.RuntimeCallArgument{}
		for _, arg := range jsArgs {
			if id, ok := arg.(proto.RuntimeRemoteObjectID); ok {
				if id == jsHelperID {
					id = p.getJSHelperObjectID()
				}
				args = append(args, &proto.RuntimeCallArgument{Value: proto.NewJSON(nil), ObjectID: id})
			} else {
				args = append(args, &proto.RuntimeCallArgument{Value: proto.NewJSON(arg)})
			}
		}

		res, err = proto.RuntimeCallFunctionOn{
			ObjectID:            objectID,
			AwaitPromise:        true,
			ReturnByValue:       byValue,
			FunctionDeclaration: SprintFnThis(js),
			Arguments:           args,
		}.Call(p)
		if thisID == "" && isNilContextErr(err) {
			_ = p.initJS(true)
			return false, nil
		}

		return true, err
	})

	if err != nil {
		return nil, err
	}

	if res.ExceptionDetails != nil {
		exp := res.ExceptionDetails.Exception.Description
		return nil, fmt.Errorf("%w: %s", newErr(ErrEval, exp), exp)
	}

	return res.Result, nil
}

// WaitE js function until it returns true
func (p *Page) WaitE(sleeper kit.Sleeper, thisID proto.RuntimeRemoteObjectID, js string, params Array) error {
	if sleeper == nil {
		sleeper = func(_ context.Context) error {
			return fmt.Errorf("%w: %s", newErr(ErrWaitJSTimeout, js), js)
		}
	}

	removeTrace := func() {}
	defer removeTrace()

	return kit.Retry(p.ctx, sleeper, func() (bool, error) {
		remove := p.tryTraceFn(fmt.Sprintf("wait(%s)", js), params)
		removeTrace()
		removeTrace = remove

		res, err := p.EvalE(true, thisID, js, params)
		if err != nil {
			return true, err
		}

		return res.Value.Bool(), nil
	})
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

// ElementFromObject creates an Element from the remote object id.
func (p *Page) ElementFromObject(id proto.RuntimeRemoteObjectID) *Element {
	return (&Element{
		page:     p,
		ObjectID: id,
	}).Context(context.WithCancel(p.ctx))
}

// ElementFromNodeE creates an Element from the node id
func (p *Page) ElementFromNodeE(id proto.DOMNodeID) (*Element, error) {
	objID, err := p.resolveNode(id)
	if err != nil {
		return nil, err
	}

	el := p.ElementFromObject(objID)

	err = el.ensureParentPage(id, objID)
	if err != nil {
		return nil, err
	}

	// make sure always return an element node
	desc, err := el.DescribeE(0, false)
	if err != nil {
		return nil, err
	}
	if desc.NodeName == "#text" {
		el, err = el.ParentE()
		if err != nil {
			return nil, err
		}
	}

	return el, nil
}

// ElementFromPointE creates an Element from the absolute point on the page.
// The point should include the window scroll offset.
func (p *Page) ElementFromPointE(x, y int64) (*Element, error) {
	defer p.enableNodeQuery()()

	node, err := proto.DOMGetNodeForLocation{X: x, Y: y}.Call(p)
	if err != nil {
		return nil, err
	}

	return p.ElementFromNodeE(node.NodeID)
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

	err = proto.PageEnable{}.Call(p)
	if err != nil {
		return err
	}

	return nil
}

func (p *Page) initJS(force bool) error {
	contextID, err := p.getExecutionID(force)
	if err != nil {
		return err
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	if !force && p.windowObjectID != "" {
		return nil
	}

	window, err := proto.RuntimeEvaluate{
		Expression: "window",
		ContextID:  contextID,
	}.Call(p)
	if err != nil {
		return err
	}

	helper, err := proto.RuntimeCallFunctionOn{
		ObjectID:            window.Result.ObjectID,
		FunctionDeclaration: assets.Helper,
	}.Call(p)
	if err != nil {
		return err
	}

	p.windowObjectID = window.Result.ObjectID
	p.jsHelperObjectID = helper.Result.ObjectID

	return nil
}

func (p *Page) getExecutionID(force bool) (proto.RuntimeExecutionContextID, error) {
	if !p.IsIframe() {
		return 0, nil
	}

	frameID, err := p.frameID()
	if err != nil {
		return 0, err
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	if !force {
		if ctxID, has := p.executionIDs[frameID]; has {
			return ctxID, nil
		}
	}

	world, err := proto.PageCreateIsolatedWorld{
		FrameID:   frameID,
		WorldName: "rod_iframe_world",
	}.Call(p)
	if err != nil {
		return 0, err
	}

	p.executionIDs[frameID] = world.ExecutionContextID

	return world.ExecutionContextID, nil
}

func (p *Page) frameID() (proto.PageFrameID, error) {
	// this is the only way we can get the window object from the iframe
	if p.IsIframe() {
		node, err := p.element.DescribeE(1, false)
		if err != nil {
			return "", err
		}
		return node.FrameID, nil
	}

	res, err := proto.PageGetFrameTree{}.Call(p)
	if err != nil {
		return "", err
	}
	return res.FrameTree.Frame.ID, nil
}

func (p *Page) getWindowObjectID() proto.RuntimeRemoteObjectID {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.windowObjectID
}

func (p *Page) getJSHelperObjectID() proto.RuntimeRemoteObjectID {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.jsHelperObjectID
}

func (p *Page) enableNodeQuery() func() {
	recover := p.EnableDomain(&proto.DOMEnable{})

	// TODO: I don't know why we need this, seems like a bug of chrome.
	// We should remove it once chrome fixed this bug.
	_, _ = proto.DOMGetDocument{}.Call(p)

	return recover
}

func (p *Page) resolveNode(nodeID proto.DOMNodeID) (proto.RuntimeRemoteObjectID, error) {
	ctxID, err := p.getExecutionID(false)
	if err != nil {
		return "", err
	}

	node, err := proto.DOMResolveNode{
		NodeID:             nodeID,
		ExecutionContextID: ctxID,
	}.Call(p)
	if err != nil {
		return "", err
	}

	return node.Object.ObjectID, nil
}

func (p *Page) hasElement(id proto.RuntimeRemoteObjectID) (bool, error) {
	// We don't have a good way to detect if a node is inside an iframe.
	// Currently this is most efficient way to do it.
	_, err := p.EvalE(true, "", "() => {}", Array{id})
	if err == nil {
		return true, nil
	}
	if cdpErr, ok := err.(*cdp.Error); ok && cdpErr.Code == -32000 {
		return false, nil
	}
	return false, err
}
