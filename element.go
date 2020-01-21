package rod

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Element represents the DOM element
type Element struct {
	ctx      context.Context
	page     *Page
	objectID string

	cancel func()
}

// Ctx sets the context for later operation
func (el *Element) Ctx(ctx context.Context) *Element {
	newP := *el
	newP.ctx = ctx
	return &newP
}

// Timeout sets the timeout for later operation
func (el *Element) Timeout(d time.Duration) *Element {
	ctx, cancel := context.WithTimeout(el.ctx, d)
	el.cancel = cancel
	return el.Ctx(ctx)
}

func (el *Element) describe() (kit.JSONResult, error) {
	node, err := el.page.Call(el.ctx,
		"DOM.describeNode",
		cdp.Object{
			"objectId": el.objectID,
		},
	)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// FrameE ...
func (el *Element) FrameE() (*Page, error) {
	node, err := el.describe()
	if err != nil {
		return nil, err
	}

	newPage := *el.page
	newPage.frameID = node.Get("node.frameId").String()
	newPage.element = el

	return &newPage, newPage.initIsolatedWorld()
}

// Frame creates a page instance that represents the iframe
func (el *Element) Frame() *Page {
	f, err := el.FrameE()
	kit.E(err)
	return f
}

// HTMLE ...
func (el *Element) HTMLE() (string, error) {
	html, err := el.page.Call(el.ctx,
		"DOM.getOuterHTML",
		cdp.Object{
			"objectId": el.objectID,
		},
	)
	return html.Get("outerHTML").String(), err
}

// HTML gets the html of the element
func (el *Element) HTML() string {
	s, err := el.HTMLE()
	kit.E(err)
	return s
}

// ClickE ...
func (el *Element) ClickE() error {
	rect, err := el.RectE()
	if err != nil {
		return err
	}

	x := rect.Get("x").Int() + rect.Get("width").Int()/2
	y := rect.Get("y").Int() + rect.Get("height").Int()/2

	// use 2 mouseMoved to simulate mouse hover event
	_, err = el.page.Call(el.ctx, "Input.dispatchMouseEvent", cdp.Object{
		"type": "mouseMoved",
		"x":    0,
		"y":    0,
	})
	if err != nil {
		return err
	}

	_, err = el.page.Call(el.ctx, "Input.dispatchMouseEvent", cdp.Object{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	if err != nil {
		return err
	}

	_, err = el.page.Call(el.ctx, "Input.dispatchMouseEvent", cdp.Object{
		"type":       "mousePressed",
		"button":     "left",
		"clickCount": 1,
		"x":          x,
		"y":          y,
	})
	if err != nil {
		return err
	}

	_, err = el.page.Call(el.ctx, "Input.dispatchMouseEvent", cdp.Object{
		"type":       "mouseReleased",
		"button":     "left",
		"clickCount": 1,
		"x":          x,
		"y":          y,
	})
	return err
}

// Click the element
func (el *Element) Click() {
	kit.E(el.ClickE())
}

// RectE ...
func (el *Element) RectE() (kit.JSONResult, error) {
	res, err := el.FuncE(true, "function() { return this.getBoundingClientRect().toJSON() }")
	if err != nil {
		return nil, err
	}
	rect := res.Get("result.value")

	var j map[string]interface{}
	json.Unmarshal([]byte(rect.String()), &j)

	if el.page.isIframe() {
		frameRect, err := el.page.element.RectE() // recursively get the rect
		if err != nil {
			return nil, err
		}
		j["x"] = rect.Get("x").Int() + frameRect.Get("x").Int()
		j["y"] = rect.Get("y").Int() + frameRect.Get("y").Int()
	}
	return kit.JSON(kit.MustToJSON(j)), nil
}

// Rect returns the size of an element and its position relative to the main frame.
// It will recursively calculate the rect with all ancestors. The spec is here:
// https://developer.mozilla.org/en-US/docs/Web/API/Element/getBoundingClientRect
func (el *Element) Rect() kit.JSONResult {
	rect, err := el.RectE()
	kit.E(err)
	return rect
}

// FuncE ...
func (el *Element) FuncE(byValue bool, fn string) (kit.JSONResult, error) {
	return el.page.Call(el.ctx, "Runtime.callFunctionOn", cdp.Object{
		"objectId":            el.objectID,
		"awaitPromise":        true,
		"returnByValue":       byValue,
		"functionDeclaration": fn,
	})
}

// Func calls function on the element
func (el *Element) Func(fn string) kit.JSONResult {
	res, err := el.FuncE(true, fn)
	kit.E(err)
	return res
}
