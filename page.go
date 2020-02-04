package rod

import (
	"context"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Page represents the webpage
type Page struct {
	ctx     context.Context
	browser *Browser

	TargetID  string
	SessionID string
	ContextID int64

	// devices
	Mouse    *Mouse
	Keyboard *Keyboard

	// iframe only
	FrameID string
	element *Element

	timeoutCancel func()
}

// Ctx sets the context for later operation
func (p *Page) Ctx(ctx context.Context) *Page {
	newObj := *p
	newObj.ctx = ctx
	return &newObj
}

// Timeout sets the timeout for later operation
func (p *Page) Timeout(d time.Duration) *Page {
	ctx, cancel := context.WithTimeout(p.ctx, d)
	p.timeoutCancel = cancel
	return p.Ctx(ctx)
}

// CancelTimeout ...
func (p *Page) CancelTimeout() {
	if p.timeoutCancel != nil {
		p.timeoutCancel()
	}
}

// NavigateE ...
func (p *Page) NavigateE(url string) error {
	_, err := p.Call("Page.navigate", cdp.Object{
		"url": url,
	})
	if err != nil {
		return err
	}
	_, err = p.WaitEventE("Page.frameStoppedLoading")
	return err
}

// Navigate to url and wait until Page.frameStoppedLoading fired
func (p *Page) Navigate(url string) *Page {
	kit.E(p.NavigateE(url))
	return p
}

// CloseE page
func (p *Page) CloseE() error {
	_, err := p.Call("Page.close", nil)
	return err
}

// Close page
func (p *Page) Close() {
	kit.E(p.CloseE())
}

// HasE ...
func (p *Page) HasE(selector string) (bool, error) {
	res, err := p.EvalE(true, "", `s => !!document.querySelector(s)`, []interface{}{selector})
	if err != nil {
		return false, err
	}
	return res.Bool(), nil
}

// Has an element that matches the css selector
func (p *Page) Has(selector string) bool {
	has, err := p.HasE(selector)
	kit.E(err)
	return has
}

// ElementE ...
func (p *Page) ElementE(selector string) (*Element, error) {
	return p.ElementByJSE("", `s => document.querySelector(s)`, []interface{}{selector})
}

// Element waits and returns the first element in the page that matches the selector
func (p *Page) Element(selector string) *Element {
	el, err := p.ElementE(selector)
	kit.E(err)
	return el
}

// ElementByJSE ...
func (p *Page) ElementByJSE(thisID, js string, params []interface{}) (*Element, error) {
	var objectID string
	err := cdp.Retry(p.ctx, func() error {
		element, err := p.EvalE(false, thisID, js, params)
		if err != nil {
			return err
		}

		objectID = element.Get("result.objectId").String()
		if objectID == "" {
			return cdp.ErrNotYet
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Element{
		page:     p,
		ctx:      p.ctx,
		ObjectID: objectID,
	}, nil
}

// ElementByJS waits and returns the element from the return value of the js
func (p *Page) ElementByJS(js string, params ...interface{}) *Element {
	el, err := p.ElementByJSE("", js, params)
	kit.E(err)
	return el
}

// ElementsE ...
func (p *Page) ElementsE(selector string) ([]*Element, error) {
	return p.ElementsByJSE("", `s => document.querySelectorAll(s)`, []interface{}{selector})
}

// Elements returns all elements that match the selector
func (p *Page) Elements(selector string) []*Element {
	list, err := p.ElementsE(selector)
	kit.E(err)
	return list
}

// ElementsByJSE ...
func (p *Page) ElementsByJSE(thisID, js string, params []interface{}) ([]*Element, error) {
	elemList := []*Element{}
	err := cdp.Retry(p.ctx, func() error {
		res, err := p.EvalE(false, thisID, js, params)
		if err != nil {
			return err
		}
		defer p.ReleaseObject(res)

		list, err := p.Call("Runtime.getProperties", cdp.Object{
			"objectId":      res.Get("result.objectId").String(),
			"ownProperties": true,
		})
		if err != nil {
			return err
		}

		for _, obj := range list.Get("result").Array() {
			if obj.Get("name").String() == "__proto__" {
				continue
			}

			elemList = append(elemList, &Element{
				page:     p,
				ctx:      p.ctx,
				ObjectID: obj.Get("value.objectId").String(),
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return elemList, nil
}

// ElementsByJS returns the elements from the return value of the js
func (p *Page) ElementsByJS(js string, params ...interface{}) []*Element {
	list, err := p.ElementsByJSE("", js, params)
	kit.E(err)
	return list
}

// WaitEventE ...
func (p *Page) WaitEventE(name string) (kit.JSONResult, error) {
	msg, err := p.browser.Event().Until(p.ctx, func(e kit.Event) bool {
		return e.(*cdp.Message).Method == name
	})
	if err != nil {
		return nil, err
	}
	return kit.JSON(kit.MustToJSON(msg.(*cdp.Message).Params)), nil
}

// WaitEvent waits for the next event to happen.
// Example event names: Page.javascriptDialogOpening, Page.frameNavigated, DOM.attributeModified
func (p *Page) WaitEvent(name string) kit.JSONResult {
	res, err := p.WaitEventE(name)
	kit.E(err)
	return res
}

// HandleDialogE ...
func (p *Page) HandleDialogE(accept bool, promptText string) error {
	_, err := p.Call("Page.handleJavaScriptDialog", cdp.Object{
		"accept":     accept,
		"promptText": promptText,
	})
	return err
}

// HandleDialog accepts or dismisses a JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload)
func (p *Page) HandleDialog(accept bool, promptText string) {
	kit.E(p.HandleDialogE(accept, promptText))
}

// WaitPageE ...
func (p *Page) WaitPageE() (*Page, error) {
	var targetInfo cdp.Object

	_, err := p.browser.event.Until(p.ctx, func(e kit.Event) bool {
		msg := e.(*cdp.Message)
		if msg.Method == "Target.targetCreated" {
			targetInfo = msg.Params.(map[string]interface{})["targetInfo"].(map[string]interface{})

			if targetInfo["openerId"] == p.TargetID {
				return true
			}
		}
		return false
	})

	if err != nil {
		return nil, err
	}

	return p.browser.page(targetInfo["targetId"].(string))
}

// WaitPage to be opened from the specified page
func (p *Page) WaitPage() *Page {
	newPage, err := p.WaitPageE()
	kit.E(err)
	return newPage
}

// EvalE thisID is the remote objectID that will be the this of the js function
func (p *Page) EvalE(byValue bool, thisID, js string, jsArgs []interface{}) (res kit.JSONResult, err error) {
	if thisID == "" {
		res, err = p.eval(byValue, js, jsArgs)
	} else {
		res, err = p.evalThis(byValue, thisID, js, jsArgs)
	}

	if err != nil {
		return nil, err
	}

	if byValue {
		return FnResult(res)
	}

	return
}

func (p *Page) eval(byValue bool, js string, jsArgs []interface{}) (res kit.JSONResult, err error) {
	params := cdp.Object{
		"expression":    SprintFnApply(js, jsArgs),
		"awaitPromise":  true,
		"returnByValue": byValue,
	}

	if p.isIframe() {
		params["contextId"] = p.ContextID
	}

	return p.Call("Runtime.evaluate", params)
}

func (p *Page) evalThis(byValue bool, thisID, js string, jsArgs []interface{}) (kit.JSONResult, error) {
	args := []interface{}{}
	for _, p := range jsArgs {
		args = append(args, cdp.Object{"value": p})
	}

	params := cdp.Object{
		"objectId":            thisID,
		"awaitPromise":        true,
		"returnByValue":       byValue,
		"functionDeclaration": SprintFnThis(js),
		"arguments":           args,
	}

	return p.Call("Runtime.callFunctionOn", params)
}

// Eval js under sessionID or contextId, if contextId doesn't exist create a new isolatedWorld.
// The first param must be a js function definition.
// For example: page.Eval(`s => document.querySelectorAll(s)`, "input")
func (p *Page) Eval(js string, params ...interface{}) kit.JSONResult {
	res, err := p.EvalE(true, "", js, params)
	kit.E(err)
	return res
}

// Call sends a control message to the browser with the page session, the call is always on the root frame.
func (p *Page) Call(method string, params interface{}) (kit.JSONResult, error) {
	return p.browser.Ctx(p.ctx).Call(&cdp.Message{
		SessionID: p.SessionID,
		Method:    method,
		Params:    params,
	})
}

// ReleaseObject remote object
func (p *Page) ReleaseObject(obj kit.JSONResult) {
	_, err := p.Call("Runtime.releaseObject", cdp.Object{
		"objectId": obj.Get("result.objectId").String(),
	})
	if err != nil {
		p.browser.fatal.Publish(err)
	}
}

func (p *Page) initIsolatedWorld() error {
	frame, err := p.Call("Page.createIsolatedWorld", cdp.Object{
		"frameId": p.FrameID,
	})
	if err != nil {
		return err
	}

	p.ContextID = frame.Get("executionContextId").Int()
	return nil
}

func (p *Page) initSession() error {
	obj, err := p.Call("Target.attachToTarget", cdp.Object{
		"targetId": p.TargetID,
		"flatten":  true, // if it's not set no response will return
	})
	if err != nil {
		return err
	}
	p.SessionID = obj.Get("sessionId").String()
	_, err = p.Call("Page.enable", nil)
	return err
}

func (p *Page) isIframe() bool {
	return p.FrameID != ""
}

func (p *Page) rootFrame() *Page {
	f := p

	for f.isIframe() {
		f = f.element.page
	}

	return f
}
