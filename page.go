//go:generate go run ./lib/assets/generate

package rod

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/assets"
	"github.com/ysmood/rod/lib/cdp"
)

// Page represents the webpage
type Page struct {
	// these are the handler for ctx
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	browser *Browser

	TargetID  string
	SessionID string
	FrameID   string
	URL       string

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard

	element             *Element // iframe only
	windowObjectID      string   // used as the thisObject when eval js
	getDownloadFileLock *sync.Mutex
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
func (p *Page) CookiesE(urls []string) ([]gjson.Result, error) {
	if len(urls) == 0 {
		info, err := p.CallE("Target.getTargetInfo", cdp.Object{
			"targetId": p.TargetID,
		})
		if err != nil {
			return nil, err
		}
		urls = []string{info.Get("targetInfo.url").String()}
	}

	res, err := p.CallE("Network.getCookies", cdp.Object{
		"urls": urls,
	})
	if err != nil {
		return nil, err
	}
	return res.Get("cookies").Array(), nil
}

// SetCookiesE of the page.
// Cookie format: https://chromedevtools.github.io/devtools-protocol/tot/Network#method-setCookie
func (p *Page) SetCookiesE(cookies []cdp.Object) error {
	_, err := p.CallE("Network.setCookies", cdp.Object{
		"cookies": cookies,
	})
	return err
}

// NavigateE doc is the same as the method Navigate
func (p *Page) NavigateE(url string) error {
	_, err := p.CallE("Page.navigate", cdp.Object{
		"url": url,
	})
	return err
}

func (p *Page) getWindowID() (int64, error) {
	res, err := p.CallE("Browser.getWindowForTarget", cdp.Object{"targetId": p.TargetID})
	if err != nil {
		return 0, err
	}
	return res.Get("windowId").Int(), err
}

// GetWindowE doc is the same as the method GetWindow
func (p *Page) GetWindowE() (kit.JSONResult, error) {
	id, err := p.getWindowID()
	if err != nil {
		return nil, err
	}

	res, err := p.CallE("Browser.getWindowBounds", cdp.Object{"windowId": id})
	if err != nil {
		return nil, err
	}

	bounds := res.Get("bounds")
	return &bounds, nil
}

// WindowE https://chromedevtools.github.io/devtools-protocol/tot/Browser#type-Bounds
func (p *Page) WindowE(bounds cdp.Object) error {
	id, err := p.getWindowID()
	if err != nil {
		return err
	}

	_, err = p.CallE("Browser.setWindowBounds", cdp.Object{"windowId": id, "bounds": bounds})
	return err
}

// ViewportE doc is the same as the method Viewport
// Prams: https://chromedevtools.github.io/devtools-protocol/tot/Emulation#method-setDeviceMetricsOverride
func (p *Page) ViewportE(params cdp.Object) error {
	if params == nil {
		return nil
	}
	_, err := p.CallE("Emulation.setDeviceMetricsOverride", params)
	return err
}

// CloseE page
func (p *Page) CloseE() error {
	_, err := p.CallE("Page.close", nil)
	return err
}

// HandleDialogE doc is the same as the method HandleDialog
func (p *Page) HandleDialogE(accept bool, promptText string) func() error {
	wait := p.WaitEventE(Method("Page.javascriptDialogOpening"))

	return func() error {
		_, err := wait()
		if err != nil {
			return err
		}
		_, err = p.CallE("Page.handleJavaScriptDialog", cdp.Object{
			"accept":     accept,
			"promptText": promptText,
		})
		return err
	}
}

// GetDownloadFileE how it works is to proxy the request, the dir is the dir to save the file.
func (p *Page) GetDownloadFileE(dir, pattern string) (func() (http.Header, []byte, error), error) {
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

	_, err := p.CallE("Page.setDownloadBehavior", cdp.Object{
		"behavior":     "allow",
		"downloadPath": dir,
	})
	if err != nil {
		return nil, err
	}

	_, err = p.CallE("Fetch.enable", params)
	if err != nil {
		return nil, err
	}

	wait := p.WaitEventE(Method("Fetch.requestPaused"))

	return func() (http.Header, []byte, error) {
		defer func() {
			defer p.getDownloadFileLock.Unlock()
			_, err := p.CallE("Fetch.disable", nil)
			kit.E(err)
		}()

		msg, err := wait()
		if err != nil {
			return nil, nil, err
		}

		msgReq := msg.Params.Get("request")
		req := kit.Req(msgReq.Get("url").String()).Context(p.ctx)

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

		_, err = p.CallE("Fetch.fulfillRequest", cdp.Object{
			"requestId":       msg.Params.Get("requestId").String(),
			"responseCode":    res.StatusCode,
			"responseHeaders": headers,
			"body":            base64.StdEncoding.EncodeToString(body),
		})

		return res.Header, body, err
	}, err
}

// ScreenshotE options: https://chromedevtools.github.io/devtools-protocol/tot/Page#method-captureScreenshot
func (p *Page) ScreenshotE(options cdp.Object) ([]byte, error) {
	res, err := p.CallE("Page.captureScreenshot", options)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res.Get("data").String())
}

// PDFE prints page as PDF
func (p *Page) PDFE(options cdp.Object) ([]byte, error) {
	res, err := p.CallE("Page.printToPDF", options)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res.Get("data").String())
}

// WaitPageE doc is the same as the method WaitPage
func (p *Page) WaitPageE() func() (*Page, error) {
	var targetInfo gjson.Result

	wait := p.browser.Context(p.ctx).WaitEventE(func(e *cdp.Event) bool {
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
	}
}

// PauseE doc is the same as the method Pause
func (p *Page) PauseE() error {
	_, err := p.CallE("Debugger.enable", nil)
	if err != nil {
		return err
	}
	_, err = p.CallE("Debugger.pause", nil)
	if err != nil {
		return err
	}
	wait := p.WaitEventE(Method("Debugger.resumed"))
	_, err = wait()
	return err
}

// WaitRequestIdleE returns a wait function that waits until no request for d duration.
// Use the includes and excludes regexp list to filter the requests by their url.
// Such as set n to 1 if there's a polling request.
func (p *Page) WaitRequestIdleE(d time.Duration, includes, excludes []string) func() error {
	s := p.browser.Event().Subscribe()
	done := false

	return func() (err error) {
		defer func() { done = true }()
		if done {
			panic("can't use wait function twice")
		}

		if p.browser.trace {
			defer p.Overlay(0, 0, 300, 0, "waiting for request idle "+strings.Join(includes, " "))()
		}
		defer p.browser.Event().Unsubscribe(s)

		reqList := map[string]kit.Nil{}
		timeout := time.NewTimer(d)

		for {
			select {
			case <-p.ctx.Done():
				return p.ctx.Err()
			case <-timeout.C:
				return
			case msg, ok := <-s.C:
				if !ok {
					return
				}

				e := msg.(*cdp.Event)
				switch e.Method {
				case "Network.requestWillBeSent":
					timeout.Stop()
					url := e.Params.Get("request.url").String()
					id := e.Params.Get("requestId").String()
					if matchWithFilter(url, includes, excludes) {
						reqList[id] = kit.Nil{}
					}
				case "Network.loadingFinished",
					"Network.loadingFailed",
					"Network.responseReceived":
					delete(reqList, e.Params.Get("requestId").String())
					if len(reqList) == 0 {
						timeout.Reset(d)
					}
				}
			}
		}
	}
}

// WaitIdleE doc is the same as the method WaitIdle
func (p *Page) WaitIdleE(timeout time.Duration) (err error) {
	_, err = p.EvalE(true, "", p.jsFn("waitIdle"), cdp.Array{timeout.Seconds()})
	return err
}

// WaitLoadE doc is the same as the method WaitLoad
func (p *Page) WaitLoadE() error {
	_, err := p.EvalE(true, "", p.jsFn("waitLoad"), nil)
	return err
}

// WaitEventE doc is the same as the method WaitEvent
func (p *Page) WaitEventE(filter EventFilter) func() (*cdp.Event, error) {
	return p.browser.Context(p.ctx).WaitEventE(func(e *cdp.Event) bool {
		return e.SessionID == p.SessionID && filter(e)
	})
}

// AddScriptTagE to page. If url is empty, content will be used.
func (p *Page) AddScriptTagE(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.EvalE(true, "", p.jsFn("addScriptTag"), cdp.Array{id, url, content})
	return err
}

// AddStyleTagE to page. If url is empty, content will be used.
func (p *Page) AddStyleTagE(url, content string) error {
	hash := md5.Sum([]byte(url + content))
	id := hex.EncodeToString(hash[:])
	_, err := p.EvalE(true, "", p.jsFn("addStyleTag"), cdp.Array{id, url, content})
	return err
}

// EvalE thisID is the remote objectID that will be the this of the js function, if it's empty "window" will be used.
// Set the byValue to true to reduce memory occupation.
func (p *Page) EvalE(byValue bool, thisID, js string, jsArgs cdp.Array) (res kit.JSONResult, err error) {
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

		args := cdp.Array{}
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

		res, err = p.CallE("Runtime.callFunctionOn", params)

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
func (p *Page) CallE(method string, params interface{}) (kit.JSONResult, error) {
	return p.browser.Context(p.ctx).CallE(&cdp.Request{
		SessionID: p.SessionID,
		Method:    method,
		Params:    params,
	})
}

// Sleeper returns the default sleeper for retry, it uses backoff and requestIdleCallback to wait
func (p *Page) Sleeper() kit.Sleeper {
	return kit.BackoffSleeper(100*time.Millisecond, time.Second, nil)
}

// ReleaseE doc is the same as the method Release
func (p *Page) ReleaseE(objectID string) error {
	_, err := p.CallE("Runtime.releaseObject", cdp.Object{
		"objectId": objectID,
	})
	return err
}

func (p *Page) initSession() error {
	obj, err := p.CallE("Target.attachToTarget", cdp.Object{
		"targetId": p.TargetID,
		"flatten":  true, // if it's not set no response will return
	})
	if err != nil {
		return err
	}
	p.SessionID = obj.Get("sessionId").String()

	_, err = p.CallE("Page.enable", nil)
	if err != nil {
		return err
	}

	_, err = p.CallE("Network.enable", nil)
	if err != nil {
		return err
	}

	res, err := p.CallE("DOM.getDocument", nil)
	if err != nil {
		return err
	}

	for _, child := range res.Get("root.children").Array() {
		frameID := child.Get("frameId")
		if frameID.Exists() {
			p.FrameID = frameID.String()
		}
	}

	return p.ViewportE(p.browser.viewport)
}

func (p *Page) initJS() error {
	scriptURL := "\n//# sourceURL=__rod_helper__"

	params := cdp.Object{
		"expression": sprintFnApply(assets.Helper, cdp.Array{p.FrameID}) + scriptURL,
	}

	if p.IsIframe() {
		res, err := p.CallE("Page.createIsolatedWorld", cdp.Object{
			"frameId": p.FrameID,
		})
		if err != nil {
			return err
		}

		params["contextId"] = res.Get("executionContextId").Int()
	}

	res, err := p.CallE("Runtime.evaluate", params)
	if err != nil {
		return err
	}

	p.windowObjectID = res.Get("result.objectId").String()

	return nil
}

func (p *Page) jsFnPrefix() string {
	return "rod" + p.FrameID + "."
}

func (p *Page) jsFn(fnName string) string {
	return p.jsFnPrefix() + fnName
}
