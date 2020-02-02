package rod

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Element represents the DOM element
type Element struct {
	ctx  context.Context
	page *Page

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

// ScrollIntoViewIfNeededE ...
func (el *Element) ScrollIntoViewIfNeededE(opts cdp.Object) error {
	_, err := el.EvalE(true, `opts => this.scrollIntoViewIfNeeded(opts)`, opts)
	return err
}

// ScrollIntoViewIfNeeded scrolls the current element into the visible area of the browser
// window if it's not already within the visible area.
func (el *Element) ScrollIntoViewIfNeeded(opts cdp.Object) {
	kit.E(el.ScrollIntoViewIfNeededE(opts))
}

// ClickE ...
func (el *Element) ClickE(button string) error {
	err := el.ScrollIntoViewIfNeededE(nil)
	if err != nil {
		return err
	}

	box, err := el.BoxE()
	if err != nil {
		return err
	}

	x := box.Get("left").Int() + box.Get("width").Int()/2
	y := box.Get("top").Int() + box.Get("height").Int()/2

	err = el.page.Mouse.MoveToE(x, y)
	if err != nil {
		return err
	}

	defer el.trace(button + " click")()

	return el.page.Mouse.ClickE(button)
}

// Click the element
func (el *Element) Click() {
	kit.E(el.ClickE("left"))
}

// PressE ...
func (el *Element) PressE(key string) error {
	err := el.ClickE("left")
	if err != nil {
		return err
	}

	defer el.trace("press " + key)()

	return el.page.Keyboard.PressE(key)
}

// Press a key
func (el *Element) Press(key string) {
	kit.E(el.PressE(key))
}

// InputE ...
func (el *Element) InputE(text string) error {
	err := el.ClickE("left")
	if err != nil {
		return err
	}

	defer el.trace("input " + text)()

	err = el.page.Keyboard.TextE(text)
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, `() => {
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
	}`)
	return err
}

// Input wll click the element and input the text
func (el *Element) Input(text string) {
	kit.E(el.InputE(text))
}

// SelectE ...
func (el *Element) SelectE(selectors ...string) error {
	defer el.trace(fmt.Sprintf(
		`<span style="color: #777;">select</span> <code>%s</code>`,
		strings.Join(selectors, "; ")))()
	el.page.browser.slowmotion("Input.select")

	_, err := el.EvalE(true, `selectors => {
		selectors.forEach(s => {
			Array.from(this.options).forEach(el => {
				if (el.innerText === s || el.matches(s)) {
					el.selected = true
				}
			})
		})
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
	}`, selectors)
	return err
}

// Select the option elements that match the selectors, the selector can be text content or css selector
func (el *Element) Select(selectors ...string) {
	kit.E(el.SelectE(selectors...))
}

// TextE ...
func (el *Element) TextE() (string, error) {
	str, err := el.EvalE(true, `() => this.innerText`)
	return str.String(), err
}

// Text gets the innerText of the element
func (el *Element) Text() string {
	s, err := el.TextE()
	kit.E(err)
	return s
}

// HTMLE ...
func (el *Element) HTMLE() (string, error) {
	str, err := el.EvalE(true, `() => this.outerHTML`)
	return str.String(), err
}

// HTML gets the outerHTML of the element
func (el *Element) HTML() string {
	s, err := el.HTMLE()
	kit.E(err)
	return s
}

// WaitE ...
func (el *Element) WaitE(js string, params ...interface{}) error {
	return cdp.Retry(el.ctx, func() error {
		res, err := el.EvalE(true, js, params...)
		if err != nil {
			return err
		}

		if res.Bool() {
			return nil
		}

		return cdp.ErrNotYet
	})
}

// Wait until the js returns true
func (el *Element) Wait(js string, params ...interface{}) {
	kit.E(el.WaitE(js, params))
}

// WaitVisibleE ...
func (el *Element) WaitVisibleE() error {
	return el.WaitE(`() => {
		var box = this.getBoundingClientRect()
		var style = window.getComputedStyle(this)
		return style.display != 'none' &&
			style.visibility != 'hidden' &&
			!!(box.top || box.bottom || box.width || box.height)
	}`)
}

// WaitVisible until the element is visible
func (el *Element) WaitVisible() {
	kit.E(el.WaitVisibleE())
}

// WaitInvisibleE ...
func (el *Element) WaitInvisibleE() error {
	return el.WaitE(`() => {
		var box = this.getBoundingClientRect()
		return window.getComputedStyle(this).visibility == 'hidden' ||
			!(box.top || box.bottom || box.width || box.height)
	}`)
}

// WaitInvisible until the element is not visible or removed
func (el *Element) WaitInvisible() {
	kit.E(el.WaitInvisibleE())
}

// BoxE ...
func (el *Element) BoxE() (kit.JSONResult, error) {
	box, err := el.EvalE(true, `() => {
		var box = this.getBoundingClientRect().toJSON()
		if (this.tagName === 'IFRAME') {
			var style = window.getComputedStyle(this)
			box.left += parseInt(style.paddingLeft) + parseInt(style.borderLeftWidth)
			box.top += parseInt(style.paddingTop) + parseInt(style.borderTopWidth)
		}
		return box
	}`)
	if err != nil {
		return nil, err
	}

	var j map[string]interface{}
	kit.E(json.Unmarshal([]byte(box.String()), &j))

	if el.page.isIframe() {
		frameRect, err := el.page.element.BoxE() // recursively get the box
		if err != nil {
			return nil, err
		}
		j["left"] = box.Get("left").Int() + frameRect.Get("left").Int()
		j["top"] = box.Get("top").Int() + frameRect.Get("top").Int()
	}
	return kit.JSON(kit.MustToJSON(j)), nil
}

// Box returns the size of an element and its position relative to the main frame.
// It will recursively calculate the box with all ancestors. The spec is here:
// https://developer.mozilla.org/en-US/docs/Web/API/Element/getBoundingClientRect
func (el *Element) Box() kit.JSONResult {
	box, err := el.BoxE()
	kit.E(err)
	return box
}

// EvalE ...
func (el *Element) EvalE(byValue bool, js string, params ...interface{}) (kit.JSONResult, error) {
	args := []interface{}{}

	for _, p := range params {
		args = append(args, cdp.Object{"value": p})
	}

	js = fmt.Sprintf(`function() {
		return (%s).apply(this, arguments)
	}`, js)

	res, err := el.page.Call(el.ctx, "Runtime.callFunctionOn", cdp.Object{
		"objectId":            el.ObjectID,
		"awaitPromise":        true,
		"returnByValue":       byValue,
		"functionDeclaration": js,
		"arguments":           args,
	})
	if err != nil {
		return nil, err
	}

	if byValue {
		return FnResult(res)
	}

	return res, nil
}

// Eval evaluates js function on the element, the first param must be a js function definition
// For example: el.Eval(`name => this.getAttribute(name)`, "value")
func (el *Element) Eval(js string, params ...interface{}) kit.JSONResult {
	res, err := el.EvalE(true, js, params...)
	kit.E(err)
	return res
}
