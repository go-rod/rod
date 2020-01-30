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
	mouse    *Mouse
	keyboard *Keyboard

	// iframe
	FrameID string
	element *Element

	timeoutCancel func()
}

func (p *Page) isIframe() bool {
	return p.FrameID != ""
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
	_, err := p.Call(p.ctx, "Page.navigate", cdp.Object{
		"url": url,
	})
	return err
}

// Navigate to url
func (p *Page) Navigate(url string) *Page {
	kit.E(p.NavigateE(url))
	return p
}

// CloseE page
func (p *Page) CloseE() error {
	_, err := p.Call(p.ctx, "Page.close", nil)
	return err
}

// Close page
func (p *Page) Close() {
	kit.E(p.CloseE())
}

// HasE ...
func (p *Page) HasE(selector string) (bool, error) {
	res, err := p.EvalE(false, `s => document.querySelector(s)`, selector)
	if err != nil {
		return false, err
	}
	return res.Get("result.objectId").String() != "", nil
}

// Has an element that matches the css selector
func (p *Page) Has(selector string) bool {
	has, err := p.HasE(selector)
	kit.E(err)
	return has
}

// ElementE ...
func (p *Page) ElementE(selector string) (*Element, error) {
	var objectID string
	err := cdp.Retry(p.ctx, func() error {
		element, err := p.EvalE(false, `s => document.querySelector(s)`, selector)
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

// Element returns the first element in the page that matches the selector
func (p *Page) Element(selector string) *Element {
	el, err := p.ElementE(selector)
	kit.E(err)
	return el
}

// EvalE ...
func (p *Page) EvalE(byValue bool, js string, jsParams ...interface{}) (res kit.JSONResult, err error) {
	params := cdp.Object{
		"expression":    SprintFn(js, jsParams...),
		"awaitPromise":  true,
		"returnByValue": byValue,
	}

	err = cdp.Retry(p.ctx, func() error {
		if p.isIframe() {
			params["contextId"] = p.ContextID
		}

		res, err = p.Call(p.ctx, "Runtime.evaluate", params)
		if err == nil {
			return nil
		}

		if cdpErr, ok := err.(*cdp.Error); ok && cdpErr.Code == -32000 {
			_ = p.initIsolatedWorld()
		}

		return err
	})

	return
}

// Eval runs script under sessionID or contextId, if contextId doesn't
// exist create a new isolatedWorld
func (p *Page) Eval(js string, params ...interface{}) kit.JSONResult {
	res, err := p.EvalE(true, js, params)
	kit.E(err)
	return res
}

// Call client with page session, the call is always on the root frame.
func (p *Page) Call(ctx context.Context, method string, params cdp.Object) (kit.JSONResult, error) {
	return p.browser.Call(ctx, &cdp.Message{
		SessionID: p.SessionID,
		Method:    method,
		Params:    params,
	})
}

func (p *Page) initIsolatedWorld() error {
	frame, err := p.Call(p.ctx, "Page.createIsolatedWorld", cdp.Object{
		"frameId": p.FrameID,
	})
	if err != nil {
		return err
	}

	p.ContextID = frame.Get("executionContextId").Int()
	return nil
}

func (p *Page) initSession() error {
	obj, err := p.Call(p.ctx, "Target.attachToTarget", cdp.Object{
		"targetId": p.TargetID,
		"flatten":  true, // if it's not set no response will return
	})
	if err != nil {
		return err
	}
	p.SessionID = obj.Get("sessionId").String()
	return nil
}
