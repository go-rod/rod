package rod

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/goob"
)

// Page implements the proto.Caller interface
var _ proto.Caller = &Page{}

// Page represents the webpage
// We try to hold as less states as possible
type Page struct {
	// these are the handler for ctx
	ctx     context.Context
	sleeper func() utils.Sleeper

	browser *Browser

	TargetID  proto.TargetTargetID
	SessionID proto.TargetSessionID
	FrameID   proto.PageFrameID

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard
	Touch    *Touch

	element          *Element                    // iframe only
	windowObjectID   proto.RuntimeRemoteObjectID // used as the thisObject when eval js
	jsHelperObjectID proto.RuntimeRemoteObjectID
	executionIDs     map[proto.PageFrameID]proto.RuntimeExecutionContextID
	jsContextLock    *sync.Mutex

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

// Info of the page, such as the URL or title of the page
func (p *Page) Info() (*proto.TargetTargetInfo, error) {
	return p.browser.pageInfo(p.TargetID)
}

// Cookies returns the page cookies. By default it will return the cookies for current page.
// The urls is the list of URLs for which applicable cookies will be fetched.
func (p *Page) Cookies(urls []string) ([]*proto.NetworkCookie, error) {
	if len(urls) == 0 {
		info, err := p.Info()
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

// SetCookies of the page.
func (p *Page) SetCookies(cookies []*proto.NetworkCookieParam) error {
	err := proto.NetworkSetCookies{Cookies: cookies}.Call(p)
	return err
}

// SetExtraHeaders whether to always send extra HTTP headers with the requests from this page.
func (p *Page) SetExtraHeaders(dict []string) (func(), error) {
	headers := proto.NetworkHeaders{}

	for i := 0; i < len(dict); i += 2 {
		headers[dict[i]] = proto.NewJSON(dict[i+1])
	}

	return p.EnableDomain(&proto.NetworkEnable{}), proto.NetworkSetExtraHTTPHeaders{Headers: headers}.Call(p)
}

// SetUserAgent (browser brand, accept-language, etc) of the page.
// If req is nil, a default user agent will be used, a typical mac chrome.
func (p *Page) SetUserAgent(req *proto.NetworkSetUserAgentOverride) error {
	if req == nil {
		req = &proto.NetworkSetUserAgentOverride{
			UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36",
			AcceptLanguage: "en",
			Platform:       "MacIntel",
		}
	}
	return req.Call(p)
}

// Navigate to the url. If the url is empty, "about:blank" will be used.
// It will return immediately after the server responds the http header.
func (p *Page) Navigate(url string) error {
	if url == "" {
		url = "about:blank"
	}

	err := p.StopLoading()
	if err != nil {
		return err
	}

	res, err := proto.PageNavigate{URL: url}.Call(p)
	if err != nil {
		return err
	}
	if res.ErrorText != "" {
		return newErr(ErrNavigation, res.ErrorText, res.ErrorText)
	}

	p.FrameID = res.FrameID

	return nil
}

// NavigateBack history.
func (p *Page) NavigateBack() error {
	// Not using cdp API because it doesn't work for iframe
	_, err := p.Eval(`history.back()`)
	return err
}

// NavigateForward history.
func (p *Page) NavigateForward() error {
	// Not using cdp API because it doesn't work for iframe
	_, err := p.Eval(`history.forward()`)
	return err
}

// Reload page.
func (p *Page) Reload() error {
	// Not using cdp API because it doesn't work for iframe
	_, err := p.Eval(`location.reload()`)
	return err
}

func (p *Page) getWindowID() (proto.BrowserWindowID, error) {
	res, err := proto.BrowserGetWindowForTarget{TargetID: p.TargetID}.Call(p)
	if err != nil {
		return 0, err
	}
	return res.WindowID, err
}

// GetWindow position and size info
func (p *Page) GetWindow() (*proto.BrowserBounds, error) {
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

// SetWindow location and size
func (p *Page) SetWindow(bounds *proto.BrowserBounds) error {
	id, err := p.getWindowID()
	if err != nil {
		return err
	}

	err = proto.BrowserSetWindowBounds{WindowID: id, Bounds: bounds}.Call(p)
	return err
}

// SetViewport overrides the values of device screen dimensions
func (p *Page) SetViewport(params *proto.EmulationSetDeviceMetricsOverride) error {
	if params == nil {
		return proto.EmulationClearDeviceMetricsOverride{}.Call(p)
	}
	return params.Call(p)
}

// Emulate the device, such as iPhone9. If device is devices.Clear, it will clear the override.
func (p *Page) Emulate(device devices.Device, landscape bool) error {
	err := p.SetViewport(device.Metrics(landscape))
	if err != nil {
		return err
	}

	err = device.Touch().Call(p)
	if err != nil {
		return err
	}

	return p.SetUserAgent(device.UserAgent())

}

// StopLoading forces the page stop navigation and pending resource fetches.
func (p *Page) StopLoading() error {
	return proto.PageStopLoading{}.Call(p)
}

// Close tries to close page, running its beforeunload hooks, if any.
func (p *Page) Close() error {
	p.browser.targetsLock.Lock()
	defer p.browser.targetsLock.Unlock()

	err := p.StopLoading()
	if err != nil {
		return err
	}

	success := true
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()

	wait := p.Context(ctx).EachEvent(func(e *proto.TargetDetachedFromTarget) bool {
		return e.TargetID == e.TargetID
	}, func(e *proto.PageJavascriptDialogClosed) bool {
		success = e.Result
		return !p.browser.headless && !success
	})

	err = proto.PageClose{}.Call(p)
	if err != nil {
		return err
	}

	wait()

	if success {
		p.cleanupStates()
	} else {
		return ErrPageCloseCanceled
	}

	return nil
}

// HandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload).
// Because alert will block js, usually you have to run the wait function in another goroutine.
func (p *Page) HandleDialog(accept bool, promptText string) func() error {
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

// Screenshot options: https://chromedevtools.github.io/devtools-protocol/tot/Page#method-captureScreenshot
func (p *Page) Screenshot(fullpage bool, req *proto.PageCaptureScreenshot) ([]byte, error) {
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

		err = p.SetViewport(&view)
		if err != nil {
			return nil, err
		}

		defer func() { // try to recover the viewport
			if !set {
				_ = proto.EmulationClearDeviceMetricsOverride{}.Call(p)
				return
			}

			_ = p.SetViewport(oldView)
		}()
	}

	shot, err := req.Call(p)
	if err != nil {
		return nil, err
	}
	return shot.Data, nil
}

// PDF prints page as PDF
func (p *Page) PDF(req *proto.PagePrintToPDF) (*StreamReader, error) {
	req.TransferMode = proto.PagePrintToPDFTransferModeReturnAsStream
	res, err := req.Call(p)
	if err != nil {
		return nil, err
	}

	return NewStreamReader(p, res.Stream), nil
}

// WaitOpen waits for the next new page opened by the current one
func (p *Page) WaitOpen() func() (*Page, error) {
	b := p.browser.Context(p.ctx)
	var targetID proto.TargetTargetID

	ctx, cancel := context.WithCancel(p.ctx)
	wait := b.Context(ctx).EachEvent(func(e *proto.TargetTargetCreated) bool {
		targetID = e.TargetInfo.TargetID
		return e.TargetInfo.OpenerID == p.TargetID
	})

	return func() (*Page, error) {
		defer cancel()
		wait()
		return b.PageFromTarget(targetID)
	}
}

// WaitPauseOpen waits for a page opened by the current page, before opening pause the js execution.
// Because the js will be paused, you should put the code that triggers it in a goroutine.
func (p *Page) WaitPauseOpen() (func() (*Page, error), func() error, error) {
	// TODO: we have to use the browser to call, seems like a chrome bug
	err := proto.TargetSetAutoAttach{
		AutoAttach:             true,
		WaitForDebuggerOnStart: true,
		Flatten:                true,
	}.Call(p.browser.Context(p.ctx))
	if err != nil {
		return nil, nil, err
	}

	return p.WaitOpen(), func() error {
		err = proto.TargetSetAutoAttach{
			Flatten: true,
		}.Call(p.browser.Context(p.ctx))
		if err != nil {
			return err
		}

		return proto.RuntimeRunIfWaitingForDebugger{}.Call(p)
	}, nil
}

// EachEvent of the specified event type, if any callback returns true the event loop will stop.
func (p *Page) EachEvent(callbacks ...interface{}) (wait func()) {
	return p.browser.eachEvent(p.ctx, p.SessionID, callbacks...)
}

// WaitEvent waits for the next event for one time. It will also load the data into the event object.
func (p *Page) WaitEvent(e proto.Payload) (wait func()) {
	return p.browser.waitEvent(p.ctx, p.SessionID, e)
}

// WaitNavigation wait for a page lifecycle event when navigating.
// Usually you will wait for proto.PageLifecycleEventNameNetworkAlmostIdle
func (p *Page) WaitNavigation(name proto.PageLifecycleEventName) func() {
	_ = proto.PageSetLifecycleEventsEnabled{Enabled: true}.Call(p)

	wait := p.EachEvent(func(e *proto.PageLifecycleEvent) bool {
		return e.Name == name
	})

	return func() {
		wait()
		_ = proto.PageSetLifecycleEventsEnabled{Enabled: false}.Call(p)
	}
}

// WaitRequestIdle returns a wait function that waits until no request for d duration.
// Be careful, d is not the max wait timeout, it's the least idle time.
// If you want to set a timeout you can use the "Page.Timeout" function.
// Use the includes and excludes regexp list to filter the requests by their url.
func (p *Page) WaitRequestIdle(d time.Duration, includes, excludes []string) func() {
	if len(includes) == 0 {
		includes = []string{""}
	}

	ctx, cancel := context.WithCancel(p.ctx)

	filter := genRegFilter(includes, excludes)
	reqList := &sync.Map{}

	var timeout *time.Timer

	reset := func(id proto.NetworkRequestID) {
		_, has := reqList.Load(id)
		if !has {
			return
		}

		reqList.Delete(id)

		// If there's no more on going requests, restart the stopped timer
		if utils.IsSyncMapEmpty(reqList) {
			timeout.Reset(d)
		}
	}

	wait := p.browser.eachEvent(ctx, p.SessionID, func(sent *proto.NetworkRequestWillBeSent) {
		if !filter(sent.Request.URL) {
			return
		}
		timeout.Stop()
		reqList.Store(sent.RequestID, sent.Request.URL)
	}, func(finished *proto.NetworkLoadingFinished) { // not use responseReceived because https://crbug.com/883475
		reset(finished.RequestID)
	}, func(failed *proto.NetworkLoadingFailed) {
		reset(failed.RequestID)
	})

	return func() {
		p.tryTraceReq(ctx, reqList, includes, excludes)
		timeout = time.NewTimer(d)

		go func() {
			<-timeout.C
			cancel()
		}()

		wait()
	}
}

// WaitIdle waits until the next window.requestIdleCallback is called.
func (p *Page) WaitIdle(timeout time.Duration) (err error) {
	_, err = p.EvalWithOptions(jsHelper(js.WaitIdle, JSArgs{timeout.Seconds()}))
	return err
}

// WaitLoad waits for the `window.onload` event, it returns immediately if the event is already fired.
func (p *Page) WaitLoad() error {
	_, err := p.EvalWithOptions(jsHelper(js.WaitLoad, nil))
	if err != nil {
		return err
	}

	// TODO: https://crbug.com/613219
	_, err = p.Root().Eval(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))`)
	return err
}

// AddScriptTag to page. If url is empty, content will be used.
func (p *Page) AddScriptTag(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.EvalWithOptions(jsHelper(js.AddScriptTag, JSArgs{id, url, content}))
	return err
}

// AddStyleTag to page. If url is empty, content will be used.
func (p *Page) AddStyleTag(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.EvalWithOptions(jsHelper(js.AddStyleTag, JSArgs{id, url, content}))
	return err
}

// EvalOnNewDocument Evaluates given script in every frame upon creation (before loading frame's scripts).
func (p *Page) EvalOnNewDocument(js string) (proto.PageScriptIdentifier, error) {
	res, err := proto.PageAddScriptToEvaluateOnNewDocument{Source: js}.Call(p)
	if err != nil {
		return "", err
	}

	return res.Identifier, nil
}

// Expose function to the page's window object. Must bind before navigate to the page. Bindings survive reloads.
// Binding function takes exactly one argument, this argument should be string.
func (p *Page) Expose(name string) (callback chan string, stop func(), err error) {
	err = proto.RuntimeAddBinding{Name: name}.Call(p)
	if err != nil {
		return
	}

	callback = make(chan string)
	ctx, cancel := context.WithCancel(p.ctx)
	stop = func() {
		cancel()
		_ = proto.RuntimeRemoveBinding{Name: name}.Call(p)
	}

	go p.EachEvent(func(e *proto.RuntimeBindingCalled) bool {
		if e.Name == name {
			select {
			case <-ctx.Done():
				return true
			case callback <- e.Payload:
			}
		}
		return false
	})()

	return
}

// Eval js on the page. It's just a shortcut for Page.EvalWithOptions.
func (p *Page) Eval(js string, jsArgs ...interface{}) (*proto.RuntimeRemoteObject, error) {
	return p.EvalWithOptions(NewEvalOptions(js, jsArgs))
}

// EvalWithOptions evaluates js on the page.
func (p *Page) EvalWithOptions(opts *EvalOptions) (*proto.RuntimeRemoteObject, error) {
	backoff := utils.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	objectID := opts.ThisID
	var err error
	var res *proto.RuntimeCallFunctionOnResult

	// js context will be invalid if a frame is reloaded or not ready, then the isNilContextErr
	// will be true, then we retry the eval again.
	err = utils.Retry(p.ctx, backoff, func() (bool, error) {
		if p.getWindowObjectID() == "" || opts.ThisID == "" {
			err := p.initJS(false)
			if err != nil {
				if isNilContextErr(err) {
					return false, nil
				}
				return true, err
			}
		}
		if opts.ThisID == "" {
			objectID = p.getWindowObjectID()
		}

		// construct arguments
		args := []*proto.RuntimeCallArgument{}
		for _, arg := range opts.JSArgs {
			if id, ok := arg.(proto.RuntimeRemoteObjectID); ok { // remote object
				if id == jsHelperID { // if it's a rod js helper object
					id = p.getJSHelperObjectID()
				}
				args = append(args, &proto.RuntimeCallArgument{Value: proto.NewJSON(nil), ObjectID: id})
			} else { // plain json data
				args = append(args, &proto.RuntimeCallArgument{Value: proto.NewJSON(arg)})
			}
		}

		res, err = proto.RuntimeCallFunctionOn{
			ObjectID:            objectID,
			AwaitPromise:        true,
			ReturnByValue:       opts.ByValue,
			UserGesture:         opts.UserGesture,
			FunctionDeclaration: SprintFnThis(opts.JS),
			Arguments:           args,
		}.Call(p)
		if opts.ThisID == "" && isNilContextErr(err) {
			_ = p.initJS(true)
			return false, nil
		}

		return true, err
	})

	if err != nil {
		return nil, err
	}

	if res.ExceptionDetails != nil {
		exp := res.ExceptionDetails.Exception
		return nil, newErr(ErrEval, exp, exp.Description+" "+exp.Value.String())
	}

	return res.Result, nil
}

// Wait js function until it returns true
func (p *Page) Wait(thisID proto.RuntimeRemoteObjectID, js string, params JSArgs) error {
	removeTrace := func() {}
	defer removeTrace()

	return utils.Retry(p.ctx, p.sleeper(), func() (bool, error) {
		remove := p.tryTraceEval(js, params)
		removeTrace()
		removeTrace = remove

		res, err := p.EvalWithOptions(NewEvalOptions(js, params).This(thisID))
		if err != nil {
			return true, err
		}

		return res.Value.Bool(), nil
	})
}

// ObjectToJSON by object id
func (p *Page) ObjectToJSON(obj *proto.RuntimeRemoteObject) (proto.JSON, error) {
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
		sleeper:  p.sleeper,
		page:     p,
		ObjectID: id,
	}).Context(p.ctx)
}

// ElementFromNode creates an Element from the node id
func (p *Page) ElementFromNode(id proto.DOMNodeID) (*Element, error) {
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
	desc, err := el.Describe(0, false)
	if err != nil {
		return nil, err
	}
	if desc.NodeName == "#text" {
		el, err = el.Parent()
		if err != nil {
			return nil, err
		}
	}

	return el, nil
}

// ElementFromPoint creates an Element from the absolute point on the page.
// The point should include the window scroll offset.
func (p *Page) ElementFromPoint(x, y int64) (*Element, error) {
	p.enableNodeQuery()

	node, err := proto.DOMGetNodeForLocation{X: x, Y: y}.Call(p)
	if err != nil {
		return nil, err
	}

	return p.ElementFromNode(node.NodeID)
}

// Release the remote object
func (p *Page) Release(objectID proto.RuntimeRemoteObjectID) error {
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

	// If we don't enable it, it will cause a lot of unexpected browser behavior.
	// Such as proto.PageAddScriptToEvaluateOnNewDocument won't work.
	p.EnableDomain(&proto.PageEnable{})

	// If we don't enable it, it will remove remote node id whenever we disable the domain
	// even after we re-enable it again we can't query the ids any more.
	p.EnableDomain(&proto.DOMEnable{})

	return nil
}

func (p *Page) initJS(force bool) error {
	contextID, err := p.getExecutionID(force)
	if err != nil {
		return err
	}

	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()

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

// We use this function to make sure every frame(page, iframe) will only have one IsolatedWorld.
func (p *Page) getExecutionID(force bool) (proto.RuntimeExecutionContextID, error) {
	if !p.IsIframe() {
		return 0, nil
	}

	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()

	if !force {
		if ctxID, has := p.executionIDs[p.FrameID]; has {
			_, err := proto.RuntimeEvaluate{ContextID: ctxID, Expression: `0`}.Call(p)
			if err == nil {
				return ctxID, nil
			} else if !isNilContextErr(err) {
				return 0, err
			}
		}
	}

	world, err := proto.PageCreateIsolatedWorld{
		FrameID:   p.FrameID,
		WorldName: "rod_iframe_world",
	}.Call(p)
	if err != nil {
		return 0, err
	}

	p.executionIDs[p.FrameID] = world.ExecutionContextID

	return world.ExecutionContextID, nil
}

func (p *Page) getWindowObjectID() proto.RuntimeRemoteObjectID {
	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()
	return p.windowObjectID
}

func (p *Page) getJSHelperObjectID() proto.RuntimeRemoteObjectID {
	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()
	return p.jsHelperObjectID
}

func (p *Page) enableNodeQuery() {
	// TODO: I don't know why we need this, seems like a bug of chrome.
	// We should remove it once chrome fixed this bug.
	_, _ = proto.DOMGetDocument{}.Call(p)
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
	_, err := p.Eval("() => {}", id)
	if err == nil {
		return true, nil
	}
	if cdpErr, ok := err.(*cdp.Error); ok && cdpErr.Code == -32000 {
		return false, nil
	}
	return false, err
}
