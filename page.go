package rod

import (
	"context"
	"errors"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Page represents the webpage
type Page struct {
	ctx       context.Context
	browser   *Browser
	targetID  string
	sessionID string
	contextID int64

	// iframe
	frameID string
	element *Element

	cancel func()
}

func (p *Page) isIframe() bool {
	return p.frameID != ""
}

// Ctx sets the context for later operation
func (p *Page) Ctx(ctx context.Context) *Page {
	newP := *p
	newP.ctx = ctx
	return &newP
}

// Timeout sets the timeout for later operation
func (p *Page) Timeout(d time.Duration) *Page {
	ctx, cancel := context.WithTimeout(p.ctx, d)
	p.cancel = cancel
	return p.Ctx(ctx)
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
	res, err := p.EvalE(true, docQuerySelector(selector))
	if err != nil {
		return false, err
	}
	return res.Get("result.objectId").String() == "", nil
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
		element, err := p.EvalE(false, docQuerySelector(selector))
		if err != nil {
			return err
		}

		objectID = element.Get("result.objectId").String()
		if objectID == "" {
			return errors.New("not yet")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Element{
		page:     p,
		ctx:      p.ctx,
		objectID: objectID,
	}, nil
}

// Element returns the first element in the page that matches the selector
func (p *Page) Element(selector string) *Element {
	el, err := p.ElementE(selector)
	kit.E(err)
	return el
}

// EvalE ...
func (p *Page) EvalE(byValue bool, code string) (res kit.JSONResult, err error) {
	params := cdp.Object{
		"expression":    code,
		"awaitPromise":  true,
		"returnByValue": byValue,
	}

	err = cdp.Retry(p.ctx, func() error {
		if p.isIframe() {
			params["contextId"] = p.contextID
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

// Call client with page session
func (p *Page) Call(ctx context.Context, method string, params cdp.Object) (kit.JSONResult, error) {
	return p.browser.client.Call(ctx, &cdp.Message{
		SessionID: p.sessionID,
		Method:    method,
		Params:    params,
	})
}

// Eval runs script under sessionID or contextId, if contextId doesn't
// exist create a new isolatedWorld
func (p *Page) Eval(code string) kit.JSONResult {
	res, err := p.EvalE(true, code)
	kit.E(err)
	return res
}

func (p *Page) initIsolatedWorld() error {
	frame, err := p.Call(p.ctx, "Page.createIsolatedWorld", cdp.Object{
		"frameId": p.frameID,
	})
	if err != nil {
		return err
	}

	p.contextID = frame.Get("executionContextId").Int()
	return nil
}

func (p *Page) initSession() error {
	obj, err := p.Call(p.ctx, "Target.attachToTarget", cdp.Object{
		"targetId": p.targetID,
		"flatten":  true, // if it's not set no response will return
	})
	if err != nil {
		return err
	}
	p.sessionID = obj.Get("sessionId").String()
	return nil
}
