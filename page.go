//go:generate go run ./lib/js/generate

package rod

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/js"
)

// Page represents the webpage
type Page struct {
	ctx     context.Context
	browser *Browser

	TargetID  string
	SessionID string
	FrameID   string

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard

	// iframe only
	element *Element

	windowObjectID string

	timeoutCancel       func()
	getDownloadFileLock *sync.Mutex

	traceDir string
}

// Context sets the context for chained sub-operations
func (p *Page) Context(ctx context.Context) *Page {
	newObj := *p
	newObj.ctx = ctx
	return &newObj
}

// Timeout sets the timeout for chained sub-operations
func (p *Page) Timeout(d time.Duration) *Page {
	ctx, cancel := context.WithTimeout(p.ctx, d)
	p.timeoutCancel = cancel
	return p.Context(ctx)
}

// CancelTimeout ...
func (p *Page) CancelTimeout() *Page {
	if p.timeoutCancel != nil {
		p.timeoutCancel()
	}
	return p
}

// TraceDir set the dir to save the trace screenshots.
// If it's set, screenshots will be taken before each trace.
func (p *Page) TraceDir(dir string) *Page {
	p.traceDir = dir
	return p
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

// NavigateE ...
func (p *Page) NavigateE(url string) error {
	res, err := p.CallE(nil, "Page.navigate", cdp.Object{
		"url": url,
	})
	if err != nil {
		return err
	}

	p.FrameID = res.Get("frameId").String()

	return nil
}

func (p *Page) getWindowID() (int64, error) {
	res, err := p.browser.CallE(p.ctx, &cdp.Request{
		Method: "Browser.getWindowForTarget",
		Params: cdp.Object{
			"targetId": p.TargetID,
		},
	})
	if err != nil {
		return 0, err
	}
	return res.Get("windowId").Int(), err
}

// GetWindowE ...
func (p *Page) GetWindowE() (kit.JSONResult, error) {
	id, err := p.getWindowID()
	if err != nil {
		return nil, err
	}

	res, err := p.browser.CallE(p.ctx, &cdp.Request{
		Method: "Browser.getWindowBounds",
		Params: cdp.Object{
			"windowId": id,
		},
	})
	if err != nil {
		return nil, err
	}

	bounds := res.Get("bounds")
	return &bounds, nil
}

// WindowE https://chromedevtools.github.io/devtools-protocol/tot/Browser#type-Bounds
func (p *Page) WindowE(bounds *cdp.Object) error {
	id, err := p.getWindowID()
	if err != nil {
		return err
	}

	_, err = p.browser.CallE(p.ctx, &cdp.Request{
		Method: "Browser.setWindowBounds",
		Params: cdp.Object{
			"windowId": id,
			"bounds":   bounds,
		},
	})
	return err
}

// ViewportE ...
// Prams: https://chromedevtools.github.io/devtools-protocol/tot/Emulation#method-setDeviceMetricsOverride
func (p *Page) ViewportE(params *cdp.Object) error {
	if params == nil {
		return nil
	}
	_, err := p.CallE(nil, "Emulation.setDeviceMetricsOverride", params)
	return err
}

// CloseE page
func (p *Page) CloseE() error {
	_, err := p.CallE(nil, "Page.close", nil)
	return err
}

// HandleDialogE ...
func (p *Page) HandleDialogE(accept bool, promptText string) (func() error, func()) {
	wait, cancel := p.WaitEventE(Method("Page.javascriptDialogOpening"))

	return func() error {
		_, err := wait()
		if err != nil {
			return err
		}
		_, err = p.CallE(nil, "Page.handleJavaScriptDialog", cdp.Object{
			"accept":     accept,
			"promptText": promptText,
		})
		return err
	}, cancel
}

// GetDownloadFileE how it works is to proxy the request, the dir is the dir to save the file.
func (p *Page) GetDownloadFileE(dir, pattern string) (func() (http.Header, []byte, error), func(), error) {
	var params cdp.Object
	if pattern != "" {
		params = cdp.Object{
			"patterns": []cdp.Object{
				{"urlPattern": pattern},
			},
		}
	}

	// both Page.setDownloadBehavior and Fetch.enable will pollute the global status,
	// we have to prevent race condition here
	p.getDownloadFileLock.Lock()

	_, err := p.CallE(nil, "Page.setDownloadBehavior", cdp.Object{
		"behavior":     "allow",
		"downloadPath": dir,
	})
	if err != nil {
		return nil, nil, err
	}

	_, err = p.CallE(nil, "Fetch.enable", params)
	if err != nil {
		return nil, nil, err
	}

	wait, cancel := p.WaitEventE(Method("Fetch.requestPaused"))

	released := false
	release := func() {
		defer cancel()
		if released {
			return
		}
		released = true
		defer p.getDownloadFileLock.Unlock()
		_, err := p.CallE(nil, "Fetch.disable", nil)
		kit.E(err)
	}

	return func() (http.Header, []byte, error) {
		defer release()

		msg, err := wait()
		if err != nil {
			return nil, nil, err
		}

		msgReq := msg.Params.Get("request")
		req := kit.Req(msgReq.Get("url").String())

		for k, v := range msgReq.Get("headers").Map() {
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

		headers := []cdp.Object{}
		for k, vs := range res.Header {
			for _, v := range vs {
				headers = append(headers, cdp.Object{
					"name":  k,
					"value": v,
				})
			}
		}

		_, err = p.CallE(nil, "Fetch.fulfillRequest", cdp.Object{
			"requestId":       msg.Params.Get("requestId").String(),
			"responseCode":    res.StatusCode,
			"responseHeaders": headers,
			"body":            base64.StdEncoding.EncodeToString(body),
		})

		return res.Header, body, err
	}, release, err
}

// ScreenshotE options: https://chromedevtools.github.io/devtools-protocol/tot/Page#method-captureScreenshot
func (p *Page) ScreenshotE(options cdp.Object) ([]byte, error) {
	res, err := p.CallE(nil, "Page.captureScreenshot", options)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res.Get("data").String())
}

// WaitPageE ...
func (p *Page) WaitPageE() (func() (*Page, error), func()) {
	var targetInfo gjson.Result

	wait, cancel := p.browser.Context(p.ctx).WaitEventE(func(e *cdp.Event) bool {
		if e.Method == "Target.targetCreated" {
			targetInfo = e.Params.Get("targetInfo")

			if targetInfo.Get("openerId").String() == p.TargetID {
				return true
			}
		}
		return false
	})

	return func() (*Page, error) {
		_, err := wait()
		if err != nil {
			return nil, err
		}
		return p.browser.Context(p.ctx).page(targetInfo.Get("targetId").String())
	}, cancel
}

// PauseE ...
func (p *Page) PauseE() error {
	_, err := p.CallE(nil, "Debugger.enable", nil)
	if err != nil {
		return err
	}
	_, err = p.CallE(nil, "Debugger.pause", nil)
	if err != nil {
		return err
	}
	wait, _ := p.WaitEventE(Method("Debugger.resumed"))
	_, err = wait()
	return err
}

// WaitIdleE ...
func (p *Page) WaitIdleE() error {
	d, _ := p.ctx.Deadline()
	timeout := d.Sub(time.Now()).Milliseconds()
	_, err := p.EvalE(true, "", p.jsFn("waitIdle"), []interface{}{timeout})
	return err
}

// WaitLoadE ...
func (p *Page) WaitLoadE() error {
	_, err := p.EvalE(true, "", p.jsFn("waitLoad"), nil)
	return err
}

// WaitEventE ...
func (p *Page) WaitEventE(filter EventFilter) (func() (*cdp.Event, error), func()) {
	return p.browser.Context(p.ctx).WaitEventE(func(e *cdp.Event) bool {
		return e.SessionID == p.SessionID && filter(e)
	})
}

// EvalE thisID is the remote objectID that will be the this of the js function, if it's empty "window" will be used.
// Set the byValue to true to reduce memory occupation.
func (p *Page) EvalE(byValue bool, thisID, js string, jsArgs []interface{}) (res kit.JSONResult, err error) {
	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	objectID := thisID

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

		args := []interface{}{}
		for _, p := range jsArgs {
			args = append(args, cdp.Object{"value": p})
		}

		params := cdp.Object{
			"objectId":            objectID,
			"awaitPromise":        true,
			"returnByValue":       byValue,
			"functionDeclaration": SprintFnThis(js),
			"arguments":           args,
		}

		res, err = p.CallE(nil, "Runtime.callFunctionOn", params)

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

	if res.Get("exceptionDetails").Exists() {
		return nil, &Error{nil, res.Get("exceptionDetails.exception.description").String(), res}
	}

	if byValue {
		val := res.Get("result.value")
		res = &val
	}

	return
}

// CallE sends a control message to the browser with the page session, the call is always on the root frame.
func (p *Page) CallE(ctx context.Context, method string, params interface{}) (kit.JSONResult, error) {
	if ctx == nil {
		ctx = p.ctx
	}
	return p.browser.CallE(ctx, &cdp.Request{
		SessionID: p.SessionID,
		Method:    method,
		Params:    params,
	})
}

// Sleeper returns the default sleeper for retry, it will wake whenever Page for DOM event happens,
// and use backoff as the backup to wake.
func (p *Page) Sleeper() kit.Sleeper {
	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)

	return kit.MergeSleepers(backoff, func(ctx context.Context) error {
		s := p.browser.event.Subscribe()
		defer p.browser.event.Unsubscribe(s)
		prefix := strings.HasPrefix

		c := s.Filter(func(e kit.Event) bool {
			m := e.(*cdp.Event).Method
			if prefix(m, "Page") || prefix(m, "DOM") {
				return true
			}
			return false
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c:
		}
		return nil
	})
}

// ReleaseE ...
func (p *Page) ReleaseE(objectID string) error {
	_, err := p.CallE(nil, "Runtime.releaseObject", cdp.Object{
		"objectId": objectID,
	})
	return err
}

func (p *Page) initSession() error {
	obj, err := p.CallE(nil, "Target.attachToTarget", cdp.Object{
		"targetId": p.TargetID,
		"flatten":  true, // if it's not set no response will return
	})
	if err != nil {
		return err
	}
	p.SessionID = obj.Get("sessionId").String()
	_, err = p.CallE(nil, "Page.enable", nil)
	if err != nil {
		return err
	}

	return p.ViewportE(p.browser.viewport)
}

func (p *Page) initJS() error {
	scriptURL := "\n//# sourceURL=__rod_helper__"

	params := cdp.Object{
		"expression": sprintFnApply(js.Rod, []interface{}{p.FrameID}) + scriptURL,
	}

	if p.IsIframe() {
		res, err := p.CallE(nil, "Page.createIsolatedWorld", cdp.Object{
			"frameId": p.FrameID,
		})
		if err != nil {
			return err
		}

		params["contextId"] = res.Get("executionContextId").Int()
	}

	res, err := p.CallE(nil, "Runtime.evaluate", params)
	if err != nil {
		return err
	}

	p.windowObjectID = res.Get("result.objectId").String()

	return nil
}

func (p *Page) jsFn(fnName string) string {
	return "rod" + p.FrameID + "." + fnName
}
