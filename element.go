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
	ObjectID string

	timeoutCancel func()
}

// Ctx sets the context for later operation
func (el *Element) Ctx(ctx context.Context) *Element {
	newObj := *el
	newObj.ctx = ctx
	return &newObj
}

// Timeout sets the timeout for later operation
func (el *Element) Timeout(d time.Duration) *Element {
	ctx, cancel := context.WithTimeout(el.ctx, d)
	el.timeoutCancel = cancel
	return el.Ctx(ctx)
}

// CancelTimeout ...
func (el *Element) CancelTimeout() {
	if el.timeoutCancel != nil {
		el.timeoutCancel()
	}
}

func (el *Element) describe() (kit.JSONResult, error) {
	node, err := el.page.Call(el.ctx,
		"DOM.describeNode",
		cdp.Object{
			"objectId": el.ObjectID,
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
	newPage.FrameID = node.Get("node.frameId").String()
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
			"objectId": el.ObjectID,
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

// ScrollIntoViewIfNeededE ...
func (el *Element) ScrollIntoViewIfNeededE(opts cdp.Object) error {
	_, err := el.FuncE(false, Fn(`function(opts) { this.scrollIntoViewIfNeeded(opts) }`, opts))
	return err
}

// ScrollIntoViewIfNeeded scrolls the current element into the visible area of the browser
// window if it's not already within the visible area.
func (el *Element) ScrollIntoViewIfNeeded(opts cdp.Object) {
	kit.E(el.ScrollIntoViewIfNeededE(opts))
}

// ClickE ...
func (el *Element) ClickE() error {
	err := el.ScrollIntoViewIfNeededE(nil)
	if err != nil {
		return err
	}

	rect, err := el.RectE()
	if err != nil {
		return err
	}

	x := rect.Get("left").Int() + rect.Get("width").Int()/2
	y := rect.Get("top").Int() + rect.Get("height").Int()/2

	err = el.page.mouse.MoveToE(x, y)
	if err != nil {
		return err
	}

	return el.page.mouse.ClickE("")
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
	kit.E(json.Unmarshal([]byte(rect.String()), &j))

	if el.page.isIframe() {
		frameRect, err := el.page.element.RectE() // recursively get the rect
		if err != nil {
			return nil, err
		}
		j["left"] = rect.Get("left").Int() + frameRect.Get("left").Int()
		j["top"] = rect.Get("top").Int() + frameRect.Get("top").Int()
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
		"objectId":            el.ObjectID,
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
