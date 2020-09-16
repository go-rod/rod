// This file contains the methods that panics when error return value is not nil.
// Their function names are all prefixed with Must.
// A function here is usually a wrapper for the error version with fixed default options to make it easier to use.
//
// For example the source code of `Element.Click` and `Element.MustClick`. `MustClick` has no argument.
// But `Click` has a `button` argument to decide which button to click.
// `MustClick` feels like a version of `Click` with some default behaviors.

package rod

import (
	"net/http"
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// MustConnect to the browser and start to control it.
// If fails to connect, try to run a local browser, if local browser not found try to download one.
func (b *Browser) MustConnect() *Browser {
	utils.E(b.Connect())
	return b
}

// MustClose the browser and release related resources
func (b *Browser) MustClose() {
	_ = b.Close()
}

// MustIncognito creates a new incognito browser
func (b *Browser) MustIncognito() *Browser {
	b, err := b.Incognito()
	utils.E(err)
	return b
}

// MustPage creates a new tab
// If url is empty, the default target will be "about:blank".
func (b *Browser) MustPage(url string) *Page {
	p, err := b.Page(url)
	utils.E(err)
	return p
}

// MustPages returns all visible pages
func (b *Browser) MustPages() Pages {
	list, err := b.Pages()
	utils.E(err)
	return list
}

// MustPageFromTargetID creates a Page instance from a targetID
func (b *Browser) MustPageFromTargetID(targetID proto.TargetTargetID) *Page {
	p, err := b.PageFromTarget(targetID)
	utils.E(err)
	return p
}

// MustHandleAuth for the next basic HTTP authentication.
// It will prevent the popup that requires user to input user name and password.
// Ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
func (b *Browser) MustHandleAuth(username, password string) {
	wait := b.HandleAuth(username, password)
	go func() { utils.E(wait()) }()
}

// MustFind the page that has the specified element with the css selector
func (ps Pages) MustFind(selector string) *Page {
	p, err := ps.Find(selector)
	utils.E(err)
	return p
}

// MustFindByURL returns the page that has the url that matches the regex
func (ps Pages) MustFindByURL(regex string) *Page {
	p, err := ps.FindByURL(regex)
	utils.E(err)
	return p
}

// MustInfo of the page, such as the URL or title of the page
func (p *Page) MustInfo() *proto.TargetTargetInfo {
	info, err := p.Info()
	utils.E(err)
	return info
}

// MustCookies returns the page cookies. By default it will return the cookies for current page.
// The urls is the list of URLs for which applicable cookies will be fetched.
func (p *Page) MustCookies(urls ...string) []*proto.NetworkCookie {
	cookies, err := p.Cookies(urls)
	utils.E(err)
	return cookies
}

// MustSetCookies of the page.
// Cookie format: https://chromedevtools.github.io/devtools-protocol/tot/Network#method-setCookie
func (p *Page) MustSetCookies(cookies ...*proto.NetworkCookieParam) *Page {
	utils.E(p.SetCookies(cookies))
	return p
}

// MustSetExtraHeaders whether to always send extra HTTP headers with the requests from this page.
// The arguments are key-value pairs, you can set multiple key-value pairs at the same time.
func (p *Page) MustSetExtraHeaders(dict ...string) (cleanup func()) {
	cleanup, err := p.SetExtraHeaders(dict)
	utils.E(err)
	return
}

// MustSetUserAgent Allows overriding user agent with the given string.
// If req is nil, the default user agent will be the same as a mac chrome.
func (p *Page) MustSetUserAgent(req *proto.NetworkSetUserAgentOverride) *Page {
	utils.E(p.SetUserAgent(req))
	return p
}

// MustNavigate to url
// If url is empty, it will navigate to "about:blank".
func (p *Page) MustNavigate(url string) *Page {
	utils.E(p.Navigate(url))
	return p
}

// MustGetWindow get window bounds
func (p *Page) MustGetWindow() *proto.BrowserBounds {
	bounds, err := p.GetWindow()
	utils.E(err)
	return bounds
}

// MustWindow set the window location and size
func (p *Page) MustWindow(left, top, width, height int64) *Page {
	utils.E(p.Window(&proto.BrowserBounds{
		Left:        left,
		Top:         top,
		Width:       width,
		Height:      height,
		WindowState: proto.BrowserWindowStateNormal,
	}))
	return p
}

// MustWindowMinimize the window
func (p *Page) MustWindowMinimize() *Page {
	utils.E(p.Window(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateMinimized,
	}))
	return p
}

// MustWindowMaximize the window
func (p *Page) MustWindowMaximize() *Page {
	utils.E(p.Window(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateMaximized,
	}))
	return p
}

// MustWindowFullscreen the window
func (p *Page) MustWindowFullscreen() *Page {
	utils.E(p.Window(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateFullscreen,
	}))
	return p
}

// MustWindowNormal the window size
func (p *Page) MustWindowNormal() *Page {
	utils.E(p.Window(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateNormal,
	}))
	return p
}

// MustViewport overrides the values of device screen dimensions.
func (p *Page) MustViewport(width, height int64, deviceScaleFactor float64, mobile bool) *Page {
	utils.E(p.Viewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: deviceScaleFactor,
		Mobile:            mobile,
	}))
	return p
}

// MustEmulate the device, such as iPhone9. If device is empty, it will clear the override.
func (p *Page) MustEmulate(device devices.DeviceType) *Page {
	utils.E(p.Emulate(device, false))
	return p
}

// MustStopLoading forces the page stop all navigations and pending resource fetches.
func (p *Page) MustStopLoading() *Page {
	utils.E(p.StopLoading())
	return p
}

// MustClose page
func (p *Page) MustClose() {
	utils.E(p.Close())
}

// MustHandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload)
// Because alert will block js, usually you have to run the wait function inside a goroutine. Check the unit test
// for it for more information.
func (p *Page) MustHandleDialog(accept bool, promptText string) (wait func()) {
	w := p.HandleDialog(accept, promptText)
	return func() {
		utils.E(w())
	}
}

// MustScreenshot the page and returns the binary of the image
// If the toFile is "", it will save output to "tmp/screenshots" folder, time as the file name.
func (p *Page) MustScreenshot(toFile ...string) []byte {
	bin, err := p.Screenshot(false, &proto.PageCaptureScreenshot{})
	utils.E(err)
	utils.E(saveFile(saveFileTypeScreenshot, bin, toFile))
	return bin
}

// MustScreenshotFullPage including all scrollable content and returns the binary of the image.
func (p *Page) MustScreenshotFullPage(toFile ...string) []byte {
	bin, err := p.Screenshot(true, &proto.PageCaptureScreenshot{})
	utils.E(err)
	utils.E(saveFile(saveFileTypeScreenshot, bin, toFile))
	return bin
}

// MustPDF prints page as MustPDF
func (p *Page) MustPDF(toFile ...string) []byte {
	bin, err := p.PDF(&proto.PagePrintToPDF{})
	utils.E(err)
	utils.E(saveFile(saveFileTypePDF, bin, toFile))
	return bin
}

// MustGetDownloadFile of the next download url that matches the pattern, returns the file content.
func (p *Page) MustGetDownloadFile(pattern string) func() []byte {
	wait := p.GetDownloadFile(pattern, "", http.DefaultClient)
	return func() []byte {
		_, body, err := wait()
		utils.E(err)
		return body
	}
}

// MustWaitOpen waits for a new page opened by the current one
func (p *Page) MustWaitOpen() (wait func() (newPage *Page)) {
	w := p.WaitOpen()
	return func() *Page {
		page, err := w()
		utils.E(err)
		return page
	}
}

// MustWaitPauseOpen waits for a page opened by the current page, before opening pause the js execution.
// Because the js will be paused, you should put the code that triggers it in a goroutine, such as the click.
func (p *Page) MustWaitPauseOpen() (wait func() *Page, resume func()) {
	newPage, r, err := p.WaitPauseOpen()
	utils.E(err)

	return func() *Page {
		page, err := newPage()
		utils.E(err)
		return page
	}, func() { utils.E(r()) }
}

// MustWaitRequestIdle returns a wait function that waits until the page doesn't send request for 300ms.
// You can pass regular expressions to exclude the requests by their url.
func (p *Page) MustWaitRequestIdle(excludes ...string) (wait func()) {
	return p.WaitRequestIdle(300*time.Millisecond, nil, excludes)
}

// MustWaitIdle wait until the next window.requestIdleCallback is called.
func (p *Page) MustWaitIdle() *Page {
	utils.E(p.WaitIdle(time.Minute))
	return p
}

// MustWaitLoad wait until the `window.onload` is complete, resolve immediately if already fired.
func (p *Page) MustWaitLoad() *Page {
	utils.E(p.WaitLoad())
	return p
}

// MustAddScriptTag to page. If url is empty, content will be used.
func (p *Page) MustAddScriptTag(url string) *Page {
	utils.E(p.AddScriptTag(url, ""))
	return p
}

// MustAddStyleTag to page. If url is empty, content will be used.
func (p *Page) MustAddStyleTag(url string) *Page {
	utils.E(p.AddStyleTag(url, ""))
	return p
}

// MustEvalOnNewDocument Evaluates given script in every frame upon creation (before loading frame's scripts).
func (p *Page) MustEvalOnNewDocument(js string) {
	_, err := p.EvalOnNewDocument(js)
	utils.E(err)
}

// MustExpose function to the page's window object. Must bind before navigate to the page. Bindings survive reloads.
// Binding function takes exactly one argument, this argument should be string.
func (p *Page) MustExpose(name string) (callback chan string, stop func()) {
	c, s, err := p.Expose(name)
	utils.E(err)
	return c, s
}

// MustEval js on the page. The first param must be a js function definition.
// For example page.MustEval(`n => n + 1`, 1) will return 2
func (p *Page) MustEval(js string, params ...interface{}) proto.JSON {
	res, err := p.Eval(js, params...)
	utils.E(err)
	return res.Value
}

// MustWait js function until it returns true
func (p *Page) MustWait(js string, params ...interface{}) {
	utils.E(p.Wait("", js, params))
}

// MustObjectToJSON by remote object
func (p *Page) MustObjectToJSON(obj *proto.RuntimeRemoteObject) proto.JSON {
	j, err := p.ObjectToJSON(obj)
	utils.E(err)
	return j
}

// MustObjectsToJSON by remote objects
func (p *Page) MustObjectsToJSON(list []*proto.RuntimeRemoteObject) proto.JSON {
	result := "[]"
	for _, obj := range list {
		j, err := p.ObjectToJSON(obj)
		utils.E(err)
		result, err = sjson.SetRaw(result, "-1", j.Raw)
		utils.E(err)
	}
	return proto.JSON{Result: gjson.Parse(result)}
}

// MustElementFromNode creates an Element from the node id
func (p *Page) MustElementFromNode(id proto.DOMNodeID) *Element {
	el, err := p.ElementFromNode(id)
	utils.E(err)
	return el
}

// MustElementFromPoint creates an Element from the absolute point on the page.
// The point should include the window scroll offset.
func (p *Page) MustElementFromPoint(left, top int) *Element {
	el, err := p.ElementFromPoint(int64(left), int64(top))
	utils.E(err)
	return el
}

// MustRelease remote object
func (p *Page) MustRelease(objectID proto.RuntimeRemoteObjectID) *Page {
	utils.E(p.Release(objectID))
	return p
}

// MustHas an element that matches the css selector
func (p *Page) MustHas(selector string) bool {
	has, _, err := p.Has(selector)
	utils.E(err)
	return has
}

// MustHasX an element that matches the XPath selector
func (p *Page) MustHasX(selector string) bool {
	has, _, err := p.HasX(selector)
	utils.E(err)
	return has
}

// MustHasMatches an element that matches the css selector and its text matches the regex.
func (p *Page) MustHasMatches(selector, regex string) bool {
	has, _, err := p.HasMatches(selector, regex)
	utils.E(err)
	return has
}

// MustSearch for each given query in the DOM tree until find one, before that it will keep retrying.
// The query can be plain text or css selector or xpath.
// It will search nested iframes and shadow doms too.
func (p *Page) MustSearch(queries ...string) *Element {
	list, err := p.Search(0, 1, queries...)
	utils.E(err)
	return list.First()
}

// MustElement retries until an element in the page that matches one of the CSS selectors
func (p *Page) MustElement(selectors ...string) *Element {
	el, err := p.Element(selectors...)
	utils.E(err)
	return el
}

// MustElementR retries until an element in the page that matches one of the pairs.
// Each pairs is a css selector and a regex. A sample call will look like page.MustElementR("div", "click me").
// The regex is the js regex, not golang's.
func (p *Page) MustElementR(pairs ...string) *Element {
	el, err := p.ElementR(pairs...)
	utils.E(err)
	return el
}

// MustElementByJS retries until returns the element from the return value of the js function
func (p *Page) MustElementByJS(js string, params ...interface{}) *Element {
	el, err := p.ElementByJS(NewEvalOptions(js, params))
	utils.E(err)
	return el
}

// MustElements returns all elements that match the css selector
func (p *Page) MustElements(selector string) Elements {
	list, err := p.Elements(selector)
	utils.E(err)
	return list
}

// MustElementsX returns all elements that match the XPath selector
func (p *Page) MustElementsX(xpath string) Elements {
	list, err := p.ElementsX(xpath)
	utils.E(err)
	return list
}

// MustElementX retries until an element in the page that matches one of the XPath selectors
func (p *Page) MustElementX(xPaths ...string) *Element {
	el, err := p.ElementX(xPaths...)
	utils.E(err)
	return el
}

// MustElementsByJS returns the elements from the return value of the js
func (p *Page) MustElementsByJS(js string, params ...interface{}) Elements {
	list, err := p.ElementsByJS(NewEvalOptions(js, params))
	utils.E(err)
	return list
}

// MustElement the doc is similar with MustElement but has a callback when a match is found
func (rc *RaceContext) MustElement(selector string, callback func(*Element)) *RaceContext {
	return rc.Element(selector, func(el *Element) error {
		callback(el)
		return nil
	})
}

// MustElementX the doc is similar with MustElement but has a callback when a match is found
func (rc *RaceContext) MustElementX(selector string, callback func(*Element)) *RaceContext {
	return rc.ElementX(selector, func(el *Element) error {
		callback(el)
		return nil
	})
}

// MustElementR the doc is similar with MustElement but has a callback when a match is found
func (rc *RaceContext) MustElementR(selector, regex string, callback func(*Element)) *RaceContext {
	return rc.ElementR(selector, regex, func(el *Element) error {
		callback(el)
		return nil
	})
}

// MustElementByJS the doc is similar with MustElementByJS but has a callback when a match is found
func (rc *RaceContext) MustElementByJS(js string, params JSArgs, callback func(*Element) error) *RaceContext {
	return rc.ElementByJS(NewEvalOptions(js, params), callback)
}

// MustDo the race
func (rc *RaceContext) MustDo() *Page {
	utils.E(rc.Do())
	return rc.page
}

// MustMove to the absolute position
func (m *Mouse) MustMove(x, y float64) *Mouse {
	utils.E(m.Move(x, y, 0))
	return m
}

// MustScroll with the relative offset
func (m *Mouse) MustScroll(x, y float64) *Mouse {
	utils.E(m.Scroll(x, y, 0))
	return m
}

// MustDown holds the button down
func (m *Mouse) MustDown(button proto.InputMouseButton) *Mouse {
	utils.E(m.Down(button, 1))
	return m
}

// MustUp release the button
func (m *Mouse) MustUp(button proto.InputMouseButton) *Mouse {
	utils.E(m.Up(button, 1))
	return m
}

// MustClick will press then release the button
func (m *Mouse) MustClick(button proto.InputMouseButton) *Mouse {
	utils.E(m.Click(button))
	return m
}

// MustDown holds key down
func (k *Keyboard) MustDown(key rune) *Keyboard {
	utils.E(k.Down(key))
	return k
}

// MustUp releases the key
func (k *Keyboard) MustUp(key rune) *Keyboard {
	utils.E(k.Up(key))
	return k
}

// MustPress a key
func (k *Keyboard) MustPress(key rune) *Keyboard {
	utils.E(k.Press(key))
	return k
}

// MustInsertText like paste text into the page
func (k *Keyboard) MustInsertText(text string) *Keyboard {
	utils.E(k.InsertText(text))
	return k
}

// MustDescribe returns the element info
// Returned json: https://chromedevtools.github.io/devtools-protocol/tot/DOM#type-Node
func (el *Element) MustDescribe() *proto.DOMNode {
	node, err := el.Describe(1, false)
	utils.E(err)
	return node
}

// MustNodeID of the node
func (el *Element) MustNodeID() proto.DOMNodeID {
	id, err := el.NodeID()
	utils.E(err)
	return id
}

// MustShadowRoot returns the shadow root of this element
func (el *Element) MustShadowRoot() *Element {
	node, err := el.ShadowRoot()
	utils.E(err)
	return node
}

// MustFrame creates a page instance that represents the iframe
func (el *Element) MustFrame() *Page {
	p, err := el.Frame()
	utils.E(err)
	return p
}

// MustFocus sets focus on the specified element
func (el *Element) MustFocus() *Element {
	utils.E(el.Focus())
	return el
}

// MustScrollIntoView scrolls the current element into the visible area of the browser
// window if it's not already within the visible area.
func (el *Element) MustScrollIntoView() *Element {
	utils.E(el.ScrollIntoView())
	return el
}

// MustHover the mouse over the center of the element.
func (el *Element) MustHover() *Element {
	utils.E(el.Hover())
	return el
}

// MustClick the element
func (el *Element) MustClick() *Element {
	utils.E(el.Click(proto.InputMouseButtonLeft))
	return el
}

// MustClickable checks if the element is behind another element, such as when covered by a modal.
func (el *Element) MustClickable() bool {
	clickable, err := el.Clickable()
	utils.E(err)
	return clickable
}

// MustPress a key
func (el *Element) MustPress(key rune) *Element {
	utils.E(el.Press(key))
	return el
}

// MustSelectText selects the text that matches the regular expression
func (el *Element) MustSelectText(regex string) *Element {
	utils.E(el.SelectText(regex))
	return el
}

// MustSelectAllText selects all text
func (el *Element) MustSelectAllText() *Element {
	utils.E(el.SelectAllText())
	return el
}

// MustInput wll click the element and input the text.
// To empty the input you can use something like el.SelectAllText().MustInput("")
func (el *Element) MustInput(text string) *Element {
	utils.E(el.Input(text))
	return el
}

// MustBlur will call the blur function on the element.
// On inputs, this will deselect the element.
func (el *Element) MustBlur() *Element {
	utils.E(el.Blur())
	return el
}

// MustSelect the option elements that match the selectors, the selector can be text content or css selector
func (el *Element) MustSelect(selectors ...string) *Element {
	utils.E(el.Select(selectors))
	return el
}

// MustMatches checks if the element can be selected by the css selector
func (el *Element) MustMatches(selector string) bool {
	res, err := el.Matches(selector)
	utils.E(err)
	return res
}

// MustAttribute returns the value of a specified attribute on the element.
// Please check the Property function before you use it, usually you don't want to use MustAttribute.
// https://stackoverflow.com/questions/6003819/what-is-the-difference-between-properties-and-attributes-in-html
func (el *Element) MustAttribute(name string) *string {
	attr, err := el.Attribute(name)
	utils.E(err)
	return attr
}

// MustProperty returns the value of a specified property on the element.
// It's similar to Attribute but attributes can only be string, properties can be types like bool, float, etc.
// https://stackoverflow.com/questions/6003819/what-is-the-difference-between-properties-and-attributes-in-html
func (el *Element) MustProperty(name string) proto.JSON {
	prop, err := el.Property(name)
	utils.E(err)
	return prop
}

// MustContainsElement check if the target is equal or inside the element.
func (el *Element) MustContainsElement(target *Element) bool {
	contains, err := el.ContainsElement(target)
	utils.E(err)
	return contains
}

// MustSetFiles sets files for the given file input element
func (el *Element) MustSetFiles(paths ...string) *Element {
	utils.E(el.SetFiles(paths))
	return el
}

// MustText gets the innerText of the element
func (el *Element) MustText() string {
	s, err := el.Text()
	utils.E(err)
	return s
}

// MustHTML gets the outerHTML of the element
func (el *Element) MustHTML() string {
	s, err := el.HTML()
	utils.E(err)
	return s
}

// MustVisible returns true if the element is visible on the page
func (el *Element) MustVisible() bool {
	v, err := el.Visible()
	utils.E(err)
	return v
}

// MustWaitLoad for element like <img />
func (el *Element) MustWaitLoad() *Element {
	utils.E(el.WaitLoad())
	return el
}

// MustWaitStable waits until the size and position are stable. Useful when waiting for the animation of modal
// or button to complete so that we can simulate the mouse to move to it and click on it.
func (el *Element) MustWaitStable() *Element {
	utils.E(el.WaitStable(100 * time.Millisecond))
	return el
}

// MustWait until the js returns true
func (el *Element) MustWait(js string, params ...interface{}) *Element {
	utils.E(el.Wait(js, params))
	return el
}

// MustWaitVisible until the element is visible
func (el *Element) MustWaitVisible() *Element {
	utils.E(el.WaitVisible())
	return el
}

// MustWaitInvisible until the element is not visible or removed
func (el *Element) MustWaitInvisible() *Element {
	utils.E(el.WaitInvisible())
	return el
}

// MustBox returns the size of an element and its position relative to the main frame.
func (el *Element) MustBox() *proto.DOMRect {
	box, err := el.Box()
	utils.E(err)
	return box
}

// MustCanvasToImage get image data of a canvas.
// The default format is image/png.
// The default quality is 0.92.
// doc: https://developer.mozilla.org/en-US/docs/Web/API/HTMLCanvasElement/toDataURL
func (el *Element) MustCanvasToImage(format string, quality float64) []byte {
	bin, err := el.CanvasToImage(format, quality)
	utils.E(err)
	return bin
}

// MustResource returns the binary of the "src" properly, such as the image or audio file.
func (el *Element) MustResource() []byte {
	bin, err := el.Resource()
	utils.E(err)
	return bin
}

// MustScreenshot of the area of the element
func (el *Element) MustScreenshot(toFile ...string) []byte {
	bin, err := el.Screenshot(proto.PageCaptureScreenshotFormatPng, 0)
	utils.E(err)
	utils.E(saveFile(saveFileTypeScreenshot, bin, toFile))
	return bin
}

// MustRelease remote object on browser
func (el *Element) MustRelease() {
	utils.E(el.Release())
}

// MustEval evaluates js function on the element, the first param must be a js function definition
// For example: el.MustEval(`name => this.getAttribute(name)`, "value")
func (el *Element) MustEval(js string, params ...interface{}) proto.JSON {
	res, err := el.Eval(js, params...)
	utils.E(err)
	return res.Value
}

// MustHas an element that matches the css selector
func (el *Element) MustHas(selector string) bool {
	has, _, err := el.Has(selector)
	utils.E(err)
	return has
}

// MustHasX an element that matches the XPath selector
func (el *Element) MustHasX(selector string) bool {
	has, _, err := el.HasX(selector)
	utils.E(err)
	return has
}

// MustHasMatches an element that matches the css selector and its text matches the regex.
func (el *Element) MustHasMatches(selector, regex string) bool {
	has, _, err := el.HasMatches(selector, regex)
	utils.E(err)
	return has
}

// MustElement returns the first child that matches the css selector
func (el *Element) MustElement(selector string) *Element {
	el, err := el.Element(selector)
	utils.E(err)
	return el
}

// MustElementX returns the first child that matches the XPath selector
func (el *Element) MustElementX(xpath string) *Element {
	el, err := el.ElementX(xpath)
	utils.E(err)
	return el
}

// MustElementByJS returns the element from the return value of the js
func (el *Element) MustElementByJS(js string, params ...interface{}) *Element {
	el, err := el.ElementByJS(NewEvalOptions(js, params))
	utils.E(err)
	return el
}

// MustParent returns the parent element
func (el *Element) MustParent() *Element {
	parent, err := el.Parent()
	utils.E(err)
	return parent
}

// MustParents that match the selector
func (el *Element) MustParents(selector string) Elements {
	list, err := el.Parents(selector)
	utils.E(err)
	return list
}

// MustNext returns the next sibling element
func (el *Element) MustNext() *Element {
	parent, err := el.Next()
	utils.E(err)
	return parent
}

// MustPrevious returns the previous sibling element
func (el *Element) MustPrevious() *Element {
	parent, err := el.Previous()
	utils.E(err)
	return parent
}

// MustElementR returns the first element in the page that matches the CSS selector and its text matches the regex.
// The regex is the js regex, not golang's.
func (el *Element) MustElementR(selector, regex string) *Element {
	el, err := el.ElementR(selector, regex)
	utils.E(err)
	return el
}

// MustElements returns all elements that match the css selector
func (el *Element) MustElements(selector string) Elements {
	list, err := el.Elements(selector)
	utils.E(err)
	return list
}

// MustElementsX returns all elements that match the XPath selector
func (el *Element) MustElementsX(xpath string) Elements {
	list, err := el.ElementsX(xpath)
	utils.E(err)
	return list
}

// MustElementsByJS returns the elements from the return value of the js
func (el *Element) MustElementsByJS(js string, params ...interface{}) Elements {
	list, err := el.ElementsByJS(NewEvalOptions(js, params))
	utils.E(err)
	return list
}

// MustAdd a hijack handler to router, the doc of the pattern is the same as "proto.FetchRequestPattern.URLPattern".
// You can add new handler even after the "Run" is called.
func (r *HijackRouter) MustAdd(pattern string, handler func(*Hijack)) *HijackRouter {
	utils.E(r.Add(pattern, "", handler))
	return r
}

// MustRemove handler via the pattern
func (r *HijackRouter) MustRemove(pattern string) *HijackRouter {
	utils.E(r.Remove(pattern))
	return r
}

// MustStop the router
func (r *HijackRouter) MustStop() {
	utils.E(r.Stop())
}

// MustLoadResponse will send request to the real destination and load the response as default response to override.
func (h *Hijack) MustLoadResponse() {
	utils.E(h.LoadResponse(http.DefaultClient, true))
}
