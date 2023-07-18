package rod

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/goob"
	"github.com/ysmood/got/lib/lcs"
	"github.com/ysmood/gson"
)

// Page implements these interfaces
var (
	_ proto.Client      = &Page{}
	_ proto.Contextable = &Page{}
	_ proto.Sessionable = &Page{}
)

// Page represents the webpage.
// We try to hold as less states as possible.
// When a page is closed by Rod or not all the ongoing operations an events on it will abort.
type Page struct {
	// TargetID is a unique ID for a remote page.
	// It's usually used in events sent from the browser to tell which page an event belongs to.
	TargetID proto.TargetTargetID

	// FrameID is a unique ID for a browsing context.
	// Usually, different FrameID means different javascript execution context.
	// Such as an iframe and the page it belongs to will have the same TargetID but different FrameIDs.
	FrameID proto.PageFrameID

	// SessionID is a unique ID for a page attachment to a controller.
	// It's usually used in transport layer to tell which page to send the control signal.
	// A page can attached to multiple controllers, the browser uses it distinguish controllers.
	SessionID proto.TargetSessionID

	e eFunc

	ctx context.Context

	// Used to abort all ongoing actions when a page closes.
	sessionCancel func()

	root *Page

	sleeper func() utils.Sleeper

	browser *Browser
	event   *goob.Observable

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard
	Touch    *Touch

	element *Element // iframe only

	jsCtxLock   *sync.Mutex
	jsCtxID     *proto.RuntimeRemoteObjectID // use pointer so that page clones can share the change
	helpersLock *sync.Mutex
	helpers     map[proto.RuntimeRemoteObjectID]map[string]proto.RuntimeRemoteObjectID
}

// String interface
func (p *Page) String() string {
	id := p.TargetID
	if len(id) > 8 {
		id = id[:8]
	}
	return fmt.Sprintf("<page:%s>", id)
}

// IsIframe tells if it's iframe
func (p *Page) IsIframe() bool {
	return p.element != nil
}

// GetSessionID interface
func (p *Page) GetSessionID() proto.TargetSessionID {
	return p.SessionID
}

// Browser of the page
func (p *Page) Browser() *Browser {
	return p.browser
}

// Info of the page, such as the URL or title of the page
func (p *Page) Info() (*proto.TargetTargetInfo, error) {
	return p.browser.pageInfo(p.TargetID)
}

// HTML of the page
func (p *Page) HTML() (string, error) {
	el, err := p.Element("html")
	if err != nil {
		return "", err
	}
	return el.HTML()
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

// SetCookies is similar to Browser.SetCookies .
func (p *Page) SetCookies(cookies []*proto.NetworkCookieParam) error {
	if cookies == nil {
		return proto.NetworkClearBrowserCookies{}.Call(p)
	}
	return proto.NetworkSetCookies{Cookies: cookies}.Call(p)
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
		req = devices.LaptopWithMDPIScreen.UserAgentEmulation()
	}
	return req.Call(p)
}

// SetBlockedURLs For some requests that do not want to be triggered, such as some dangerous operations, delete, quit logout, etc.
// Wildcards ('*') are allowed, such as ["*/api/logout/*","delete"].
// NOTE: if you set empty pattern "", it will block all requests.
func (p *Page) SetBlockedURLs(urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	return proto.NetworkSetBlockedURLs{Urls: urls}.Call(p)
}

// Navigate to the url. If the url is empty, "about:blank" will be used.
// It will return immediately after the server responds the http header.
func (p *Page) Navigate(url string) error {
	if url == "" {
		url = "about:blank"
	}

	// try to stop loading
	_ = p.StopLoading()

	res, err := proto.PageNavigate{URL: url}.Call(p)
	if err != nil {
		return err
	}
	if res.ErrorText != "" {
		return &ErrNavigation{res.ErrorText}
	}

	p.root.unsetJSCtxID()

	return nil
}

// NavigateBack history.
func (p *Page) NavigateBack() error {
	// Not using cdp API because it doesn't work for iframe
	_, err := p.Evaluate(Eval(`() => history.back()`).ByUser())
	return err
}

// NavigateForward history.
func (p *Page) NavigateForward() error {
	// Not using cdp API because it doesn't work for iframe
	_, err := p.Evaluate(Eval(`() => history.forward()`).ByUser())
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
	_, err := p.Evaluate(Eval(`() => location.reload()`).ByUser())
	if err != nil {
		return err
	}

	wait()

	p.unsetJSCtxID()

	return nil
}

// Activate (focuses) the page
func (p *Page) Activate() (*Page, error) {
	err := proto.TargetActivateTarget{TargetID: p.TargetID}.Call(p.browser)
	return p, err
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

// SetDocumentContent sets the page document html content
func (p *Page) SetDocumentContent(html string) error {
	return proto.PageSetDocumentContent{
		FrameID: p.FrameID,
		HTML:    html,
	}.Call(p)
}

// Emulate the device, such as iPhone9. If device is devices.Clear, it will clear the override.
func (p *Page) Emulate(device devices.Device) error {
	err := p.SetViewport(device.MetricsEmulation())
	if err != nil {
		return err
	}

	err = device.TouchEmulation().Call(p)
	if err != nil {
		return err
	}

	return p.SetUserAgent(device.UserAgentEmulation())
}

// StopLoading forces the page stop navigation and pending resource fetches.
func (p *Page) StopLoading() error {
	return proto.PageStopLoading{}.Call(p)
}

// Close tries to close page, running its beforeunload hooks, if has any.
func (p *Page) Close() error {
	p.browser.targetsLock.Lock()
	defer p.browser.targetsLock.Unlock()

	success := true
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()
	messages := p.browser.Context(ctx).Event()

	for {
		err := proto.PageClose{}.Call(p)
		if errors.Is(err, cdp.ErrNotAttachedToActivePage) {
			// TODO: I don't know why chromium doesn't allow us to close a page while it's navigating.
			// Looks like a bug in chromium.
			utils.Sleep(0.1)
			continue
		} else if err != nil {
			return err
		}
		break
	}

	for msg := range messages {
		stop := false

		destroyed := proto.TargetTargetDestroyed{}
		closed := proto.PageJavascriptDialogClosed{}
		if msg.Load(&destroyed) {
			stop = destroyed.TargetID == p.TargetID
		} else if msg.SessionID == p.SessionID && msg.Load(&closed) {
			success = closed.Result
			stop = !success
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

// TriggerFavicon supports when browser in headless mode
// to trigger favicon's request. Pay attention to this
// function only supported when browser in headless mode,
// if you call it in no-headless mode, it will raise an error
// with the message "browser is no-headless".
func (p *Page) TriggerFavicon() error {
	// check if browser whether in headless mode
	// if not in headless mode then raise error
	if !p.browser.isHeadless() {
		return errors.New("browser is no-headless")
	}

	_, err := p.Evaluate(evalHelper(js.TriggerFavicon).ByPromise())
	if err != nil {
		return err
	}
	return nil
}

// HandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload).
// Because modal dialog will block js, usually you have to trigger the dialog in another goroutine.
// For example:
//
//	wait, handle := page.MustHandleDialog()
//	go page.MustElement("button").MustClick()
//	wait()
//	handle(true, "")
func (p *Page) HandleDialog() (
	wait func() *proto.PageJavascriptDialogOpening,
	handle func(*proto.PageHandleJavaScriptDialog) error,
) {
	restore := p.EnableDomain(&proto.PageEnable{})

	var e proto.PageJavascriptDialogOpening
	w := p.WaitEvent(&e)

	return func() *proto.PageJavascriptDialogOpening {
			w()
			return &e
		}, func(h *proto.PageHandleJavaScriptDialog) error {
			defer restore()
			return h.Call(p)
		}
}

// HandleFileDialog return a functions that waits for the next file chooser dialog pops up and returns the element
// for the event.
func (p *Page) HandleFileDialog() (func([]string) error, error) {
	err := proto.PageSetInterceptFileChooserDialog{Enabled: true}.Call(p)
	if err != nil {
		return nil, err
	}

	var e proto.PageFileChooserOpened
	w := p.WaitEvent(&e)

	return func(paths []string) error {
		w()

		err := proto.PageSetInterceptFileChooserDialog{Enabled: false}.Call(p)
		if err != nil {
			return err
		}

		return proto.DOMSetFileInputFiles{
			Files:         utils.AbsolutePaths(paths),
			BackendNodeID: e.BackendNodeID,
		}.Call(p)
	}, nil
}

// Screenshot captures the screenshot of current page.
func (p *Page) Screenshot(fullPage bool, req *proto.PageCaptureScreenshot) ([]byte, error) {
	if req == nil {
		req = &proto.PageCaptureScreenshot{}
	}
	if fullPage {
		metrics, err := proto.PageGetLayoutMetrics{}.Call(p)
		if err != nil {
			return nil, err
		}

		if metrics.CSSContentSize == nil {
			return nil, errors.New("failed to get css content size")
		}

		oldView := proto.EmulationSetDeviceMetricsOverride{}
		set := p.LoadState(&oldView)
		view := oldView
		view.Width = int(metrics.CSSContentSize.Width)
		view.Height = int(metrics.CSSContentSize.Height)

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

// CaptureDOMSnapshot Returns a document snapshot, including the full DOM tree of the root node
// (including iframes, template contents, and imported documents) in a flattened array,
// as well as layout and white-listed computed style information for the nodes.
// Shadow DOM in the returned DOM tree is flattened.
// `Documents` The nodes in the DOM tree. The DOMNode at index 0 corresponds to the root document.
// `Strings` Shared string table that all string properties refer to with indexes.
// Normally use `Strings` is enough.
func (p *Page) CaptureDOMSnapshot() (domSnapshot *proto.DOMSnapshotCaptureSnapshotResult, err error) {
	_ = proto.DOMSnapshotEnable{}.Call(p)

	snapshot, err := proto.DOMSnapshotCaptureSnapshot{
		ComputedStyles:                 []string{},
		IncludePaintOrder:              true,
		IncludeDOMRects:                true,
		IncludeBlendedBackgroundColors: true,
		IncludeTextColorOpacities:      true,
	}.Call(p)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
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

// GetResource content by the url. Such as image, css, html, etc.
// Use the proto.PageGetResourceTree to list all the resources.
func (p *Page) GetResource(url string) ([]byte, error) {
	res, err := proto.PageGetResourceContent{
		FrameID: p.FrameID,
		URL:     url,
	}.Call(p)
	if err != nil {
		return nil, err
	}

	data := res.Content

	var bin []byte
	if res.Base64Encoded {
		bin, err = base64.StdEncoding.DecodeString(data)
		utils.E(err)
	} else {
		bin = []byte(data)
	}

	return bin, nil
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
		defer p.tryTrace(TraceTypeWait, "wait open")()
		wait()
		return b.PageFromTarget(targetID)
	}
}

// EachEvent of the specified event types, if any callback returns true the wait function will resolve,
// The type of each callback is (? means optional):
//
//	func(proto.Event, proto.TargetSessionID?) bool?
//
// You can listen to multiple event types at the same time like:
//
//	browser.EachEvent(func(a *proto.A) {}, func(b *proto.B) {})
//
// Such as subscribe the events to know when the navigation is complete or when the page is rendered.
// Here's an example to dismiss all dialogs/alerts on the page:
//
//	go page.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
//	    _ = proto.PageHandleJavaScriptDialog{ Accept: false, PromptText: ""}.Call(page)
//	})()
func (p *Page) EachEvent(callbacks ...interface{}) (wait func()) {
	return p.browser.Context(p.ctx).eachEvent(p.SessionID, callbacks...)
}

// WaitEvent waits for the next event for one time. It will also load the data into the event object.
func (p *Page) WaitEvent(e proto.Event) (wait func()) {
	defer p.tryTrace(TraceTypeWait, "event", e.ProtoEvent())()
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
		defer p.tryTrace(TraceTypeWait, "navigation", name)()
		wait()
		_ = proto.PageSetLifecycleEventsEnabled{Enabled: false}.Call(p)
	}
}

// WaitRequestIdle returns a wait function that waits until no request for d duration.
// Be careful, d is not the max wait timeout, it's the least idle time.
// If you want to set a timeout you can use the "Page.Timeout" function.
// Use the includes and excludes regexp list to filter the requests by their url.
func (p *Page) WaitRequestIdle(d time.Duration, includes, excludes []string, excludeTypes []proto.NetworkResourceType) func() {
	defer p.tryTrace(TraceTypeWait, "request-idle")()

	if excludeTypes == nil {
		excludeTypes = []proto.NetworkResourceType{
			proto.NetworkResourceTypeWebSocket,
			proto.NetworkResourceTypeEventSource,
			proto.NetworkResourceTypeMedia,
			proto.NetworkResourceTypeImage,
			proto.NetworkResourceTypeFont,
		}
	}

	if len(includes) == 0 {
		includes = []string{""}
	}

	p, cancel := p.WithCancel()
	match := genRegMatcher(includes, excludes)
	waitList := map[proto.NetworkRequestID]string{}
	idleCounter := utils.NewIdleCounter(d)
	update := p.tryTraceReq(includes, excludes)
	update(nil)

	checkDone := func(id proto.NetworkRequestID) {
		if _, has := waitList[id]; has {
			delete(waitList, id)
			update(waitList)
			idleCounter.Done()
		}
	}

	wait := p.EachEvent(func(sent *proto.NetworkRequestWillBeSent) {
		for _, t := range excludeTypes {
			if sent.Type == t {
				return
			}
		}

		if match(sent.Request.URL) {
			// Redirect will send multiple NetworkRequestWillBeSent events with the same RequestID,
			// we should filter them out.
			if _, has := waitList[sent.RequestID]; !has {
				waitList[sent.RequestID] = sent.Request.URL
				update(waitList)
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

// WaitDOMStable waits until the change of the DOM tree is less or equal than diff percent for d duration.
// Be careful, d is not the max wait timeout, it's the least stable time.
// If you want to set a timeout you can use the "Page.Timeout" function.
func (p *Page) WaitDOMStable(d time.Duration, diff float64) error {
	defer p.tryTrace(TraceTypeWait, "dom-stable")()

	domSnapshot, err := p.CaptureDOMSnapshot()
	if err != nil {
		return err
	}

	t := time.NewTicker(d)
	defer t.Stop()

	for {
		select {
		case <-t.C:
		case <-p.ctx.Done():
			return p.ctx.Err()
		}

		currentDomSnapshot, err := p.CaptureDOMSnapshot()
		if err != nil {
			return err
		}

		xs := lcs.NewWords(domSnapshot.Strings)
		ys := lcs.NewWords(currentDomSnapshot.Strings)
		lcs := xs.YadLCS(p.ctx, ys)

		df := 1 - float64(len(lcs))/float64(len(ys))
		if df <= diff {
			break
		}

		domSnapshot = currentDomSnapshot
	}
	return nil
}

// WaitStable waits until the page is stable for d duration.
func (p *Page) WaitStable(d time.Duration) error {
	defer p.tryTrace(TraceTypeWait, "stable")()

	var err error
	lock := sync.Mutex{}

	utils.All(func() {
		e := p.WaitLoad()
		lock.Lock()
		err = e
		lock.Unlock()
	}, func() {
		p.WaitRequestIdle(d, nil, nil, nil)()
	}, func() {
		e := p.WaitDOMStable(d, 0)
		lock.Lock()
		err = e
		lock.Unlock()
	})()

	return err
}

// WaitIdle waits until the next window.requestIdleCallback is called.
func (p *Page) WaitIdle(timeout time.Duration) (err error) {
	_, err = p.Evaluate(evalHelper(js.WaitIdle, timeout.Milliseconds()).ByPromise())
	return err
}

// WaitRepaint waits until the next repaint.
// Doc: https://developer.mozilla.org/en-US/docs/Web/API/window/requestAnimationFrame
func (p *Page) WaitRepaint() error {
	// we use root here because iframe doesn't trigger requestAnimationFrame
	_, err := p.root.Eval(`() => new Promise(r => requestAnimationFrame(r))`)
	return err
}

// WaitLoad waits for the `window.onload` event, it returns immediately if the event is already fired.
func (p *Page) WaitLoad() error {
	defer p.tryTrace(TraceTypeWait, "load")()
	_, err := p.Evaluate(evalHelper(js.WaitLoad).ByPromise())
	return err
}

// AddScriptTag to page. If url is empty, content will be used.
func (p *Page) AddScriptTag(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.Evaluate(evalHelper(js.AddScriptTag, id, url, content).ByPromise())
	return err
}

// AddStyleTag to page. If url is empty, content will be used.
func (p *Page) AddStyleTag(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.Evaluate(evalHelper(js.AddStyleTag, id, url, content).ByPromise())
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

// Wait until the js returns true
func (p *Page) Wait(opts *EvalOptions) error {
	return utils.Retry(p.ctx, p.sleeper(), func() (bool, error) {
		res, err := p.Evaluate(opts)
		if err != nil {
			return true, err
		}

		return res.Value.Bool(), nil
	})
}

// WaitElementsMoreThan Wait until there are more than <num> <selector> elements.
func (p *Page) WaitElementsMoreThan(selector string, num int) error {
	return p.Wait(Eval(`(s, n) => document.querySelectorAll(s).length > n`, selector, num))
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
func (p *Page) ElementFromObject(obj *proto.RuntimeRemoteObject) (*Element, error) {
	// If the element is in an iframe, we need the jsCtxID to inject helper.js to the correct context.
	id, err := p.jsCtxIDByObjectID(obj.ObjectID)
	if err != nil {
		return nil, err
	}

	pid, err := p.getJSCtxID()
	if err != nil {
		return nil, err
	}

	if id != pid {
		clone := *p
		clone.jsCtxID = &id
		p = &clone
	}

	return &Element{
		e:       p.e,
		ctx:     p.ctx,
		sleeper: p.sleeper,
		page:    p,
		Object:  obj,
	}, nil
}

// ElementFromNode creates an Element from the node, NodeID or BackendNodeID must be specified.
func (p *Page) ElementFromNode(node *proto.DOMNode) (*Element, error) {
	res, err := proto.DOMResolveNode{
		NodeID:        node.NodeID,
		BackendNodeID: node.BackendNodeID,
	}.Call(p)
	if err != nil {
		return nil, err
	}

	el, err := p.ElementFromObject(res.Object)
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
func (p *Page) ElementFromPoint(x, y int) (*Element, error) {
	node, err := proto.DOMGetNodeForLocation{X: x, Y: y}.Call(p)
	if err != nil {
		return nil, err
	}

	return p.ElementFromNode(&proto.DOMNode{
		BackendNodeID: node.BackendNodeID,
	})
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
	s := p.event.Subscribe(p.ctx)

	go func() {
		defer close(dst)
		for {
			select {
			case <-p.ctx.Done():
				return
			case msg, ok := <-s:
				if !ok {
					return
				}
				select {
				case <-p.ctx.Done():
					return
				case dst <- msg.(*Message):
				}
			}
		}
	}()

	return dst
}

func (p *Page) initEvents() {
	p.event = goob.New(p.ctx)
	event := p.browser.Context(p.ctx).Event()

	go func() {
		for msg := range event {
			detached := proto.TargetDetachedFromTarget{}
			destroyed := proto.TargetTargetDestroyed{}

			if (msg.Load(&detached) && detached.SessionID == p.SessionID) ||
				(msg.Load(destroyed) && destroyed.TargetID == p.TargetID) {
				p.sessionCancel()
				return
			}

			if msg.SessionID != p.SessionID {
				continue
			}

			p.event.Publish(msg)
		}
	}()
}
