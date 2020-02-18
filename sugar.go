package rod

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Connect to the browser and start to control it.
// If fails to connect, try to run a local browser, if local browser not found try to download one.
func (b *Browser) Connect() *Browser {
	kit.E(b.ConnectE())
	return b
}

// Close the browser and release related resources
func (b *Browser) Close() {
	kit.E(b.CloseE())
}

// Page creates a new tab
func (b *Browser) Page(url string) *Page {
	p, err := b.PageE(url)
	kit.E(err)
	return p
}

// Pages returns all visible pages
func (b *Browser) Pages() []*Page {
	list, err := b.PagesE()
	kit.E(err)
	return list
}

// WaitEvent resolves the wait function when the filter returns true, call cancel to release the resource
func (b *Browser) WaitEvent(name string) (wait func() *cdp.Event, cancel func()) {
	w, c := b.WaitEventE(Method(name))
	return func() *cdp.Event {
		e, err := w()
		kit.E(err)
		return e
	}, c
}

// Call sends a control message to browser
func (b *Browser) Call(req *cdp.Request) kit.JSONResult {
	res, err := b.CallE(req)
	kit.E(err)
	return res
}

// Navigate to url
func (p *Page) Navigate(url string) *Page {
	kit.E(p.NavigateE(url))
	return p
}

// GetWindow get window bounds
func (p *Page) GetWindow() kit.JSONResult {
	bounds, err := p.GetWindowE()
	kit.E(err)
	return bounds
}

// Window set window bounds. The state must be one of normal, minimized, maximized, fullscreen.
func (p *Page) Window(left, top, width, height int64, state string) *Page {
	kit.E(p.WindowE(&cdp.Object{
		"left":        left,
		"top":         top,
		"width":       width,
		"height":      height,
		"windowState": state,
	}))
	return p
}

// Viewport overrides the values of device screen dimensions.
func (p *Page) Viewport(width, height int64, deviceScaleFactor float32, mobile bool) *Page {
	kit.E(p.ViewportE(&cdp.Object{
		"width":             width,
		"height":            height,
		"deviceScaleFactor": deviceScaleFactor,
		"mobile":            mobile,
	}))
	return p
}

// Close page
func (p *Page) Close() {
	kit.E(p.CloseE())
}

// HandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload)
func (p *Page) HandleDialog(accept bool, promptText string) (wait func(), cancel func()) {
	w, c := p.HandleDialogE(accept, promptText)
	return func() {
		kit.E(w())
	}, c
}

// GetDownloadFile of the next download url that matches the pattern, returns the response header and file content.
// Wildcards ('*' -> zero or more, '?' -> exactly one) are allowed. Escape character is backslash. Omitting is equivalent to "*".
func (p *Page) GetDownloadFile(pattern string) (wait func() (http.Header, []byte), cancel func()) {
	w, c, err := p.GetDownloadFileE(filepath.FromSlash("tmp/rod-downloads"), pattern)
	kit.E(err)
	return func() (http.Header, []byte) {
		header, data, err := w()
		kit.E(err)
		return header, data
	}, c
}

// Screenshot the page
func (p *Page) Screenshot() []byte {
	png, err := p.ScreenshotE(nil)
	kit.E(err)
	return png
}

// WaitPage to be opened from the specified page
func (p *Page) WaitPage() (wait func() *Page, cancel func()) {
	w, c := p.WaitPageE()
	return func() *Page {
		page, err := w()
		kit.E(err)
		return page
	}, c
}

// Pause stops on the next JavaScript statement
func (p *Page) Pause() *Page {
	kit.E(p.PauseE())
	return p
}

// WaitLoad wait until the `window.onload` is complete, resolve immediately if already fired.
func (p *Page) WaitLoad() *Page {
	kit.E(p.WaitLoadE())
	return p
}

// WaitEvent waits for the next event to happen.
func (p *Page) WaitEvent(name string) (wait func(), cancel func()) {
	w, c := p.WaitEventE(Method(name))
	return func() { kit.E(w()) }, c
}

// Eval js under sessionID or contextId, if contextId doesn't exist create a new isolatedWorld.
// The first param must be a js function definition.
// For example: page.Eval(`s => document.querySelectorAll(s)`, "input")
func (p *Page) Eval(js string, params ...interface{}) kit.JSONResult {
	res, err := p.EvalE(true, "", js, params)
	kit.E(err)
	return res
}

// Release remote object
func (p *Page) Release(objectID string) *Page {
	kit.E(p.ReleaseE(objectID))
	return p
}

// Call sends a control message to the browser with the page session, the call is always on the root frame.
func (p *Page) Call(method string, params interface{}) kit.JSONResult {
	res, err := p.CallE(method, params)
	kit.E(err)
	return res
}

// Describe returns the element info
// Returned json: https://chromedevtools.github.io/devtools-protocol/tot/DOM#type-Node
func (el *Element) Describe() kit.JSONResult {
	node, err := el.DescribeE()
	kit.E(err)
	return node
}

// Frame creates a page instance that represents the iframe
func (el *Element) Frame() *Page {
	f, err := el.FrameE()
	kit.E(err)
	return f
}

// Focus sets focus on the specified element
func (el *Element) Focus() *Element {
	kit.E(el.FocusE())
	return el
}

// ScrollIntoViewIfNeeded scrolls the current element into the visible area of the browser
// window if it's not already within the visible area.
func (el *Element) ScrollIntoViewIfNeeded() *Element {
	kit.E(el.ScrollIntoViewIfNeededE())
	return el
}

// Click the element
func (el *Element) Click() *Element {
	kit.E(el.ClickE("left"))
	return el
}

// Press a key
func (el *Element) Press(key rune) *Element {
	kit.E(el.PressE(key))
	return el
}

// Input wll click the element and input the text
func (el *Element) Input(text string) *Element {
	kit.E(el.InputE(text))
	return el
}

// Select the option elements that match the selectors, the selector can be text content or css selector
func (el *Element) Select(selectors ...string) *Element {
	kit.E(el.SelectE(selectors...))
	return el
}

// SetFiles sets files for the given file input element
func (el *Element) SetFiles(paths ...string) *Element {
	kit.E(el.SetFilesE(paths))
	return el
}

// Text gets the innerText of the element
func (el *Element) Text() string {
	s, err := el.TextE()
	kit.E(err)
	return s
}

// HTML gets the outerHTML of the element
func (el *Element) HTML() string {
	s, err := el.HTMLE()
	kit.E(err)
	return s
}

// WaitStable waits until the size and position are stable. Useful when waiting for the animation of modal
// or button to complete so that we can simulate the mouse to move to it and click on it.
func (el *Element) WaitStable() *Element {
	kit.E(el.WaitStableE(100 * time.Millisecond))
	return el
}

// Wait until the js returns true
func (el *Element) Wait(js string, params ...interface{}) *Element {
	kit.E(el.WaitE(js, params))
	return el
}

// WaitVisible until the element is visible
func (el *Element) WaitVisible() *Element {
	kit.E(el.WaitVisibleE())
	return el
}

// WaitInvisible until the element is not visible or removed
func (el *Element) WaitInvisible() *Element {
	kit.E(el.WaitInvisibleE())
	return el
}

// Box returns the size of an element and its position relative to the main frame.
// It will recursively calculate the box with all ancestors. The spec is here:
// https://developer.mozilla.org/en-US/docs/Web/API/Element/getBoundingClientRect
func (el *Element) Box() kit.JSONResult {
	box, err := el.BoxE()
	kit.E(err)
	return box
}

// Resource returns the binary of the "src" properly, such as the image or audio file.
func (el *Element) Resource() []byte {
	bin, err := el.ResourceE()
	kit.E(err)
	return bin
}

// Release remote object on browser
func (el *Element) Release() {
	kit.E(el.ReleaseE())
}

// Eval evaluates js function on the element, the first param must be a js function definition
// For example: el.Eval(`name => this.getAttribute(name)`, "value")
func (el *Element) Eval(js string, params ...interface{}) kit.JSONResult {
	res, err := el.EvalE(true, js, params...)
	kit.E(err)
	return res
}
