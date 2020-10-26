package rod

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

// Page implements these interfaces
var _ proto.Client = &Page{}
var _ proto.Contextable = &Page{}
var _ proto.Sessionable = &Page{}

// Page represents the webpage
// We try to hold as less states as possible
type Page struct {
	TargetID  proto.TargetTargetID
	SessionID proto.TargetSessionID
	FrameID   proto.PageFrameID

	ctx context.Context

	root *Page

	sleeper func() utils.Sleeper

	browser *Browser

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard
	Touch    *Touch

	element *Element // iframe only

	jsCtxLock *sync.Mutex
	jsCtxID   *proto.RuntimeExecutionContextID // use pointer so that page clones can share the change
	helpers   map[proto.RuntimeExecutionContextID]map[string]proto.RuntimeRemoteObjectID
}

// IsIframe tells if it's iframe
func (p *Page) IsIframe() bool {
	return p.element != nil
}

// GetSessionID interface
func (p *Page) GetSessionID() proto.TargetSessionID {
	return p.SessionID
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
		headers[dict[i]] = gson.New(dict[i+1])
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
		return &ErrNavigation{res.ErrorText}
	}

	return p.root.updateJSCtxID()
}

// NavigateBack history.
func (p *Page) NavigateBack() error {
	// Not using cdp API because it doesn't work for iframe
	_, err := p.Evaluate(Eval(`history.back()`).ByUser())
	return err
}

// NavigateForward history.
func (p *Page) NavigateForward() error {
	// Not using cdp API because it doesn't work for iframe
	_, err := p.Evaluate(Eval(`history.forward()`).ByUser())
	return err
}

// Reload page.
func (p *Page) Reload() error {
	p, cancel := p.WithCancel()
	defer cancel()

	wait := p.EachEvent(func(e *proto.PageFrameNavigated) bool {
		return e.Frame.ID == p.FrameID
	})

	// Not using cdp API because it doesn't work for iframe
	_, err := p.Evaluate(Eval(`location.reload()`).ByUser())
	if err != nil {
		return err
	}

	wait()
	return p.updateJSCtxID()
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

	success := true
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()
	messages := p.browser.Context(ctx).Event()

	err := proto.PageClose{}.Call(p)
	if err != nil {
		return err
	}

	for msg := range messages {
		stop := false

		destroyed := proto.TargetTargetDestroyed{}
		closed := proto.PageJavascriptDialogClosed{}
		if msg.Load(&destroyed) {
			stop = destroyed.TargetID == p.TargetID
		} else if msg.SessionID == p.SessionID && msg.Load(&closed) {
			success = closed.Result
			stop = !p.browser.headless && !success
		}

		if stop {
			break
		}
	}

	if success {
		p.cleanupStates()
	} else {
		return &ErrPageCloseCanceled{}
	}

	return nil
}

// HandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload).
// Because alert will block js, usually you have to run the wait function in another goroutine.
func (p *Page) HandleDialog(accept bool, promptText string) func() error {
	restore := p.EnableDomain(&proto.PageEnable{})

	wait := p.WaitEvent(&proto.PageJavascriptDialogOpening{})

	return func() error {
		defer restore()

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

		oldView := proto.EmulationSetDeviceMetricsOverride{}
		set := p.LoadState(&oldView)
		view := oldView
		view.Width = int(metrics.ContentSize.Width)
		view.Height = int(metrics.ContentSize.Height)

		err = p.SetViewport(&view)
		if err != nil {
			return nil, err
		}

		defer func() { // try to recover the viewport
			if !set {
				_ = proto.EmulationClearDeviceMetricsOverride{}.Call(p)
				return
			}

			_ = p.SetViewport(&oldView)
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
	var targetID proto.TargetTargetID

	b := p.browser.Context(p.ctx)
	wait := b.EachEvent(func(e *proto.TargetTargetCreated) bool {
		targetID = e.TargetInfo.TargetID
		return e.TargetInfo.OpenerID == p.TargetID
	})

	return func() (*Page, error) {
		wait()
		return b.PageFromTarget(targetID)
	}
}

// WaitPauseOpen waits for a page opened by the current page, before opening pause the js execution.
// Because the js will be paused, you should put the code that triggers it in a goroutine.
func (p *Page) WaitPauseOpen() (func() (*Page, func() error, error), error) {
	// TODO: we have to use the browser to call, seems like a chrome bug
	err := proto.TargetSetAutoAttach{
		AutoAttach:             true,
		WaitForDebuggerOnStart: true,
		Flatten:                true,
	}.Call(p.browser.Context(p.ctx))
	if err != nil {
		return nil, err
	}

	wait := p.WaitOpen()

	return func() (*Page, func() error, error) {
		newPage, err := wait()
		return newPage, func() error {
			err := proto.TargetSetAutoAttach{
				Flatten: true,
			}.Call(p.browser.Context(p.ctx))
			if err != nil {
				return err
			}

			return proto.RuntimeRunIfWaitingForDebugger{}.Call(newPage)
		}, err
	}, nil
}

// EachEvent is similar to Browser.EachEvent, but only catches events for current page.
func (p *Page) EachEvent(callbacks ...interface{}) (wait func()) {
	return p.browser.Context(p.ctx).eachEvent(p.SessionID, callbacks...)
}

// WaitEvent waits for the next event for one time. It will also load the data into the event object.
func (p *Page) WaitEvent(e proto.Event) (wait func()) {
	return p.browser.Context(p.ctx).waitEvent(p.SessionID, e)
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

	p, cancel := p.WithCancel()
	match := genRegMatcher(includes, excludes)
	waitlist := map[proto.NetworkRequestID]string{}
	idleCounter := utils.NewIdleCounter(d)
	update := p.tryTraceReq(includes, excludes)
	update(nil)

	checkDone := func(id proto.NetworkRequestID) {
		if _, has := waitlist[id]; has {
			delete(waitlist, id)
			update(waitlist)
			idleCounter.Done()
		}
	}

	wait := p.EachEvent(func(sent *proto.NetworkRequestWillBeSent) {
		if match(sent.Request.URL) {
			// Redirect will send multiple NetworkRequestWillBeSent events with the same RequestID,
			// we should filter them out.
			if _, has := waitlist[sent.RequestID]; !has {
				waitlist[sent.RequestID] = sent.Request.URL
				update(waitlist)
				idleCounter.Add()
			}
		}
	}, func(e *proto.NetworkLoadingFinished) {
		checkDone(e.RequestID)
	}, func(e *proto.NetworkLoadingFailed) {
		checkDone(e.RequestID)
	})

	return func() {
		go func() {
			idleCounter.Wait(p.ctx)
			cancel()
		}()
		wait()
	}
}

// WaitIdle waits until the next window.requestIdleCallback is called.
func (p *Page) WaitIdle(timeout time.Duration) (err error) {
	_, err = p.Evaluate(jsHelper(js.WaitIdle, timeout.Seconds()).ByPromise())
	return err
}

// WaitLoad waits for the `window.onload` event, it returns immediately if the event is already fired.
func (p *Page) WaitLoad() error {
	_, err := p.Evaluate(jsHelper(js.WaitLoad).ByPromise())
	return err
}

// AddScriptTag to page. If url is empty, content will be used.
func (p *Page) AddScriptTag(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.Evaluate(jsHelper(js.AddScriptTag, id, url, content).ByPromise())
	return err
}

// AddStyleTag to page. If url is empty, content will be used.
func (p *Page) AddStyleTag(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.Evaluate(jsHelper(js.AddStyleTag, id, url, content).ByPromise())
	return err
}

// EvalOnNewDocument Evaluates given script in every frame upon creation (before loading frame's scripts).
func (p *Page) EvalOnNewDocument(js string) (remove func() error, err error) {
	res, err := proto.PageAddScriptToEvaluateOnNewDocument{Source: js}.Call(p)
	if err != nil {
		return
	}

	remove = func() error {
		return proto.PageRemoveScriptToEvaluateOnNewDocument{
			Identifier: res.Identifier,
		}.Call(p)
	}

	return
}

// Expose function to the page's window object. Must bind before navigation. Bindings survive reloads.
func (p *Page) Expose(name string) (callback chan []gson.JSON, stop func() error, err error) {
	fn := "__" + name

	err = proto.RuntimeAddBinding{Name: fn, ExecutionContextID: p.getJSCtxID()}.Call(p)
	if err != nil {
		return
	}

	remove, err := p.EvalOnNewDocument(fmt.Sprintf(
		`function %s(...args) { %s(JSON.stringify(args)) }`, name, fn,
	))
	if err != nil {
		return
	}

	callback = make(chan []gson.JSON)
	p, cancel := p.WithCancel()

	stop = func() error {
		defer cancel()
		err := remove()
		if err != nil {
			return err
		}
		return proto.RuntimeRemoveBinding{Name: fn}.Call(p)
	}

	go func() {
		p.EachEvent(func(e *proto.RuntimeBindingCalled) {
			if e.Name == fn {
				select {
				case <-p.ctx.Done():
				case callback <- gson.NewFrom(e.Payload).Arr():
				}
			}
		})()
	}()

	return
}

// Wait js function until it returns true
func (p *Page) Wait(this *proto.RuntimeRemoteObject, js string, params []interface{}) error {
	removeTrace := func() {}
	defer removeTrace()

	return utils.Retry(p.ctx, p.sleeper(), func() (bool, error) {
		opts := Eval(js, params...).This(this)

		remove := p.tryTraceEval(opts)
		removeTrace()
		removeTrace = remove

		res, err := p.Evaluate(opts)
		if err != nil {
			return true, err
		}

		return res.Value.Bool(), nil
	})
}

// ObjectToJSON by object id
func (p *Page) ObjectToJSON(obj *proto.RuntimeRemoteObject) (gson.JSON, error) {
	if obj.ObjectID == "" {
		return obj.Value, nil
	}

	res, err := proto.RuntimeCallFunctionOn{
		ObjectID:            obj.ObjectID,
		FunctionDeclaration: `function() { return this }`,
		ReturnByValue:       true,
	}.Call(p)
	if err != nil {
		return gson.New(nil), err
	}
	return res.Result.Value, nil
}

// ElementFromObject creates an Element from the remote object id.
func (p *Page) ElementFromObject(obj *proto.RuntimeRemoteObject) *Element {
	// If the element is in an iframe, we need the jsCtxID to inject helper.js to the correct context.
	id := obj.ObjectID.ExecutionID()
	if id != p.getJSCtxID() {
		clone := *p
		clone.jsCtxID = &id
		p = &clone
	}

	return &Element{
		ctx:     p.ctx,
		sleeper: p.sleeper,
		page:    p,
		Object:  obj,
	}
}

// ElementFromNode creates an Element from the node id
func (p *Page) ElementFromNode(id proto.DOMNodeID) (*Element, error) {
	node, err := proto.DOMResolveNode{NodeID: id}.Call(p)
	if err != nil {
		return nil, err
	}

	el := p.ElementFromObject(node.Object)

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
func (p *Page) ElementFromPoint(x, y int) (*Element, error) {
	p.enableNodeQuery()

	node, err := proto.DOMGetNodeForLocation{X: x, Y: y}.Call(p)
	if err != nil {
		return nil, err
	}

	return p.ElementFromNode(node.NodeID)
}

// Release the remote object. Usually, you don't need to call it.
// When a page is closed or reloaded, all remote objects will be released automatically.
// It's useful if the page never closes or reloads.
func (p *Page) Release(obj *proto.RuntimeRemoteObject) error {
	err := proto.RuntimeReleaseObject{ObjectID: obj.ObjectID}.Call(p)
	return err
}

// Call implements the proto.Client
func (p *Page) Call(ctx context.Context, sessionID, methodName string, params interface{}) (res []byte, err error) {
	return p.browser.Call(ctx, sessionID, methodName, params)
}

// Event of the page
func (p *Page) Event() <-chan *Message {
	dst := make(chan *Message)
	p, cancel := p.WithCancel()
	messages := p.browser.Context(p.ctx).Event()

	go func() {
		defer close(dst)
		defer cancel()
		for m := range messages {
			detached := proto.TargetDetachedFromTarget{}
			if m.Load(&detached) && detached.SessionID == p.SessionID {
				return
			}

			if m.SessionID == p.SessionID {
				select {
				case <-p.ctx.Done():
					return
				case dst <- m:
				}
			}
		}
	}()

	return dst
}

func (p *Page) enableNodeQuery() {
	// TODO: I don't know why we need this, seems like a bug of chrome.
	// We should remove it once chrome fixed this bug.
	_, _ = proto.DOMGetDocument{}.Call(p)
}
