package rod

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

// Element implements these interfaces.
var (
	_ proto.Client      = &Element{}
	_ proto.Contextable = &Element{}
	_ proto.Sessionable = &Element{}
)

// Element represents the DOM element.
type Element struct {
	Object *proto.RuntimeRemoteObject

	e eFunc

	ctx context.Context

	sleeper func() utils.Sleeper

	page *Page
}

// GetSessionID interface.
func (el *Element) GetSessionID() proto.TargetSessionID {
	return el.page.SessionID
}

// String interface.
func (el *Element) String() string {
	return fmt.Sprintf("<%s>", el.Object.Description)
}

// Page of the element.
func (el *Element) Page() *Page {
	return el.page
}

// Focus sets focus on the specified element.
// Before the action, it will try to scroll to the element.
func (el *Element) Focus() error {
	err := el.ScrollIntoView()
	if err != nil {
		return err
	}

	_, err = el.Evaluate(Eval(`() => this.focus()`).ByUser())
	return err
}

// ScrollIntoView scrolls the current element into the visible area of the browser
// window if it's not already within the visible area.
func (el *Element) ScrollIntoView() error {
	defer el.tryTrace(TraceTypeInput, "scroll into view")()
	el.page.browser.trySlowMotion()

	err := el.WaitStableRAF()
	if err != nil {
		return err
	}

	return proto.DOMScrollIntoViewIfNeeded{ObjectID: el.id()}.Call(el)
}

// Hover the mouse over the center of the element.
// Before the action, it will try to scroll to the element and wait until it's interactable.
func (el *Element) Hover() error {
	pt, err := el.WaitInteractable()
	if err != nil {
		return err
	}

	return el.page.Context(el.ctx).Mouse.MoveTo(*pt)
}

// MoveMouseOut of the current element.
func (el *Element) MoveMouseOut() error {
	shape, err := el.Shape()
	if err != nil {
		return err
	}
	box := shape.Box()
	return el.page.Mouse.MoveTo(proto.NewPoint(box.X+box.Width, box.Y))
}

// Click will press then release the button just like a human.
// Before the action, it will try to scroll to the element, hover the mouse over it,
// wait until the it's interactable and enabled.
func (el *Element) Click(button proto.InputMouseButton, clickCount int) error {
	err := el.Hover()
	if err != nil {
		return err
	}

	err = el.WaitEnabled()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeInput, string(button)+" click")()

	return el.page.Context(el.ctx).Mouse.Click(button, clickCount)
}

// Tap will scroll to the button and tap it just like a human.
// Before the action, it will try to scroll to the element and wait until it's interactable and enabled.
func (el *Element) Tap() error {
	err := el.ScrollIntoView()
	if err != nil {
		return err
	}

	err = el.WaitEnabled()
	if err != nil {
		return err
	}

	pt, err := el.WaitInteractable()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeInput, "tap")()

	return el.page.Context(el.ctx).Touch.Tap(pt.X, pt.Y)
}

// Interactable checks if the element is interactable with cursor.
// The cursor can be mouse, finger, stylus, etc.
// If not interactable err will be ErrNotInteractable, such as when covered by a modal,.
func (el *Element) Interactable() (pt *proto.Point, err error) {
	noPointerEvents, err := el.Eval(`() => getComputedStyle(this).pointerEvents === 'none'`)
	if err != nil {
		return nil, err
	}

	if noPointerEvents.Value.Bool() {
		return nil, &NoPointerEventsError{el}
	}

	shape, err := el.Shape()
	if err != nil {
		return nil, err
	}

	pt = shape.OnePointInside()
	if pt == nil {
		err = &InvisibleShapeError{el}
		return
	}

	scroll, err := el.page.root.Context(el.ctx).Eval(`() => ({ x: window.scrollX, y: window.scrollY })`)
	if err != nil {
		return
	}

	elAtPoint, err := el.page.Context(el.ctx).ElementFromPoint(
		int(pt.X)+scroll.Value.Get("x").Int(),
		int(pt.Y)+scroll.Value.Get("y").Int(),
	)
	if err != nil {
		if errors.Is(err, cdp.ErrNodeNotFoundAtPos) {
			err = &InvisibleShapeError{el}
		}
		return
	}

	isParent, err := el.ContainsElement(elAtPoint)
	if err != nil {
		return
	}

	if !isParent {
		err = &CoveredError{elAtPoint}
	}
	return
}

// Shape of the DOM element content. The shape is a group of 4-sides polygons.
// A 4-sides polygon is not necessary a rectangle. 4-sides polygons can be apart from each other.
// For example, we use 2 4-sides polygons to describe the shape below:
//
//	  ____________          ____________
//	 /        ___/    =    /___________/    +     _________
//	/________/                                   /________/
func (el *Element) Shape() (*proto.DOMGetContentQuadsResult, error) {
	return proto.DOMGetContentQuads{ObjectID: el.id()}.Call(el)
}

// Type is similar with Keyboard.Type.
// Before the action, it will try to scroll to the element and focus on it.
func (el *Element) Type(keys ...input.Key) error {
	err := el.Focus()
	if err != nil {
		return err
	}
	return el.page.Context(el.ctx).Keyboard.Type(keys...)
}

// KeyActions is similar with Page.KeyActions.
// Before the action, it will try to scroll to the element and focus on it.
func (el *Element) KeyActions() (*KeyActions, error) {
	err := el.Focus()
	if err != nil {
		return nil, err
	}

	return el.page.Context(el.ctx).KeyActions(), nil
}

// SelectText selects the text that matches the regular expression.
// Before the action, it will try to scroll to the element and focus on it.
func (el *Element) SelectText(regex string) error {
	err := el.Focus()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeInput, "select text: "+regex)()
	el.page.browser.trySlowMotion()

	_, err = el.Evaluate(evalHelper(js.SelectText, regex).ByUser())
	return err
}

// SelectAllText selects all text
// Before the action, it will try to scroll to the element and focus on it.
func (el *Element) SelectAllText() error {
	err := el.Focus()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeInput, "select all text")()
	el.page.browser.trySlowMotion()

	_, err = el.Evaluate(evalHelper(js.SelectAllText).ByUser())
	return err
}

// Input focuses on the element and input text to it.
// Before the action, it will scroll to the element, wait until it's visible, enabled and writable.
// To empty the input you can use something like
//
//	el.SelectAllText().MustInput("")
func (el *Element) Input(text string) error {
	err := el.Focus()
	if err != nil {
		return err
	}

	err = el.WaitEnabled()
	if err != nil {
		return err
	}

	err = el.WaitWritable()
	if err != nil {
		return err
	}

	err = el.page.Context(el.ctx).InsertText(text)
	_, _ = el.Evaluate(evalHelper(js.InputEvent).ByUser())
	return err
}

// InputTime focuses on the element and input time to it.
// Before the action, it will scroll to the element, wait until it's visible, enabled and writable.
// It will wait until the element is visible, enabled and writable.
func (el *Element) InputTime(t time.Time) error {
	err := el.Focus()
	if err != nil {
		return err
	}

	err = el.WaitEnabled()
	if err != nil {
		return err
	}

	err = el.WaitWritable()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeInput, "input "+t.String())()

	_, err = el.Evaluate(evalHelper(js.InputTime, t.UnixNano()/1e6).ByUser())
	return err
}

// InputColor focuses on the element and inputs a color string to it.
// Before the action, it will scroll to the element, wait until it's visible, enabled and writable.
func (el *Element) InputColor(color string) error {
	err := el.Focus()
	if err != nil {
		return err
	}

	err = el.WaitEnabled()
	if err != nil {
		return err
	}

	err = el.WaitWritable()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeInput, "input "+color)()

	_, err = el.Evaluate(evalHelper(js.InputColor, color))
	return err
}

// Blur removes focus from the element.
func (el *Element) Blur() error {
	_, err := el.Evaluate(Eval("() => this.blur()").ByUser())
	return err
}

// Select the children option elements that match the selectors.
// Before the action, it will scroll to the element, wait until it's visible.
// If no option matches the selectors, it will return [ErrElementNotFound].
func (el *Element) Select(selectors []string, selected bool, t SelectorType) error {
	err := el.Focus()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeInput, fmt.Sprintf(`select "%s"`, strings.Join(selectors, "; ")))()
	el.page.browser.trySlowMotion()

	res, err := el.Evaluate(evalHelper(js.Select, selectors, selected, t).ByUser())
	if err != nil {
		return err
	}
	if !res.Value.Bool() {
		return &ElementNotFoundError{}
	}
	return nil
}

// Matches checks if the element can be selected by the css selector.
func (el *Element) Matches(selector string) (bool, error) {
	res, err := el.Eval(`s => this.matches(s)`, selector)
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// Attribute of the DOM object.
// Attribute vs Property:
// https://stackoverflow.com/questions/6003819/what-is-the-difference-between-properties-and-attributes-in-html
func (el *Element) Attribute(name string) (*string, error) {
	attr, err := el.Eval("(n) => this.getAttribute(n)", name)
	if err != nil {
		return nil, err
	}

	if attr.Value.Nil() {
		return nil, nil //nolint: nilnil
	}

	s := attr.Value.Str()
	return &s, nil
}

// Property of the DOM object.
// Property vs Attribute:
// https://stackoverflow.com/questions/6003819/what-is-the-difference-between-properties-and-attributes-in-html
func (el *Element) Property(name string) (gson.JSON, error) {
	prop, err := el.Eval("(n) => this[n]", name)
	if err != nil {
		return gson.New(nil), err
	}

	return prop.Value, nil
}

// Disabled checks if the element is disabled.
func (el *Element) Disabled() (bool, error) {
	prop, err := el.Property("disabled")
	if err != nil {
		return false, err
	}
	return prop.Bool(), nil
}

// SetFiles of the current file input element.
func (el *Element) SetFiles(paths []string) error {
	absPaths := utils.AbsolutePaths(paths)

	defer el.tryTrace(TraceTypeInput, fmt.Sprintf("set files: %v", absPaths))()
	el.page.browser.trySlowMotion()

	err := proto.DOMSetFileInputFiles{
		Files:    absPaths,
		ObjectID: el.id(),
	}.Call(el)

	return err
}

// Describe the current element. The depth is the maximum depth at which children should be retrieved, defaults to 1,
// use -1 for the entire subtree or provide an integer larger than 0.
// The pierce decides whether or not iframes and shadow roots should be traversed when returning the subtree.
// The returned [proto.DOMNode.NodeID] will always be empty,
// because NodeID is not stable (when [proto.DOMDocumentUpdated]
// is fired all NodeID on the page will be reassigned to another value)
// we don't recommend using the NodeID, instead, use the [proto.DOMBackendNodeID] to identify the element.
func (el *Element) Describe(depth int, pierce bool) (*proto.DOMNode, error) {
	val, err := proto.DOMDescribeNode{ObjectID: el.id(), Depth: gson.Int(depth), Pierce: pierce}.Call(el)
	if err != nil {
		return nil, err
	}
	return val.Node, nil
}

// ShadowRoot returns the shadow root of this element.
func (el *Element) ShadowRoot() (*Element, error) {
	node, err := el.Describe(1, false)
	if err != nil {
		return nil, err
	}

	// though now it's an array, w3c changed the spec of it to be a single.
	if len(node.ShadowRoots) == 0 {
		return nil, &NoShadowRootError{el}
	}
	id := node.ShadowRoots[0].BackendNodeID

	shadowNode, err := proto.DOMResolveNode{BackendNodeID: id}.Call(el)
	if err != nil {
		return nil, err
	}

	return el.page.Context(el.ctx).ElementFromObject(shadowNode.Object)
}

// Frame creates a page instance that represents the iframe.
func (el *Element) Frame() (*Page, error) {
	node, err := el.Describe(1, false)
	if err != nil {
		return nil, err
	}

	clone := *el.page
	clone.FrameID = node.FrameID
	clone.jsCtxID = new(proto.RuntimeRemoteObjectID)
	clone.element = el
	clone.sleeper = el.sleeper

	return &clone, nil
}

// ContainsElement check if the target is equal or inside the element.
func (el *Element) ContainsElement(target *Element) (bool, error) {
	res, err := el.Evaluate(evalHelper(js.ContainsElement, target.Object))
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// Text that the element displays.
func (el *Element) Text() (string, error) {
	str, err := el.Evaluate(evalHelper(js.Text))
	if err != nil {
		return "", err
	}
	return str.Value.String(), nil
}

// HTML of the element.
func (el *Element) HTML() (string, error) {
	res, err := proto.DOMGetOuterHTML{ObjectID: el.Object.ObjectID}.Call(el)
	if err != nil {
		return "", err
	}
	return res.OuterHTML, nil
}

// Visible returns true if the element is visible on the page.
func (el *Element) Visible() (bool, error) {
	res, err := el.Evaluate(evalHelper(js.Visible))
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// WaitLoad for element like <img>.
func (el *Element) WaitLoad() error {
	defer el.tryTrace(TraceTypeWait, "load")()
	_, err := el.Evaluate(evalHelper(js.WaitLoad).ByPromise())
	return err
}

// WaitStable waits until no shape or position change for d duration.
// Be careful, d is not the max wait timeout, it's the least stable time.
// If you want to set a timeout you can use the [Element.Timeout] function.
func (el *Element) WaitStable(d time.Duration) error {
	err := el.WaitVisible()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeWait, "stable")()

	shape, err := el.Shape()
	if err != nil {
		return err
	}

	t := time.NewTicker(d)
	defer t.Stop()

	for {
		select {
		case <-t.C:
		case <-el.ctx.Done():
			return el.ctx.Err()
		}
		current, err := el.Shape()
		if err != nil {
			return err
		}
		if reflect.DeepEqual(shape, current) {
			break
		}
		shape = current
	}
	return nil
}

// WaitStableRAF waits until no shape or position change for 2 consecutive animation frames.
// If you want to wait animation that is triggered by JS not CSS, you'd better use [Element.WaitStable].
// About animation frame: https://developer.mozilla.org/en-US/docs/Web/API/window/requestAnimationFrame
func (el *Element) WaitStableRAF() error {
	err := el.WaitVisible()
	if err != nil {
		return err
	}

	defer el.tryTrace(TraceTypeWait, "stable RAF")()

	var shape *proto.DOMGetContentQuadsResult
	page := el.page.Context(el.ctx)

	for {
		err = page.WaitRepaint()
		if err != nil {
			return err
		}

		current, err := el.Shape()
		if err != nil {
			return err
		}
		if reflect.DeepEqual(shape, current) {
			break
		}
		shape = current
	}
	return nil
}

// WaitInteractable waits for the element to be interactable.
// It will try to scroll to the element on each try.
func (el *Element) WaitInteractable() (pt *proto.Point, err error) {
	defer el.tryTrace(TraceTypeWait, "interactable")()

	err = utils.Retry(el.ctx, el.sleeper(), func() (bool, error) {
		// For lazy loading page the element can be outside of the viewport.
		// If we don't scroll to it, it will never be available.
		err := el.ScrollIntoView()
		if err != nil {
			return true, err
		}

		pt, err = el.Interactable()
		if errors.Is(err, &CoveredError{}) {
			return false, nil
		}
		return true, err
	})
	return
}

// Wait until the js returns true.
func (el *Element) Wait(opts *EvalOptions) error {
	return el.page.Context(el.ctx).Sleeper(el.sleeper).Wait(opts.This(el.Object))
}

// WaitVisible until the element is visible.
func (el *Element) WaitVisible() error {
	defer el.tryTrace(TraceTypeWait, "visible")()
	return el.Wait(evalHelper(js.Visible))
}

// WaitEnabled until the element is not disabled.
// Doc for readonly: https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/readonly
func (el *Element) WaitEnabled() error {
	defer el.tryTrace(TraceTypeWait, "enabled")()
	return el.Wait(Eval(`() => !this.disabled`))
}

// WaitWritable until the element is not readonly.
// Doc for disabled: https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/disabled
func (el *Element) WaitWritable() error {
	defer el.tryTrace(TraceTypeWait, "writable")()
	return el.Wait(Eval(`() => !this.readonly`))
}

// WaitInvisible until the element invisible.
func (el *Element) WaitInvisible() error {
	defer el.tryTrace(TraceTypeWait, "invisible")()
	return el.Wait(evalHelper(js.Invisible))
}

// CanvasToImage get image data of a canvas.
// The default format is image/png.
// The default quality is 0.92.
// doc: https://developer.mozilla.org/en-US/docs/Web/API/HTMLCanvasElement/toDataURL
func (el *Element) CanvasToImage(format string, quality float64) ([]byte, error) {
	res, err := el.Eval(`(format, quality) => this.toDataURL(format, quality)`, format, quality)
	if err != nil {
		return nil, err
	}

	_, bin := parseDataURI(res.Value.Str())
	return bin, nil
}

// Resource returns the "src" content of current element. Such as the jpg of <img src="a.jpg">.
func (el *Element) Resource() ([]byte, error) {
	src, err := el.Evaluate(evalHelper(js.Resource).ByPromise())
	if err != nil {
		return nil, err
	}

	return el.page.Context(el.ctx).GetResource(src.Value.String())
}

// BackgroundImage returns the css background-image of the element.
func (el *Element) BackgroundImage() ([]byte, error) {
	res, err := el.Eval(`() => window.getComputedStyle(this).backgroundImage.replace(/^url\("/, '').replace(/"\)$/, '')`)
	if err != nil {
		return nil, err
	}

	u := res.Value.Str()

	return el.page.Context(el.ctx).GetResource(u)
}

// Screenshot of the area of the element.
func (el *Element) Screenshot(format proto.PageCaptureScreenshotFormat, quality int) ([]byte, error) {
	err := el.ScrollIntoView()
	if err != nil {
		return nil, err
	}

	opts := &proto.PageCaptureScreenshot{
		Quality: gson.Int(quality),
		Format:  format,
	}

	bin, err := el.page.Context(el.ctx).Screenshot(false, opts)
	if err != nil {
		return nil, err
	}

	// so that it won't clip the css-transformed element
	shape, err := el.Shape()
	if err != nil {
		return nil, err
	}

	box := shape.Box()

	// TODO: proto.PageCaptureScreenshot has a Clip option, but it's buggy, so now we do in Go.
	return utils.CropImage(bin, quality,
		int(box.X),
		int(box.Y),
		int(box.Width),
		int(box.Height),
	)
}

// Release is a shortcut for [Page.Release] current element.
func (el *Element) Release() error {
	return el.page.Context(el.ctx).Release(el.Object)
}

// Remove the element from the page.
func (el *Element) Remove() error {
	_, err := el.Eval(`() => this.remove()`)
	if err != nil {
		return err
	}
	return el.Release()
}

// Call implements the [proto.Client].
func (el *Element) Call(ctx context.Context, sessionID, methodName string, params interface{}) (res []byte, err error) {
	return el.page.Call(ctx, sessionID, methodName, params)
}

// Eval is a shortcut for [Element.Evaluate] with AwaitPromise, ByValue and AutoExp set to true.
func (el *Element) Eval(js string, params ...interface{}) (*proto.RuntimeRemoteObject, error) {
	return el.Evaluate(Eval(js, params...).ByPromise())
}

// Evaluate is just a shortcut of [Page.Evaluate] with This set to current element.
func (el *Element) Evaluate(opts *EvalOptions) (*proto.RuntimeRemoteObject, error) {
	return el.page.Context(el.ctx).Evaluate(opts.This(el.Object))
}

// Equal checks if the two elements are equal.
func (el *Element) Equal(elm *Element) (bool, error) {
	res, err := el.Eval(`elm => this === elm`, elm.Object)
	return res.Value.Bool(), err
}

func (el *Element) id() proto.RuntimeRemoteObjectID {
	return el.Object.ObjectID
}

// GetXPath returns the xpath of the element.
func (el *Element) GetXPath(optimized bool) (string, error) {
	str, err := el.Evaluate(evalHelper(js.GetXPath, optimized))
	if err != nil {
		return "", err
	}
	return str.Value.String(), nil
}
