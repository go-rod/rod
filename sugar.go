// This file contains the methods that panics when error return value is not nil.

package rod

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/input"
	"github.com/ysmood/rod/lib/proto"
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

// EachEvent of the specified event type, if the fn returns true the event loop will stop.
func (b *Browser) EachEvent(fn interface{}) {
	eachEvent(b.Event(), fn)
}

// Incognito creates a new incognito browser
func (b *Browser) Incognito() *Browser {
	b, err := b.IncognitoE()
	kit.E(err)
	return b
}

// Page creates a new tab
func (b *Browser) Page(url string) *Page {
	p, err := b.PageE(url)
	kit.E(err)
	return p
}

// Pages returns all visible pages
func (b *Browser) Pages() Pages {
	list, err := b.PagesE()
	kit.E(err)
	return list
}

// PageFromTargetID creates a Page instance from a targetID
func (b *Browser) PageFromTargetID(targetID proto.TargetTargetID) *Page {
	p, err := b.PageFromTargetIDE(targetID)
	kit.E(err)
	return p
}

// WaitEvent resolves the wait function when the filter returns true
func (b *Browser) WaitEvent(e proto.Event) (wait func()) {
	w := b.WaitEventE(NewEventFilter(e))
	return func() { kit.E(<-w) }
}

// HandleAuth for the next basic HTTP authentication.
// It will prevent the popup that requires user to input user name and password.
// Ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
func (b *Browser) HandleAuth(username, password string) {
	wait, err := b.HandleAuthE(username, password)
	kit.E(err)
	go func() { kit.E(wait()) }()
}

// Cookies returns the page cookies. By default it will return the cookies for current page.
// The urls is the list of URLs for which applicable cookies will be fetched.
func (p *Page) Cookies(urls ...string) []*proto.NetworkCookie {
	cookies, err := p.CookiesE(urls)
	kit.E(err)
	return cookies
}

// SetCookies of the page.
// Cookie format: https://chromedevtools.github.io/devtools-protocol/tot/Network#method-setCookie
func (p *Page) SetCookies(cookies ...*proto.NetworkCookieParam) *Page {
	kit.E(p.SetCookiesE(cookies))
	return p
}

// SetExtraHeaders whether to always send extra HTTP headers with the requests from this page.
// The arguments are key-value pairs, you can set multiple key-value pairs at the same time.
func (p *Page) SetExtraHeaders(dict ...string) *Page {
	kit.E(p.SetExtraHeadersE(dict))
	return p
}

// SetUserAgent Allows overriding user agent with the given string.
func (p *Page) SetUserAgent(req *proto.NetworkSetUserAgentOverride) *Page {
	kit.E(p.SetUserAgentE(req))
	return p
}

// Navigate to url
func (p *Page) Navigate(url string) *Page {
	kit.E(p.NavigateE(url))
	return p
}

// GetWindow get window bounds
func (p *Page) GetWindow() *proto.BrowserBounds {
	bounds, err := p.GetWindowE()
	kit.E(err)
	return bounds
}

// Window set the window location and size
func (p *Page) Window(left, top, width, height int64) *Page {
	kit.E(p.WindowE(&proto.BrowserBounds{
		Left:        left,
		Top:         top,
		Width:       width,
		Height:      height,
		WindowState: proto.BrowserWindowStateNormal,
	}))
	return p
}

// WindowMinimize the window
func (p *Page) WindowMinimize() *Page {
	kit.E(p.WindowE(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateMinimized,
	}))
	return p
}

// WindowMaximize the window
func (p *Page) WindowMaximize() *Page {
	kit.E(p.WindowE(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateMaximized,
	}))
	return p
}

// WindowFullscreen the window
func (p *Page) WindowFullscreen() *Page {
	kit.E(p.WindowE(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateFullscreen,
	}))
	return p
}

// WindowNormal the window size
func (p *Page) WindowNormal() *Page {
	kit.E(p.WindowE(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateNormal,
	}))
	return p
}

// Viewport overrides the values of device screen dimensions.
func (p *Page) Viewport(width, height int64, deviceScaleFactor float64, mobile bool) *Page {
	kit.E(p.ViewportE(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: deviceScaleFactor,
		Mobile:            mobile,
	}))
	return p
}

// StopLoading forces the page stop all navigations and pending resource fetches.
func (p *Page) StopLoading() *Page {
	kit.E(p.StopLoadingE())
	return p
}

// Close page
func (p *Page) Close() {
	kit.E(p.CloseE())
}

// EachEvent of the specified event type, if the fn returns true the event loop will stop.
func (p *Page) EachEvent(fn interface{}) {
	eachEvent(p.Event(), fn)
}

// HandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload)
func (p *Page) HandleDialog(accept bool, promptText string) (wait func()) {
	w := p.HandleDialogE(accept, promptText)
	return func() {
		kit.E(w())
	}
}

// GetDownloadFile of the next download url that matches the pattern, returns the response header and file content.
// Wildcards ('*' -> zero or more, '?' -> exactly one) are allowed. Escape character is backslash. Omitting is equivalent to "*".
func (p *Page) GetDownloadFile(pattern string) (wait func() (http.Header, []byte)) {
	w, err := p.GetDownloadFileE(filepath.FromSlash("tmp/rod-downloads"), pattern)
	kit.E(err)
	return func() (http.Header, []byte) {
		header, data, err := w()
		kit.E(err)
		return header, data
	}
}

// GetViewport returns the current viewport
func (p *Page) GetViewport() *proto.PageLayoutViewport {
	view, err := p.GetViewportE()
	kit.E(err)
	return view
}

// Screenshot the page and returns the binary of the image
// If the toFile is "", it will save output to "tmp/screenshots" folder, time as the file name.
func (p *Page) Screenshot(toFile ...string) []byte {
	bin, err := p.ScreenshotE(&proto.PageCaptureScreenshot{})
	kit.E(err)
	kit.E(saveScreenshot(bin, toFile))
	return bin
}

// PDF prints page as PDF
func (p *Page) PDF() []byte {
	pdf, err := p.PDFE(&proto.PagePrintToPDF{})
	kit.E(err)
	return pdf
}

// WaitPage to be created from a new window
func (p *Page) WaitPage() (wait func() *Page) {
	w := p.WaitPageE()
	return func() *Page {
		page, err := w()
		kit.E(err)
		return page
	}
}

// Pause stops on the next JavaScript statement
func (p *Page) Pause() *Page {
	kit.E(p.PauseE())
	return p
}

// WaitRequestIdle returns a wait function that waits until the page doesn't send request for 300ms.
// You can pass regular expressions to exclude the requests by their url.
func (p *Page) WaitRequestIdle(excludes ...string) (wait func()) {
	w := p.WaitRequestIdleE(300*time.Millisecond, []string{""}, excludes)
	return func() { kit.E(w()) }
}

// WaitIdle wait until the next window.requestIdleCallback is called.
func (p *Page) WaitIdle() *Page {
	kit.E(p.WaitIdleE(time.Minute))
	return p
}

// WaitLoad wait until the `window.onload` is complete, resolve immediately if already fired.
func (p *Page) WaitLoad() *Page {
	kit.E(p.WaitLoadE())
	return p
}

// WaitEvent returns a wait function that waits for the next event to happen.
func (p *Page) WaitEvent(e proto.Event) (wait func()) {
	w := p.WaitEventE(NewEventFilter(e))
	return func() { kit.E(<-w) }
}

// AddScriptTag to page. If url is empty, content will be used.
func (p *Page) AddScriptTag(url string) *Page {
	kit.E(p.AddScriptTagE(url, ""))
	return p
}

// AddStyleTag to page. If url is empty, content will be used.
func (p *Page) AddStyleTag(url string) *Page {
	kit.E(p.AddStyleTagE(url, ""))
	return p
}

// Eval js on the page. The first param must be a js function definition.
// For example page.Eval(`n => n + 1`, 1) will return 2
func (p *Page) Eval(js string, params ...interface{}) proto.JSON {
	res, err := p.EvalE(true, "", js, params)
	kit.E(err)
	return res.Value
}

// Release remote object
func (p *Page) Release(objectID proto.RuntimeRemoteObjectID) *Page {
	kit.E(p.ReleaseE(objectID))
	return p
}

// Has an element that matches the css selector
func (p *Page) Has(selector string) bool {
	has, err := p.HasE(selector)
	kit.E(err)
	return has
}

// HasX an element that matches the XPath selector
func (p *Page) HasX(selector string) bool {
	has, err := p.HasXE(selector)
	kit.E(err)
	return has
}

// HasMatches an element that matches the css selector and its text matches the regex.
func (p *Page) HasMatches(selector, regex string) bool {
	has, err := p.HasMatchesE(selector, regex)
	kit.E(err)
	return has
}

// Element retries until returns the first element in the page that matches the CSS selector
func (p *Page) Element(selector string) *Element {
	el, err := p.ElementE(p.Sleeper(), "", selector)
	kit.E(err)
	return el
}

// ElementMatches retries until returns the first element in the page that matches the CSS selector and its text matches the regex.
// The regex is the js regex, not golang's.
func (p *Page) ElementMatches(selector, regex string) *Element {
	el, err := p.ElementMatchesE(p.Sleeper(), "", selector, regex)
	kit.E(err)
	return el
}

// ElementByJS retries until returns the element from the return value of the js function
func (p *Page) ElementByJS(js string, params ...interface{}) *Element {
	el, err := p.ElementByJSE(p.Sleeper(), "", js, params)
	kit.E(err)
	return el
}

// Elements returns all elements that match the css selector
func (p *Page) Elements(selector string) Elements {
	list, err := p.ElementsE("", selector)
	kit.E(err)
	return list
}

// ElementsX returns all elements that match the XPath selector
func (p *Page) ElementsX(xpath string) Elements {
	list, err := p.ElementsXE("", xpath)
	kit.E(err)
	return list
}

// ElementX retries until returns the first element in the page that matches the XPath selector
func (p *Page) ElementX(xpath string) *Element {
	el, err := p.ElementXE(p.Sleeper(), "", xpath)
	kit.E(err)
	return el
}

// ElementsByJS returns the elements from the return value of the js
func (p *Page) ElementsByJS(js string, params ...interface{}) Elements {
	list, err := p.ElementsByJSE("", js, params)
	kit.E(err)
	return list
}

// Move to the location
func (m *Mouse) Move(x, y float64) {
	kit.E(m.MoveE(x, y, 0))
}

// Scroll the wheel
func (m *Mouse) Scroll(x, y float64) {
	kit.E(m.ScrollE(x, y, 0))
}

// Down button
func (m *Mouse) Down(button proto.InputMouseButton) {
	kit.E(m.DownE(button, 1))
}

// Up button
func (m *Mouse) Up(button proto.InputMouseButton) {
	kit.E(m.UpE(button, 1))
}

// Click button
func (m *Mouse) Click(button proto.InputMouseButton) {
	kit.E(m.ClickE(button))
}

// Down holds key down
func (k *Keyboard) Down(key rune) {
	kit.E(k.DownE(key))
}

// Up releases the key
func (k *Keyboard) Up(key rune) {
	kit.E(k.UpE(key))
}

// Press a key
func (k *Keyboard) Press(key rune) {
	if k.page.browser.trace {
		defer k.page.Overlay(0, 0, 200, 0, "press "+input.Keys[key].Key)()
	}

	kit.E(k.PressE(key))
}

// InsertText like paste text into the page
func (k *Keyboard) InsertText(text string) {
	kit.E(k.InsertTextE(text))
}

// Describe returns the element info
// Returned json: https://chromedevtools.github.io/devtools-protocol/tot/DOM#type-Node
func (el *Element) Describe() *proto.DOMNode {
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

// ShadowRoot returns the shadow root of this element
func (el *Element) ShadowRoot() *Element {
	node, err := el.ShadowRootE()
	kit.E(err)
	return node
}

// Focus sets focus on the specified element
func (el *Element) Focus() *Element {
	kit.E(el.FocusE())
	return el
}

// ScrollIntoView scrolls the current element into the visible area of the browser
// window if it's not already within the visible area.
func (el *Element) ScrollIntoView() *Element {
	kit.E(el.ScrollIntoViewE())
	return el
}

// Click the element
func (el *Element) Click() *Element {
	kit.E(el.ClickE(proto.InputMouseButtonLeft))
	return el
}

// Press a key
func (el *Element) Press(key rune) *Element {
	kit.E(el.PressE(key))
	return el
}

// SelectText selects the text that matches the regular expression
func (el *Element) SelectText(regex string) *Element {
	kit.E(el.SelectTextE(regex))
	return el
}

// SelectAllText selects all text
func (el *Element) SelectAllText() *Element {
	kit.E(el.SelectAllTextE())
	return el
}

// Input wll click the element and input the text.
// To empty the input you can use something like el.SelectAllText().Input("")
func (el *Element) Input(text string) *Element {
	kit.E(el.InputE(text))
	return el
}

// Select the option elements that match the selectors, the selector can be text content or css selector
func (el *Element) Select(selectors ...string) *Element {
	kit.E(el.SelectE(selectors))
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

// Visible returns true if the element is visible on the page
func (el *Element) Visible() bool {
	v, err := el.VisibleE()
	kit.E(err)
	return v
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
func (el *Element) Box() *Box {
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

// Screenshot of the area of the element
func (el *Element) Screenshot(toFile ...string) []byte {
	bin, err := el.ScreenshotE(proto.PageCaptureScreenshotFormatPng, -1)
	kit.E(err)
	kit.E(saveScreenshot(bin, toFile))
	return bin
}

// Release remote object on browser
func (el *Element) Release() {
	kit.E(el.ReleaseE())
}

// Eval evaluates js function on the element, the first param must be a js function definition
// For example: el.Eval(`name => this.getAttribute(name)`, "value")
func (el *Element) Eval(js string, params ...interface{}) proto.JSON {
	res, err := el.EvalE(true, js, params)
	kit.E(err)
	return res.Value
}

// Has an element that matches the css selector
func (el *Element) Has(selector string) bool {
	has, err := el.HasE(selector)
	kit.E(err)
	return has
}

// HasX an element that matches the XPath selector
func (el *Element) HasX(selector string) bool {
	has, err := el.HasXE(selector)
	kit.E(err)
	return has
}

// HasMatches an element that matches the css selector and its text matches the regex.
func (el *Element) HasMatches(selector, regex string) bool {
	has, err := el.HasMatchesE(selector, regex)
	kit.E(err)
	return has
}

// Element returns the first child that matches the css selector
func (el *Element) Element(selector string) *Element {
	el, err := el.ElementE(selector)
	kit.E(err)
	return el
}

// ElementX returns the first child that matches the XPath selector
func (el *Element) ElementX(xpath string) *Element {
	el, err := el.ElementXE(xpath)
	kit.E(err)
	return el
}

// ElementByJS returns the element from the return value of the js
func (el *Element) ElementByJS(js string, params ...interface{}) *Element {
	el, err := el.ElementByJSE(js, params)
	kit.E(err)
	return el
}

// Parent returns the parent element
func (el *Element) Parent() *Element {
	parent, err := el.ParentE()
	kit.E(err)
	return parent
}

// Parents that match the selector
func (el *Element) Parents(selector string) Elements {
	list, err := el.ParentsE(selector)
	kit.E(err)
	return list
}

// Next returns the next sibling element
func (el *Element) Next() *Element {
	parent, err := el.NextE()
	kit.E(err)
	return parent
}

// Previous returns the previous sibling element
func (el *Element) Previous() *Element {
	parent, err := el.PreviousE()
	kit.E(err)
	return parent
}

// ElementMatches returns the first element in the page that matches the CSS selector and its text matches the regex.
// The regex is the js regex, not golang's.
func (el *Element) ElementMatches(selector, regex string) *Element {
	el, err := el.ElementMatchesE(selector, regex)
	kit.E(err)
	return el
}

// Elements returns all elements that match the css selector
func (el *Element) Elements(selector string) Elements {
	list, err := el.ElementsE(selector)
	kit.E(err)
	return list
}

// ElementsX returns all elements that match the XPath selector
func (el *Element) ElementsX(xpath string) Elements {
	list, err := el.ElementsXE(xpath)
	kit.E(err)
	return list
}

// ElementsByJS returns the elements from the return value of the js
func (el *Element) ElementsByJS(js string, params ...interface{}) Elements {
	list, err := el.ElementsByJSE(js, params)
	kit.E(err)
	return list
}
