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

	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/kit"
)

// Element represents the DOM element
type Element struct {
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	page *Page

	ObjectID proto.RuntimeRemoteObjectID
}

// FocusE doc is similar to the method Focus
func (el *Element) FocusE() error {
	err := el.ScrollIntoViewE()
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, `this.focus()`, nil)
	return err
}

// ScrollIntoViewE doc is similar to the method ScrollIntoViewIfNeeded
func (el *Element) ScrollIntoViewE() error {
	defer el.tryTrace("scroll into view")()
	el.page.browser.trySlowmotion()

	return proto.DOMScrollIntoViewIfNeeded{ObjectID: el.ObjectID}.Call(el)
}

// ClickE will press then release the button just like a human.
func (el *Element) ClickE(button proto.InputMouseButton) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.ScrollIntoViewE()
	if err != nil {
		return err
	}

	box, err := el.boxCenter()
	if err != nil {
		return err
	}

	err = el.page.Mouse.MoveE(box.X, box.Y, 1)
	if err != nil {
		return err
	}

	clickable, err := el.ClickableE()
	if err != nil {
		return err
	}
	if !clickable {
		return fmt.Errorf("%w: %s", newErr(ErrNotClickable, el.HTML()), "such as covered by a modal")
	}

	defer el.tryTrace(string(button) + " click")()

	return el.page.Mouse.ClickE(button)
}

// ClickableE checks if the element is behind another element, such as when invisible or covered by a modal.
func (el *Element) ClickableE() (bool, error) {
	box, err := el.boxCenter()
	if err != nil {
		return false, err
	}

	scroll, err := el.page.Root().EvalE(true, "", `{ x: window.scrollX, y: window.scrollY }`, nil)
	if err != nil {
		return false, err
	}

	elAtPoint, err := el.page.ElementFromPointE(
		int64(box.X)+scroll.Value.Get("x").Int(),
		int64(box.Y)+scroll.Value.Get("y").Int(),
	)
	if err != nil {
		return false, err
	}

	contains, err := el.ContainsElementE(elAtPoint)
	if err != nil {
		return false, err
	}

	if contains {
		return true, nil
	}

	return false, nil
}

func (el *Element) boxCenter() (*proto.DOMRect, error) {
	box, err := el.BoxE()
	if err != nil {
		return nil, err
	}

	x := box.X + box.Width/2
	y := box.Y + box.Height/2

	return &proto.DOMRect{X: x, Y: y}, nil
}

// BoxE returns the size of an element and its position relative to the main frame.
func (el *Element) BoxE() (*proto.DOMRect, error) {
	res, err := proto.DOMGetBoxModel{ObjectID: el.ObjectID}.Call(el)
	if err != nil {
		return nil, err
	}
	return res.Model.Rect(), nil
}

// PressE doc is similar to the method Press
func (el *Element) PressE(key rune) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.FocusE()
	if err != nil {
		return err
	}

	defer el.tryTrace("press " + string(key))()

	return el.page.Keyboard.PressE(key)
}

// SelectTextE doc is similar to the method SelectText
func (el *Element) SelectTextE(regex string) error {
	err := el.FocusE()
	if err != nil {
		return err
	}

	defer el.tryTrace("select text: " + regex)()
	el.page.browser.trySlowmotion()

	js, jsArgs := jsHelper("selectText", Array{regex})
	_, err = el.EvalE(true, js, jsArgs)
	return err
}

// SelectAllTextE doc is similar to the method SelectAllText
func (el *Element) SelectAllTextE() error {
	err := el.FocusE()
	if err != nil {
		return err
	}

	defer el.tryTrace("select all text")()
	el.page.browser.trySlowmotion()

	js, jsArgs := jsHelper("selectAllText", nil)
	_, err = el.EvalE(true, js, jsArgs)
	return err
}

// InputE doc is similar to the method Input
func (el *Element) InputE(text string) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.FocusE()
	if err != nil {
		return err
	}

	defer el.tryTrace("input " + text)()

	err = el.page.Keyboard.InsertTextE(text)
	if err != nil {
		return err
	}

	js, jsArgs := jsHelper("inputEvent", nil)
	_, err = el.EvalE(true, js, jsArgs)
	return err
}

// BlurE is similar to the method Blur
func (el *Element) BlurE() error {
	_, err := el.EvalE(true, "this.blur()", nil)
	return err
}

// SelectE doc is similar to the method Select
func (el *Element) SelectE(selectors []string) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	defer el.tryTrace(fmt.Sprintf(
		`select "%s"`,
		strings.Join(selectors, "; ")))()
	el.page.browser.trySlowmotion()

	js, jsArgs := jsHelper("select", Array{selectors})
	_, err = el.EvalE(true, js, jsArgs)
	return err
}

// MatchesE checks if the element can be selected by the css selector
func (el *Element) MatchesE(selector string) (bool, error) {
	res, err := el.EvalE(true, `s => this.matches(s)`, Array{selector})
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// AttributeE is similar to the method Attribute
func (el *Element) AttributeE(name string) (*string, error) {
	attr, err := el.EvalE(true, "(n) => this.getAttribute(n)", Array{name})
	if err != nil {
		return nil, err
	}

	if attr.Value.Type == gjson.Null {
		return nil, nil
	}

	return &attr.Value.Str, nil
}

// PropertyE is similar to the method Property
func (el *Element) PropertyE(name string) (proto.JSON, error) {
	prop, err := el.EvalE(true, "(n) => this[n]", Array{name})
	if err != nil {
		return proto.JSON{}, err
	}

	return prop.Value, nil
}

// SetFilesE doc is similar to the method SetFiles
func (el *Element) SetFilesE(paths []string) error {
	absPaths := []string{}
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		kit.E(err)
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

// DescribeE doc is similar to the method Describe
// please see https://chromedevtools.github.io/devtools-protocol/tot/DOM/#method-describeNode
func (el *Element) DescribeE(depth int, pierce bool) (*proto.DOMNode, error) {
	val, err := proto.DOMDescribeNode{ObjectID: el.ObjectID, Depth: int64(depth), Pierce: pierce}.Call(el)
	if err != nil {
		return nil, err
	}
	return val.Node, nil
}

// NodeIDE of the node
func (el *Element) NodeIDE() (proto.DOMNodeID, error) {
	el.page.enableNodeQuery()
	node, err := proto.DOMRequestNode{ObjectID: el.ObjectID}.Call(el)
	if err != nil {
		return 0, err
	}
	return node.NodeID, nil
}

// ShadowRootE returns the shadow root of this element
func (el *Element) ShadowRootE() (*Element, error) {
	node, err := el.DescribeE(1, false)
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

// ContainsElementE check if the target is equal or inside the element.
func (el *Element) ContainsElementE(target *Element) (bool, error) {
	js, args := jsHelper("containsElement", Array{target.ObjectID})
	res, err := el.EvalE(true, js, args)
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// TextE doc is similar to the method Text
func (el *Element) TextE() (string, error) {
	js, jsArgs := jsHelper("text", nil)
	str, err := el.EvalE(true, js, jsArgs)
	if err != nil {
		return "", err
	}
	return str.Value.String(), nil
}

// HTMLE doc is similar to the method HTML
func (el *Element) HTMLE() (string, error) {
	str, err := el.EvalE(true, `this.outerHTML`, nil)
	if err != nil {
		return "", err
	}
	return str.Value.String(), nil
}

// VisibleE doc is similar to the method Visible
func (el *Element) VisibleE() (bool, error) {
	js, jsArgs := jsHelper("visible", nil)
	res, err := el.EvalE(true, js, jsArgs)
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
}

// WaitLoadE for element like <img />
func (el *Element) WaitLoadE() error {
	js, jsArgs := jsHelper("waitLoad", nil)
	_, err := el.EvalE(true, js, jsArgs)
	return err
}

// WaitStableE not using requestAnimation here because it can trigger to many checks,
// or miss checks for jQuery css animation.
func (el *Element) WaitStableE(interval time.Duration) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	box := el.Box()

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
		case <-el.ctx.Done():
			return el.ctx.Err()
		}
		current := el.Box()
		if *box == *current {
			break
		}
		box = current
	}
	return nil
}

// WaitE doc is similar to the method Wait
func (el *Element) WaitE(js string, params Array) error {
	return kit.Retry(el.ctx, Sleeper(), func() (bool, error) {
		res, err := el.EvalE(true, js, params)
		if err != nil {
			return true, err
		}

		if res.Value.Bool() {
			return true, nil
		}

		return false, nil
	})
}

// WaitVisibleE doc is similar to the method WaitVisible
func (el *Element) WaitVisibleE() error {
	js, jsArgs := jsHelper("visible", nil)
	return el.WaitE(js, jsArgs)
}

// WaitInvisibleE doc is similar to the method WaitInvisible
func (el *Element) WaitInvisibleE() error {
	js, jsArgs := jsHelper("invisible", nil)
	return el.WaitE(js, jsArgs)
}

// CanvasToImageE get image data of a canvas.
// The default format is image/png.
// The default quality is 0.92.
// doc: https://developer.mozilla.org/en-US/docs/Web/API/HTMLCanvasElement/toDataURL
func (el *Element) CanvasToImageE(format string, quality float64) ([]byte, error) {
	res, err := el.EvalE(true,
		`(format, quality) => this.toDataURL(format, quality)`,
		Array{format, quality})
	if err != nil {
		return nil, err
	}

	_, bin := parseDataURI(res.Value.Str)
	return bin, nil
}

// ResourceE doc is similar to the method Resource
func (el *Element) ResourceE() ([]byte, error) {
	js, jsArgs := jsHelper("resource", nil)
	src, err := el.EvalE(true, js, jsArgs)
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
		kit.E(err)
	} else {
		bin = []byte(data)
	}

	return bin, nil
}

// ScreenshotE of the area of the element
func (el *Element) ScreenshotE(format proto.PageCaptureScreenshotFormat, quality int) ([]byte, error) {
	err := el.WaitVisibleE()
	if err != nil {
		return nil, err
	}

	err = el.ScrollIntoViewE()
	if err != nil {
		return nil, err
	}

	box, err := el.BoxE()
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

	return el.page.Root().ScreenshotE(false, opts)
}

// ReleaseE doc is similar to the method Release
func (el *Element) ReleaseE() error {
	err := el.page.Context(el.ctx, el.ctxCancel).ReleaseE(el.ObjectID)
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

// EvalE doc is similar to the method Eval
func (el *Element) EvalE(byValue bool, js string, params Array) (*proto.RuntimeRemoteObject, error) {
	return el.page.Context(el.ctx, el.ctxCancel).EvalE(byValue, el.ObjectID, js, params)
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
		list, err := page.ElementsE("", "iframe")
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
