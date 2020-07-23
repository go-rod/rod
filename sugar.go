// This file contains the methods that panics when error return value is not nil.

package rod

import (
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/ysmood/kit"
)

// Connect to the browser and start to control it.
// If fails to connect, try to run a local browser, if local browser not found try to download one.
func (b *Browser) Connect() *Browser {
	kit.E(b.ConnectE())
	return b
}

// Close the browser and release related resources
func (b *Browser) Close() {
	_ = b.CloseE()
}

// Incognito creates a new incognito browser
func (b *Browser) Incognito() *Browser {
	b, err := b.IncognitoE()
	kit.E(err)
	return b
}

// Page creates a new tab
// If url is empty, the default target will be "about:blank".
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

// HandleAuth for the next basic HTTP authentication.
// It will prevent the popup that requires user to input user name and password.
// Ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
func (b *Browser) HandleAuth(username, password string) {
	wait := b.HandleAuthE(username, password)
	go func() { kit.E(wait()) }()
}

// FindByURL returns the page that has the url that matches the regex
func (ps Pages) FindByURL(regex string) *Page {
	p, err := ps.FindByURLE(regex)
	kit.E(err)
	return p
}

// Info of the page, such as the URL or title of the page
func (p *Page) Info() *proto.TargetTargetInfo {
	info, err := p.InfoE()
	kit.E(err)
	return info
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
func (p *Page) SetExtraHeaders(dict ...string) (cleanup func()) {
	cleanup, err := p.SetExtraHeadersE(dict)
	kit.E(err)
	return
}

// SetUserAgent Allows overriding user agent with the given string.
// If req is nil, the default user agent will be the same as a mac chrome.
func (p *Page) SetUserAgent(req *proto.NetworkSetUserAgentOverride) *Page {
	kit.E(p.SetUserAgentE(req))
	return p
}

// Navigate to url
// If url is empty, it will navigate to "about:blank".
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

// Emulate the device, such as iPhone9. If device is empty, it will clear the override.
func (p *Page) Emulate(device devices.DeviceType) *Page {
	kit.E(p.EmulateE(device, false))
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

// HandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload)
func (p *Page) HandleDialog(accept bool, promptText string) (wait func()) {
	w := p.HandleDialogE(accept, promptText)
	return func() {
		kit.E(w())
	}
}

// Screenshot the page and returns the binary of the image
// If the toFile is "", it will save output to "tmp/screenshots" folder, time as the file name.
func (p *Page) Screenshot(toFile ...string) []byte {
	bin, err := p.ScreenshotE(false, &proto.PageCaptureScreenshot{})
	kit.E(err)
	kit.E(saveScreenshot(bin, toFile))
	return bin
}

// ScreenshotFullPage including all scrollable content and returns the binary of the image.
func (p *Page) ScreenshotFullPage(toFile ...string) []byte {
	bin, err := p.ScreenshotE(true, &proto.PageCaptureScreenshot{})
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

// WaitOpen to be created from a new window
func (p *Page) WaitOpen() (wait func() *Page) {
	w := p.WaitOpenE()
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
	return p.WaitRequestIdleE(300*time.Millisecond, []string{""}, excludes)
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

// EvalOnNewDocument Evaluates given script in every frame upon creation (before loading frame's scripts).
func (p *Page) EvalOnNewDocument(js string) {
	_, err := p.EvalOnNewDocumentE(js)
	kit.E(err)
}

// Eval js on the page. The first param must be a js function definition.
// For example page.Eval(`n => n + 1`, 1) will return 2
func (p *Page) Eval(js string, params ...interface{}) proto.JSON {
	res, err := p.EvalE(true, "", js, params)
	kit.E(err)
	return res.Value
}

// Wait js function until it returns true
func (p *Page) Wait(js string, params ...interface{}) {
	kit.E(p.WaitE(Sleeper(), "", js, params))
}

// ObjectToJSON by remote object
func (p *Page) ObjectToJSON(obj *proto.RuntimeRemoteObject) proto.JSON {
	j, err := p.ObjectToJSONE(obj)
	kit.E(err)
	return j
}

// ObjectsToJSON by remote objects
func (p *Page) ObjectsToJSON(list []*proto.RuntimeRemoteObject) proto.JSON {
	result := "[]"
	for _, obj := range list {
		j, err := p.ObjectToJSONE(obj)
		kit.E(err)
		result, err = sjson.SetRaw(result, "-1", j.Raw)
		kit.E(err)
	}
	return proto.JSON{Result: gjson.Parse(result)}
}

// ElementFromNode creates an Element from the node id
func (p *Page) ElementFromNode(id proto.DOMNodeID) *Element {
	el, err := p.ElementFromNodeE(id)
	kit.E(err)
	return el
}

// ElementFromPoint creates an Element from the absolute point on the page.
// The point should include the window scroll offset.
func (p *Page) ElementFromPoint(left, top int) *Element {
	el, err := p.ElementFromPointE(int64(left), int64(top))
	kit.E(err)
	return el
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

// Search for a given query in the DOM tree until the result count is not zero.
// The query can be plain text or css selector or xpath.
// It will search nested iframes and shadow doms too.
func (p *Page) Search(query string) *Element {
	list, err := p.SearchE(Sleeper(), query, 0, 1)
	kit.E(err)
	return list.First()
}

// Element retries until an element in the page that matches one of the CSS selectors
func (p *Page) Element(selectors ...string) *Element {
	el, err := p.ElementE(Sleeper(), "", selectors)
	kit.E(err)
	return el
}

// ElementMatches retries until an element in the page that matches one of the pairs.
// Each pairs is a css selector and a regex. A sample call will look like page.ElementMatches("div", "click me").
// The regex is the js regex, not golang's.
func (p *Page) ElementMatches(pairs ...string) *Element {
	el, err := p.ElementMatchesE(Sleeper(), "", pairs)
	kit.E(err)
	return el
}

// ElementByJS retries until returns the element from the return value of the js function
func (p *Page) ElementByJS(js string, params ...interface{}) *Element {
	el, err := p.ElementByJSE(Sleeper(), "", js, params)
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

// ElementX retries until an element in the page that matches one of the XPath selectors
func (p *Page) ElementX(xPaths ...string) *Element {
	el, err := p.ElementXE(Sleeper(), "", xPaths)
	kit.E(err)
	return el
}

// ElementsByJS returns the elements from the return value of the js
func (p *Page) ElementsByJS(js string, params ...interface{}) Elements {
	list, err := p.ElementsByJSE("", js, params)
	kit.E(err)
	return list
}

// Move to the absolute position
func (m *Mouse) Move(x, y float64) *Mouse {
	kit.E(m.MoveE(x, y, 0))
	return m
}

// Scroll with the relative offset
func (m *Mouse) Scroll(x, y float64) *Mouse {
	kit.E(m.ScrollE(x, y, 0))
	return m
}

// Down holds the button down
func (m *Mouse) Down(button proto.InputMouseButton) *Mouse {
	kit.E(m.DownE(button, 1))
	return m
}

// Up release the button
func (m *Mouse) Up(button proto.InputMouseButton) *Mouse {
	kit.E(m.UpE(button, 1))
	return m
}

// Click will press then release the button
func (m *Mouse) Click(button proto.InputMouseButton) *Mouse {
	kit.E(m.ClickE(button))
	return m
}

// Down holds key down
func (k *Keyboard) Down(key rune) *Keyboard {
	kit.E(k.DownE(key))
	return k
}

// Up releases the key
func (k *Keyboard) Up(key rune) *Keyboard {
	kit.E(k.UpE(key))
	return k
}

// Press a key
func (k *Keyboard) Press(key rune) *Keyboard {
	kit.E(k.PressE(key))
	return k
}

// InsertText like paste text into the page
func (k *Keyboard) InsertText(text string) *Keyboard {
	kit.E(k.InsertTextE(text))
	return k
}

// Describe returns the element info
// Returned json: https://chromedevtools.github.io/devtools-protocol/tot/DOM#type-Node
func (el *Element) Describe() *proto.DOMNode {
	node, err := el.DescribeE(1, false)
	kit.E(err)
	return node
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

// Clickable checks if the element is behind another element, such as when covered by a modal.
func (el *Element) Clickable() bool {
	clickable, err := el.ClickableE()
	kit.E(err)
	return clickable
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

// Blur will call the blur function on the element.
// On inputs, this will deselect the element.
func (el *Element) Blur() *Element {
	kit.E(el.BlurE())
	return el
}

// Select the option elements that match the selectors, the selector can be text content or css selector
func (el *Element) Select(selectors ...string) *Element {
	kit.E(el.SelectE(selectors))
	return el
}

// Matches checks if the element can be selected by the css selector
func (el *Element) Matches(selector string) bool {
	res, err := el.MatchesE(selector)
	kit.E(err)
	return res
}

// Attribute returns the value of a specified attribute on the element.
// Please check the Property function before you use it, usually you don't want to use Attribute.
// https://stackoverflow.com/questions/6003819/what-is-the-difference-between-properties-and-attributes-in-html
func (el *Element) Attribute(name string) *string {
	attr, err := el.AttributeE(name)
	kit.E(err)
	return attr
}

// Property returns the value of a specified property on the element.
// It's similar to Attribute but attributes can only be string, properties can be types like bool, float, etc.
// https://stackoverflow.com/questions/6003819/what-is-the-difference-between-properties-and-attributes-in-html
func (el *Element) Property(name string) proto.JSON {
	prop, err := el.PropertyE(name)
	kit.E(err)
	return prop
}

// ContainsElement check if the target is equal or inside the element.
func (el *Element) ContainsElement(target *Element) bool {
	contains, err := el.ContainsElementE(target)
	kit.E(err)
	return contains
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
func (el *Element) Box() *proto.DOMRect {
	box, err := el.BoxE()
	kit.E(err)
	return box
}

// CanvasToImage get image data of a canvas.
// The default format is image/png.
// The default quality is 0.92.
// doc: https://developer.mozilla.org/en-US/docs/Web/API/HTMLCanvasElement/toDataURL
func (el *Element) CanvasToImage(format string, quality float64) []byte {
	bin, err := el.CanvasToImageE(format, quality)
	kit.E(err)
	return bin
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
