// This file contains the methods that panics when error return value is not nil.
// Their function names are all prefixed with Must.
// A function here is usually a wrapper for the error version with fixed default options to make it easier to use.
//
// For example the source code of `Element.Click` and `Element.MustClick`. `MustClick` has no argument.
// But `Click` has a `button` argument to decide which button to click.
// `MustClick` feels like a version of `Click` with some default behaviors.

package rod

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// MustConnect is similar to Connect
func (b *Browser) MustConnect() *Browser {
	utils.E(b.Connect())
	return b
}

// MustClose is similar to Close
func (b *Browser) MustClose() {
	_ = b.Close()
}

// MustIncognito is similar to Incognito
func (b *Browser) MustIncognito() *Browser {
	b, err := b.Incognito()
	utils.E(err)
	return b
}

// MustPage is similar to Page
func (b *Browser) MustPage(url string) *Page {
	p, err := b.Page(url)
	utils.E(err)
	return p
}

// MustPages is similar to Pages
func (b *Browser) MustPages() Pages {
	list, err := b.Pages()
	utils.E(err)
	return list
}

// MustPageFromTargetID is similar to PageFromTargetID
func (b *Browser) MustPageFromTargetID(targetID proto.TargetTargetID) *Page {
	p, err := b.PageFromTarget(targetID)
	utils.E(err)
	return p
}

// MustHandleAuth is similar to HandleAuth
func (b *Browser) MustHandleAuth(username, password string) {
	wait := b.HandleAuth(username, password)
	go func() { utils.E(wait()) }()
}

// MustIgnoreCertErrors is similar to IgnoreCertErrors
func (b *Browser) MustIgnoreCertErrors(enable bool) *Browser {
	utils.E(b.IgnoreCertErrors(enable))
	return b
}

// MustFind is similar to Find
func (ps Pages) MustFind(selector string) *Page {
	p, err := ps.Find(selector)
	utils.E(err)
	return p
}

// MustFindByURL is similar to FindByURL
func (ps Pages) MustFindByURL(regex string) *Page {
	p, err := ps.FindByURL(regex)
	utils.E(err)
	return p
}

// MustInfo is similar to Info
func (p *Page) MustInfo() *proto.TargetTargetInfo {
	info, err := p.Info()
	utils.E(err)
	return info
}

// MustCookies is similar to Cookies
func (p *Page) MustCookies(urls ...string) []*proto.NetworkCookie {
	cookies, err := p.Cookies(urls)
	utils.E(err)
	return cookies
}

// MustSetCookies is similar to SetCookies
func (p *Page) MustSetCookies(cookies ...*proto.NetworkCookieParam) *Page {
	utils.E(p.SetCookies(cookies))
	return p
}

// MustSetExtraHeaders is similar to SetExtraHeaders
func (p *Page) MustSetExtraHeaders(dict ...string) (cleanup func()) {
	cleanup, err := p.SetExtraHeaders(dict)
	utils.E(err)
	return
}

// MustSetUserAgent is similar to SetUserAgent
func (p *Page) MustSetUserAgent(req *proto.NetworkSetUserAgentOverride) *Page {
	utils.E(p.SetUserAgent(req))
	return p
}

// MustNavigate is similar to Navigate
func (p *Page) MustNavigate(url string) *Page {
	utils.E(p.Navigate(url))
	return p
}

// MustReload is similar to Reload
func (p *Page) MustReload() *Page {
	utils.E(p.Reload())
	return p
}

// MustNavigateBack is similar to NavigateBack
func (p *Page) MustNavigateBack() *Page {
	utils.E(p.NavigateBack())
	return p
}

// MustNavigateForward is similar to NavigateForward
func (p *Page) MustNavigateForward() *Page {
	utils.E(p.NavigateForward())
	return p
}

// MustGetWindow is similar to GetWindow
func (p *Page) MustGetWindow() *proto.BrowserBounds {
	bounds, err := p.GetWindow()
	utils.E(err)
	return bounds
}

// MustSetWindow is similar to SetWindow
func (p *Page) MustSetWindow(left, top, width, height int64) *Page {
	utils.E(p.SetWindow(&proto.BrowserBounds{
		Left:        left,
		Top:         top,
		Width:       width,
		Height:      height,
		WindowState: proto.BrowserWindowStateNormal,
	}))
	return p
}

// MustWindowMinimize is similar to WindowMinimize
func (p *Page) MustWindowMinimize() *Page {
	utils.E(p.SetWindow(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateMinimized,
	}))
	return p
}

// MustWindowMaximize is similar to WindowMaximize
func (p *Page) MustWindowMaximize() *Page {
	utils.E(p.SetWindow(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateMaximized,
	}))
	return p
}

// MustWindowFullscreen is similar to WindowFullscreen
func (p *Page) MustWindowFullscreen() *Page {
	utils.E(p.SetWindow(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateFullscreen,
	}))
	return p
}

// MustWindowNormal is similar to WindowNormal
func (p *Page) MustWindowNormal() *Page {
	utils.E(p.SetWindow(&proto.BrowserBounds{
		WindowState: proto.BrowserWindowStateNormal,
	}))
	return p
}

// MustSetViewport is similar to SetViewport
func (p *Page) MustSetViewport(width, height int64, deviceScaleFactor float64, mobile bool) *Page {
	utils.E(p.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: deviceScaleFactor,
		Mobile:            mobile,
	}))
	return p
}

// MustEmulate is similar to Emulate
func (p *Page) MustEmulate(device devices.Device) *Page {
	utils.E(p.Emulate(device, false))
	return p
}

// MustStopLoading is similar to StopLoading
func (p *Page) MustStopLoading() *Page {
	utils.E(p.StopLoading())
	return p
}

// MustClose is similar to Close
func (p *Page) MustClose() {
	utils.E(p.Close())
}

// MustHandleDialog is similar to HandleDialog
func (p *Page) MustHandleDialog(accept bool, promptText string) (wait func()) {
	w := p.HandleDialog(accept, promptText)
	return func() {
		utils.E(w())
	}
}

// MustScreenshot is similar to Screenshot
func (p *Page) MustScreenshot(toFile ...string) []byte {
	bin, err := p.Screenshot(false, &proto.PageCaptureScreenshot{})
	utils.E(err)
	utils.E(saveFile(saveFileTypeScreenshot, bin, toFile))
	return bin
}

// MustScreenshotFullPage is similar to ScreenshotFullPage
func (p *Page) MustScreenshotFullPage(toFile ...string) []byte {
	bin, err := p.Screenshot(true, &proto.PageCaptureScreenshot{})
	utils.E(err)
	utils.E(saveFile(saveFileTypeScreenshot, bin, toFile))
	return bin
}

// MustPDF is similar to PDF
func (p *Page) MustPDF(toFile ...string) []byte {
	r, err := p.PDF(&proto.PagePrintToPDF{})
	utils.E(err)
	bin, err := ioutil.ReadAll(r)
	utils.E(err)

	utils.E(saveFile(saveFileTypePDF, bin, toFile))
	return bin
}

// MustGetDownloadFile is similar to GetDownloadFile
func (p *Page) MustGetDownloadFile(pattern string) func() []byte {
	wait := p.GetDownloadFile(pattern, "", http.DefaultClient)
	return func() []byte {
		_, body, err := wait()
		utils.E(err)
		return body
	}
}

// MustWaitOpen is similar to WaitOpen
func (p *Page) MustWaitOpen() (wait func() (newPage *Page)) {
	w := p.WaitOpen()
	return func() *Page {
		page, err := w()
		utils.E(err)
		return page
	}
}

// MustWaitPauseOpen is similar to WaitPauseOpen
func (p *Page) MustWaitPauseOpen() (wait func() *Page, resume func()) {
	newPage, r, err := p.WaitPauseOpen()
	utils.E(err)

	return func() *Page {
		page, err := newPage()
		utils.E(err)
		return page
	}, func() { utils.E(r()) }
}

// MustWaitNavigation is similar to WaitNavigation
func (p *Page) MustWaitNavigation() func() {
	return p.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)
}

// MustWaitRequestIdle is similar to WaitRequestIdle
func (p *Page) MustWaitRequestIdle(excludes ...string) (wait func()) {
	return p.WaitRequestIdle(300*time.Millisecond, nil, excludes)
}

// MustWaitIdle is similar to WaitIdle
func (p *Page) MustWaitIdle() *Page {
	utils.E(p.WaitIdle(time.Minute))
	return p
}

// MustWaitLoad is similar to WaitLoad
func (p *Page) MustWaitLoad() *Page {
	utils.E(p.WaitLoad())
	return p
}

// MustAddScriptTag is similar to AddScriptTag
func (p *Page) MustAddScriptTag(url string) *Page {
	utils.E(p.AddScriptTag(url, ""))
	return p
}

// MustAddStyleTag is similar to AddStyleTag
func (p *Page) MustAddStyleTag(url string) *Page {
	utils.E(p.AddStyleTag(url, ""))
	return p
}

// MustEvalOnNewDocument is similar to EvalOnNewDocument
func (p *Page) MustEvalOnNewDocument(js string) {
	_, err := p.EvalOnNewDocument(js)
	utils.E(err)
}

// MustExpose is similar to Expose
func (p *Page) MustExpose(name string) (callback chan string, stop func()) {
	c, s, err := p.Expose(name)
	utils.E(err)
	return c, s
}

// MustEval is similar to Eval
func (p *Page) MustEval(js string, params ...interface{}) proto.JSON {
	res, err := p.Eval(js, params...)
	utils.E(err)
	return res.Value
}

// MustWait is similar to Wait
func (p *Page) MustWait(js string, params ...interface{}) {
	utils.E(p.Wait("", js, params))
}

// MustObjectToJSON is similar to ObjectToJSON
func (p *Page) MustObjectToJSON(obj *proto.RuntimeRemoteObject) proto.JSON {
	j, err := p.ObjectToJSON(obj)
	utils.E(err)
	return j
}

// MustObjectsToJSON is similar to ObjectsToJSON
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

// MustElementFromNode is similar to ElementFromNode
func (p *Page) MustElementFromNode(id proto.DOMNodeID) *Element {
	el, err := p.ElementFromNode(id)
	utils.E(err)
	return el
}

// MustElementFromPoint is similar to ElementFromPoint
func (p *Page) MustElementFromPoint(left, top int) *Element {
	el, err := p.ElementFromPoint(int64(left), int64(top))
	utils.E(err)
	return el
}

// MustRelease is similar to Release
func (p *Page) MustRelease(objectID proto.RuntimeRemoteObjectID) *Page {
	utils.E(p.Release(objectID))
	return p
}

// MustHas is similar to Has
func (p *Page) MustHas(selector string) bool {
	has, _, err := p.Has(selector)
	utils.E(err)
	return has
}

// MustHasX is similar to HasX
func (p *Page) MustHasX(selector string) bool {
	has, _, err := p.HasX(selector)
	utils.E(err)
	return has
}

// MustHasR is similar to HasR
func (p *Page) MustHasR(selector, regex string) bool {
	has, _, err := p.HasR(selector, regex)
	utils.E(err)
	return has
}

// MustSearch is similar to Search
func (p *Page) MustSearch(queries ...string) *Element {
	list, err := p.Search(0, 1, queries...)
	utils.E(err)
	return list.First()
}

// MustElement is similar to Element
func (p *Page) MustElement(selectors ...string) *Element {
	el, err := p.Element(selectors...)
	utils.E(err)
	return el
}

// MustElementR is similar to ElementR
func (p *Page) MustElementR(pairs ...string) *Element {
	el, err := p.ElementR(pairs...)
	utils.E(err)
	return el
}

// MustElementX is similar to ElementX
func (p *Page) MustElementX(xPaths ...string) *Element {
	el, err := p.ElementX(xPaths...)
	utils.E(err)
	return el
}

// MustElementByJS is similar to ElementByJS
func (p *Page) MustElementByJS(js string, params ...interface{}) *Element {
	el, err := p.ElementByJS(NewEvalOptions(js, params))
	utils.E(err)
	return el
}

// MustElements is similar to Elements
func (p *Page) MustElements(selector string) Elements {
	list, err := p.Elements(selector)
	utils.E(err)
	return list
}

// MustElementsX is similar to ElementsX
func (p *Page) MustElementsX(xpath string) Elements {
	list, err := p.ElementsX(xpath)
	utils.E(err)
	return list
}

// MustElementsByJS is similar to ElementsByJS
func (p *Page) MustElementsByJS(js string, params ...interface{}) Elements {
	list, err := p.ElementsByJS(NewEvalOptions(js, params))
	utils.E(err)
	return list
}

// MustElement is similar to Element
func (rc *RaceContext) MustElement(selector string, callback func(*Element)) *RaceContext {
	return rc.Element(selector, func(el *Element) error {
		callback(el)
		return nil
	})
}

// MustElementX is similar to ElementX
func (rc *RaceContext) MustElementX(selector string, callback func(*Element)) *RaceContext {
	return rc.ElementX(selector, func(el *Element) error {
		callback(el)
		return nil
	})
}

// MustElementR is similar to ElementR
func (rc *RaceContext) MustElementR(selector, regex string, callback func(*Element)) *RaceContext {
	return rc.ElementR(selector, regex, func(el *Element) error {
		callback(el)
		return nil
	})
}

// MustElementByJS is similar to ElementByJS
func (rc *RaceContext) MustElementByJS(js string, params JSArgs, callback func(*Element) error) *RaceContext {
	return rc.ElementByJS(NewEvalOptions(js, params), callback)
}

// MustDo is similar to Do
func (rc *RaceContext) MustDo() *Page {
	utils.E(rc.Do())
	return rc.page
}

// MustMove is similar to Move
func (m *Mouse) MustMove(x, y float64) *Mouse {
	utils.E(m.Move(x, y, 0))
	return m
}

// MustScroll is similar to Scroll
func (m *Mouse) MustScroll(x, y float64) *Mouse {
	utils.E(m.Scroll(x, y, 0))
	return m
}

// MustDown is similar to Down
func (m *Mouse) MustDown(button proto.InputMouseButton) *Mouse {
	utils.E(m.Down(button, 1))
	return m
}

// MustUp is similar to Up
func (m *Mouse) MustUp(button proto.InputMouseButton) *Mouse {
	utils.E(m.Up(button, 1))
	return m
}

// MustClick is similar to Click
func (m *Mouse) MustClick(button proto.InputMouseButton) *Mouse {
	utils.E(m.Click(button))
	return m
}

// MustDown is similar to Down
func (k *Keyboard) MustDown(key rune) *Keyboard {
	utils.E(k.Down(key))
	return k
}

// MustUp is similar to Up
func (k *Keyboard) MustUp(key rune) *Keyboard {
	utils.E(k.Up(key))
	return k
}

// MustPress is similar to Press
func (k *Keyboard) MustPress(key rune) *Keyboard {
	utils.E(k.Press(key))
	return k
}

// MustInsertText is similar to InsertText
func (k *Keyboard) MustInsertText(text string) *Keyboard {
	utils.E(k.InsertText(text))
	return k
}

// MustStart is similar to Start
func (t *Touch) MustStart(points ...*proto.InputTouchPoint) *Touch {
	utils.E(t.Start(points...))
	return t
}

// MustMove is similar to Move
func (t *Touch) MustMove(points ...*proto.InputTouchPoint) *Touch {
	utils.E(t.Move(points...))
	return t
}

// MustEnd is similar to End
func (t *Touch) MustEnd() *Touch {
	utils.E(t.End())
	return t
}

// MustCancel is similar to Cancel
func (t *Touch) MustCancel() *Touch {
	utils.E(t.Cancel())
	return t
}

// MustTap is similar to Tap
func (t *Touch) MustTap(x, y float64) *Touch {
	utils.E(t.Tap(x, y))
	return t
}

// MustDescribe is similar to Describe
func (el *Element) MustDescribe() *proto.DOMNode {
	node, err := el.Describe(1, false)
	utils.E(err)
	return node
}

// MustNodeID is similar to NodeID
func (el *Element) MustNodeID() proto.DOMNodeID {
	id, err := el.NodeID()
	utils.E(err)
	return id
}

// MustShadowRoot is similar to ShadowRoot
func (el *Element) MustShadowRoot() *Element {
	node, err := el.ShadowRoot()
	utils.E(err)
	return node
}

// MustFrame is similar to Frame
func (el *Element) MustFrame() *Page {
	p, err := el.Frame()
	utils.E(err)
	return p
}

// MustFocus is similar to Focus
func (el *Element) MustFocus() *Element {
	utils.E(el.Focus())
	return el
}

// MustScrollIntoView is similar to ScrollIntoView
func (el *Element) MustScrollIntoView() *Element {
	utils.E(el.ScrollIntoView())
	return el
}

// MustHover is similar to Hover
func (el *Element) MustHover() *Element {
	utils.E(el.Hover())
	return el
}

// MustClick is similar to Click
func (el *Element) MustClick() *Element {
	utils.E(el.Click(proto.InputMouseButtonLeft))
	return el
}

// MustTap is similar to Tap
func (el *Element) MustTap() *Element {
	utils.E(el.Tap())
	return el
}

// MustClickable is similar to Clickable
func (el *Element) MustClickable() bool {
	clickable, err := el.Clickable()
	utils.E(err)
	return clickable
}

// MustPress is similar to Press
func (el *Element) MustPress(key rune) *Element {
	utils.E(el.Press(key))
	return el
}

// MustSelectText is similar to SelectText
func (el *Element) MustSelectText(regex string) *Element {
	utils.E(el.SelectText(regex))
	return el
}

// MustSelectAllText is similar to SelectAllText
func (el *Element) MustSelectAllText() *Element {
	utils.E(el.SelectAllText())
	return el
}

// MustInput is similar to Input
func (el *Element) MustInput(text string) *Element {
	utils.E(el.Input(text))
	return el
}

// MustBlur is similar to Blur
func (el *Element) MustBlur() *Element {
	utils.E(el.Blur())
	return el
}

// MustSelect is similar to Select
func (el *Element) MustSelect(selectors ...string) *Element {
	utils.E(el.Select(selectors))
	return el
}

// MustMatches is similar to Matches
func (el *Element) MustMatches(selector string) bool {
	res, err := el.Matches(selector)
	utils.E(err)
	return res
}

// MustAttribute is similar to Attribute
func (el *Element) MustAttribute(name string) *string {
	attr, err := el.Attribute(name)
	utils.E(err)
	return attr
}

// MustProperty is similar to Property
func (el *Element) MustProperty(name string) proto.JSON {
	prop, err := el.Property(name)
	utils.E(err)
	return prop
}

// MustContainsElement is similar to ContainsElement
func (el *Element) MustContainsElement(target *Element) bool {
	contains, err := el.ContainsElement(target)
	utils.E(err)
	return contains
}

// MustSetFiles is similar to SetFiles
func (el *Element) MustSetFiles(paths ...string) *Element {
	utils.E(el.SetFiles(paths))
	return el
}

// MustText is similar to Text
func (el *Element) MustText() string {
	s, err := el.Text()
	utils.E(err)
	return s
}

// MustHTML is similar to HTML
func (el *Element) MustHTML() string {
	s, err := el.HTML()
	utils.E(err)
	return s
}

// MustVisible is similar to Visible
func (el *Element) MustVisible() bool {
	v, err := el.Visible()
	utils.E(err)
	return v
}

// MustWaitLoad is similar to WaitLoad
func (el *Element) MustWaitLoad() *Element {
	utils.E(el.WaitLoad())
	return el
}

// MustWaitStable is similar to WaitStable
func (el *Element) MustWaitStable() *Element {
	utils.E(el.WaitStable(100 * time.Millisecond))
	return el
}

// MustWait is similar to Wait
func (el *Element) MustWait(js string, params ...interface{}) *Element {
	utils.E(el.Wait(js, params))
	return el
}

// MustWaitVisible is similar to WaitVisible
func (el *Element) MustWaitVisible() *Element {
	utils.E(el.WaitVisible())
	return el
}

// MustWaitInvisible is similar to WaitInvisible
func (el *Element) MustWaitInvisible() *Element {
	utils.E(el.WaitInvisible())
	return el
}

// MustBox is similar to Box
func (el *Element) MustBox() *proto.DOMRect {
	box, err := el.Box()
	utils.E(err)
	return box
}

// MustCanvasToImage is similar to CanvasToImage
func (el *Element) MustCanvasToImage(format string, quality float64) []byte {
	bin, err := el.CanvasToImage(format, quality)
	utils.E(err)
	return bin
}

// MustResource is similar to Resource
func (el *Element) MustResource() []byte {
	bin, err := el.Resource()
	utils.E(err)
	return bin
}

// MustScreenshot is similar to Screenshot
func (el *Element) MustScreenshot(toFile ...string) []byte {
	bin, err := el.Screenshot(proto.PageCaptureScreenshotFormatPng, 0)
	utils.E(err)
	utils.E(saveFile(saveFileTypeScreenshot, bin, toFile))
	return bin
}

// MustRelease is similar to Release
func (el *Element) MustRelease() {
	utils.E(el.Release())
}

// MustEval is similar to Eval
func (el *Element) MustEval(js string, params ...interface{}) proto.JSON {
	res, err := el.Eval(js, params...)
	utils.E(err)
	return res.Value
}

// MustHas is similar to Has
func (el *Element) MustHas(selector string) bool {
	has, _, err := el.Has(selector)
	utils.E(err)
	return has
}

// MustHasX is similar to HasX
func (el *Element) MustHasX(selector string) bool {
	has, _, err := el.HasX(selector)
	utils.E(err)
	return has
}

// MustHasR is similar to HasR
func (el *Element) MustHasR(selector, regex string) bool {
	has, _, err := el.HasR(selector, regex)
	utils.E(err)
	return has
}

// MustElement is similar to Element
func (el *Element) MustElement(selector string) *Element {
	el, err := el.Element(selector)
	utils.E(err)
	return el
}

// MustElementX is similar to ElementX
func (el *Element) MustElementX(xpath string) *Element {
	el, err := el.ElementX(xpath)
	utils.E(err)
	return el
}

// MustElementByJS is similar to ElementByJS
func (el *Element) MustElementByJS(js string, params ...interface{}) *Element {
	el, err := el.ElementByJS(NewEvalOptions(js, params))
	utils.E(err)
	return el
}

// MustParent is similar to Parent
func (el *Element) MustParent() *Element {
	parent, err := el.Parent()
	utils.E(err)
	return parent
}

// MustParents is similar to Parents
func (el *Element) MustParents(selector string) Elements {
	list, err := el.Parents(selector)
	utils.E(err)
	return list
}

// MustNext is similar to Next
func (el *Element) MustNext() *Element {
	parent, err := el.Next()
	utils.E(err)
	return parent
}

// MustPrevious is similar to Previous
func (el *Element) MustPrevious() *Element {
	parent, err := el.Previous()
	utils.E(err)
	return parent
}

// MustElementR is similar to ElementR
func (el *Element) MustElementR(selector, regex string) *Element {
	el, err := el.ElementR(selector, regex)
	utils.E(err)
	return el
}

// MustElements is similar to Elements
func (el *Element) MustElements(selector string) Elements {
	list, err := el.Elements(selector)
	utils.E(err)
	return list
}

// MustElementsX is similar to ElementsX
func (el *Element) MustElementsX(xpath string) Elements {
	list, err := el.ElementsX(xpath)
	utils.E(err)
	return list
}

// MustElementsByJS is similar to ElementsByJS
func (el *Element) MustElementsByJS(js string, params ...interface{}) Elements {
	list, err := el.ElementsByJS(NewEvalOptions(js, params))
	utils.E(err)
	return list
}

// MustAdd is similar to Add
func (r *HijackRouter) MustAdd(pattern string, handler func(*Hijack)) *HijackRouter {
	utils.E(r.Add(pattern, "", handler))
	return r
}

// MustRemove is similar to Remove
func (r *HijackRouter) MustRemove(pattern string) *HijackRouter {
	utils.E(r.Remove(pattern))
	return r
}

// MustStop is similar to Stop
func (r *HijackRouter) MustStop() {
	utils.E(r.Stop())
}

// MustLoadResponse is similar to LoadResponse
func (h *Hijack) MustLoadResponse() {
	utils.E(h.LoadResponse(http.DefaultClient, true))
}
