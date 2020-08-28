package rod

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// Element represents the DOM element
type Element struct {
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()
	sleeper       utils.Sleeper

	page *Page

	ObjectID proto.RuntimeRemoteObjectID
}

// Focus doc is similar to the method MustFocus
func (el *Element) Focus() error {
	err := el.ScrollIntoView()
	if err != nil {
		return err
	}

	_, err = el.Eval(`this.focus()`)
	return err
}

// ScrollIntoView doc is similar to the method MustScrollIntoViewIfNeeded
func (el *Element) ScrollIntoView() error {
	defer el.tryTrace("scroll into view")()
	el.page.browser.trySlowmotion()

	return proto.DOMScrollIntoViewIfNeeded{ObjectID: el.ObjectID}.Call(el)
}

// Hover the mouse over the center of the element.
func (el *Element) Hover() error {
	err := el.WaitVisible()
	if err != nil {
		return err
	}

	err = el.ScrollIntoView()
	if err != nil {
		return err
	}

	box, err := el.Box()
	if err != nil {
		return err
	}

	err = el.page.Mouse.Move(box.CenterX(), box.CenterY(), 1)
	if err != nil {
		return err
	}

	return nil
}

// Click will press then release the button just like a human.
func (el *Element) Click(button proto.InputMouseButton) error {
	err := el.Hover()
	if err != nil {
		return err
	}

	clickable, err := el.Clickable()
	if err != nil {
		return err
	}
	if !clickable {
		s, err := el.HTML()
		if err != nil {
			return err
		}
		return newErr(ErrNotClickable, s, "such as covered by a modal")
	}

	defer el.tryTrace(string(button) + " click")()

	return el.page.Mouse.Click(button)
}

// Clickable checks if the element is behind another element, such as when invisible or covered by a modal.
func (el *Element) Clickable() (bool, error) {
	box, err := el.Box()
	if err != nil {
		return false, err
	}

	scroll, err := el.page.Root().Eval(`{ x: window.scrollX, y: window.scrollY }`)
	if err != nil {
		return false, err
	}

	elAtPoint, err := el.page.ElementFromPoint(
		int64(box.CenterX())+scroll.Value.Get("x").Int(),
		int64(box.CenterY())+scroll.Value.Get("y").Int(),
	)
	if err != nil {
		return false, err
	}

	contains, err := el.ContainsElement(elAtPoint)
	if err != nil {
		return false, err
	}

	if contains {
		return true, nil
	}

	return false, nil
}

// Box returns the size of an element and its position relative to the main frame.
func (el *Element) Box() (*proto.DOMRect, error) {
	res, err := proto.DOMGetBoxModel{ObjectID: el.ObjectID}.Call(el)
	if err != nil {
		return nil, err
	}
	return res.Model.Rect(), nil
}

// Press doc is similar to the method MustPress
func (el *Element) Press(key rune) error {
	err := el.WaitVisible()
	if err != nil {
		return err
	}

	err = el.Focus()
	if err != nil {
		return err
	}

	defer el.tryTrace("press " + string(key))()

	return el.page.Keyboard.Press(key)
}

// SelectText doc is similar to the method MustSelectText
func (el *Element) SelectText(regex string) error {
	err := el.Focus()
	if err != nil {
		return err
	}

	defer el.tryTrace("select text: " + regex)()
	el.page.browser.trySlowmotion()

	_, err = el.EvalWithOptions(jsHelper(js.SelectText, Array{regex}))
	return err
}

// SelectAllText doc is similar to the method MustSelectAllText
func (el *Element) SelectAllText() error {
	err := el.Focus()
	if err != nil {
		return err
	}

	defer el.tryTrace("select all text")()
	el.page.browser.trySlowmotion()

	_, err = el.EvalWithOptions(jsHelper(js.SelectAllText, nil))
	return err
}

// Input doc is similar to the method MustInput
func (el *Element) Input(text string) error {
	err := el.WaitVisible()
	if err != nil {
		return err
	}

	err = el.Focus()
	if err != nil {
		return err
	}

	defer el.tryTrace("input " + text)()

	err = el.page.Keyboard.InsertText(text)
	if err != nil {
		return err
	}

	_, err = el.EvalWithOptions(jsHelper(js.InputEvent, nil))
	return err
}

// Blur is similar to the method Blur
func (el *Element) Blur() error {
	_, err := el.Eval("this.blur()")
	return err
}

// Select doc is similar to the method MustSelect
func (el *Element) Select(selectors []string) error {
	err := el.WaitVisible()
	if err != nil {
		return err
	}

	defer el.tryTrace(fmt.Sprintf(
		`select "%s"`,
		strings.Join(selectors, "; ")))()
	el.page.browser.trySlowmotion()

	_, err = el.EvalWithOptions(jsHelper(js.Select, Array{selectors}))
	return err
}

// Matches checks if the element can be selected by the css selector
func (el *Element) Matches(selector string) (bool, error) {
	res, err := el.Eval(`s => this.matches(s)`, selector)
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// Attribute is similar to the method Attribute
func (el *Element) Attribute(name string) (*string, error) {
	attr, err := el.Eval("(n) => this.getAttribute(n)", name)
	if err != nil {
		return nil, err
	}

	if attr.Value.Type == gjson.Null {
		return nil, nil
	}

	return &attr.Value.Str, nil
}

// Property is similar to the method Property
func (el *Element) Property(name string) (proto.JSON, error) {
	prop, err := el.Eval("(n) => this[n]", name)
	if err != nil {
		return proto.JSON{}, err
	}

	return prop.Value, nil
}

// SetFiles doc is similar to the method MustSetFiles
func (el *Element) SetFiles(paths []string) error {
	absPaths := []string{}
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		utils.E(err)
		absPaths = append(absPaths, absPath)
	}

	defer el.tryTrace(fmt.Sprintf("set files: %v", absPaths))
	el.page.browser.trySlowmotion()

	err := proto.DOMSetFileInputFiles{
		Files:    absPaths,
		ObjectID: el.ObjectID,
	}.Call(el)

	return err
}

// Describe doc is similar to the method MustDescribe
// please see https://chromedevtools.github.io/devtools-protocol/tot/DOM/#method-describeNode
func (el *Element) Describe(depth int, pierce bool) (*proto.DOMNode, error) {
	val, err := proto.DOMDescribeNode{ObjectID: el.ObjectID, Depth: int64(depth), Pierce: pierce}.Call(el)
	if err != nil {
		return nil, err
	}
	return val.Node, nil
}

// NodeID of the node
func (el *Element) NodeID() (proto.DOMNodeID, error) {
	el.page.enableNodeQuery()
	node, err := proto.DOMRequestNode{ObjectID: el.ObjectID}.Call(el)
	if err != nil {
		return 0, err
	}
	return node.NodeID, nil
}

// ShadowRoot returns the shadow root of this element
func (el *Element) ShadowRoot() (*Element, error) {
	node, err := el.Describe(1, false)
	if err != nil {
		return nil, err
	}

	// though now it's an array, w3c changed the spec of it to be a single.
	id := node.ShadowRoots[0].BackendNodeID

	shadowNode, err := proto.DOMResolveNode{BackendNodeID: id}.Call(el)
	if err != nil {
		return nil, err
	}

	return el.page.ElementFromObject(shadowNode.Object.ObjectID), nil
}

// Frame creates a page instance that represents the iframe
func (el *Element) Frame() *Page {
	newPage := *el.page
	newPage.element = el
	newPage.jsHelperObjectID = ""
	newPage.windowObjectID = ""
	return &newPage
}

// ContainsElement check if the target is equal or inside the element.
func (el *Element) ContainsElement(target *Element) (bool, error) {
	res, err := el.EvalWithOptions(jsHelper(js.ContainsElement, Array{target.ObjectID}))
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// Text doc is similar to the method MustText
func (el *Element) Text() (string, error) {
	str, err := el.EvalWithOptions(jsHelper(js.Text, nil))
	if err != nil {
		return "", err
	}
	return str.Value.String(), nil
}

// HTML doc is similar to the method MustHTML
func (el *Element) HTML() (string, error) {
	str, err := el.Eval(`this.outerHTML`)
	if err != nil {
		return "", err
	}
	return str.Value.String(), nil
}

// Visible doc is similar to the method MustVisible
func (el *Element) Visible() (bool, error) {
	res, err := el.EvalWithOptions(jsHelper(js.Visible, nil))
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// WaitLoad for element like <img />
func (el *Element) WaitLoad() error {
	_, err := el.EvalWithOptions(jsHelper(js.WaitLoad, nil))
	return err
}

// WaitStable not using requestAnimation here because it can trigger to many checks,
// or miss checks for jQuery css animation.
func (el *Element) WaitStable(interval time.Duration) error {
	err := el.WaitVisible()
	if err != nil {
		return err
	}

	box, err := el.Box()
	if err != nil {
		return err
	}

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
		case <-el.ctx.Done():
			return el.ctx.Err()
		}
		current, err := el.Box()
		if err != nil {
			return err
		}
		if *box == *current {
			break
		}
		box = current
	}
	return nil
}

// Wait doc is similar to the method MustWait
func (el *Element) Wait(js string, params ...interface{}) error {
	return utils.Retry(el.ctx, el.sleeper, func() (bool, error) {
		res, err := el.Eval(js, params...)
		if err != nil {
			return true, err
		}

		if res.Value.Bool() {
			return true, nil
		}

		return false, nil
	})
}

// WaitVisible doc is similar to the method MustWaitVisible
func (el *Element) WaitVisible() error {
	opts := jsHelper(js.Visible, nil)
	return el.Wait(opts.JS, opts.JSArgs...)
}

// WaitInvisible doc is similar to the method MustWaitInvisible
func (el *Element) WaitInvisible() error {
	opts := jsHelper(js.Invisible, nil)
	return el.Wait(opts.JS, opts.JSArgs...)
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

	_, bin := parseDataURI(res.Value.Str)
	return bin, nil
}

// Resource doc is similar to the method MustResource
func (el *Element) Resource() ([]byte, error) {
	src, err := el.EvalWithOptions(jsHelper(js.Resource, nil))
	if err != nil {
		return nil, err
	}

	defer el.page.EnableDomain(&proto.PageEnable{})()

	frameID, err := el.page.frameID()
	if err != nil {
		return nil, err
	}

	res, err := proto.PageGetResourceContent{
		FrameID: frameID,
		URL:     src.Value.String(),
	}.Call(el)
	if err != nil {
		return nil, err
	}

	data := res.Content

	var bin []byte
	if res.Base64Encoded {
		bin, err = base64.StdEncoding.DecodeString(data)
		utils.E(err)
	} else {
		bin = []byte(data)
	}

	return bin, nil
}

// Screenshot of the area of the element
func (el *Element) Screenshot(format proto.PageCaptureScreenshotFormat, quality int) ([]byte, error) {
	err := el.WaitVisible()
	if err != nil {
		return nil, err
	}

	err = el.ScrollIntoView()
	if err != nil {
		return nil, err
	}

	box, err := el.Box()
	if err != nil {
		return nil, err
	}

	opts := &proto.PageCaptureScreenshot{
		Format: format,
		Clip: &proto.PageViewport{
			X:      box.X,
			Y:      box.Y,
			Width:  box.Width,
			Height: box.Height,
			Scale:  1,
		},
	}

	return el.page.Root().Screenshot(false, opts)
}

// Release doc is similar to the method MustRelease
func (el *Element) Release() error {
	err := el.page.Context(el.ctx, el.ctxCancel).Release(el.ObjectID)
	if err != nil {
		return err
	}

	el.ctxCancel()
	return nil
}

// CallContext parameters for proto
func (el *Element) CallContext() (context.Context, proto.Client, string) {
	return el.ctx, el.page.browser, string(el.page.SessionID)
}

// Eval doc is similar to the method MustEval
func (el *Element) Eval(js string, params ...interface{}) (*proto.RuntimeRemoteObject, error) {
	return el.EvalWithOptions(NewEvalOptions(js, params))
}

// EvalWithOptions of Eval
func (el *Element) EvalWithOptions(opts *EvalOptions) (*proto.RuntimeRemoteObject, error) {
	return el.page.Context(el.ctx, el.ctxCancel).EvalWithOptions(opts.This(el.ObjectID))
}

func (el *Element) ensureParentPage(nodeID proto.DOMNodeID, objID proto.RuntimeRemoteObjectID) error {
	has, err := el.page.hasElement(objID)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	// DFS for the iframe that holds the element
	var walk func(page *Page) error
	walk = func(page *Page) error {
		list, err := page.Elements("iframe")
		if err != nil {
			return err
		}

		for _, f := range list {
			p := f.Frame()

			objID, err := p.resolveNode(nodeID)
			if err != nil {
				return err
			}
			if objID != "" {
				el.page = p
				el.ObjectID = objID
				return io.EOF
			}

			err = walk(p)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err = walk(el.page)
	if err == io.EOF {
		return nil
	}
	return err
}
