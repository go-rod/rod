package rod

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/proto"
)

// Element represents the DOM element
type Element struct {
	ctx           context.Context
	timeoutCancel func()

	page *Page

	ObjectID proto.RuntimeRemoteObjectID
}

// DescribeE doc is the same as the method Describe
func (el *Element) DescribeE() (*proto.DOMNode, error) {
	val, err := proto.DOMDescribeNode{ObjectID: el.ObjectID}.Call(el)
	if err != nil {
		return nil, err
	}
	return val.Node, nil
}

// ShadowRootE returns the shadow root of this element
func (el *Element) ShadowRootE() (*Element, error) {
	node, err := el.DescribeE()
	if err != nil {
		return nil, err
	}

	// though now it's an array, w3c changed the spec of it to be a single.
	id := node.ShadowRoots[0].BackendNodeID

	shadowNode, err := proto.DOMResolveNode{BackendNodeID: id}.Call(el)
	if err != nil {
		return nil, err
	}

	return el.page.ElementFromObjectID(shadowNode.Object.ObjectID), nil
}

// FrameE doc is the same as the method Frame
func (el *Element) FrameE() (*Page, error) {
	node, err := el.DescribeE()
	if err != nil {
		return nil, err
	}

	newPage := *el.page
	newPage.FrameID = node.FrameID
	newPage.element = el
	newPage.windowObjectID = ""

	return &newPage, nil
}

// FocusE doc is the same as the method Focus
func (el *Element) FocusE() error {
	err := el.ScrollIntoViewE()
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, `() => this.focus()`, nil)
	return err
}

// ScrollIntoViewE doc is the same as the method ScrollIntoViewIfNeeded
func (el *Element) ScrollIntoViewE() error {
	_, err := el.EvalE(true, el.page.jsFn("scrollIntoViewIfNeeded"), nil)
	return err
}

// ClickE doc is the same as the method Click
func (el *Element) ClickE(button proto.InputMouseButton) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.ScrollIntoViewE()
	if err != nil {
		return err
	}

	box, err := el.BoxE()
	if err != nil {
		return err
	}

	x := box.Left + box.Width/2
	y := box.Top + box.Height/2

	err = el.page.Mouse.MoveE(x, y, 1)
	if err != nil {
		return err
	}

	if el.page.browser.trace {
		defer el.Trace(string(button) + " click")()
	}

	return el.page.Mouse.ClickE(button)
}

// PressE doc is the same as the method Press
func (el *Element) PressE(key rune) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.FocusE()
	if err != nil {
		return err
	}

	if el.page.browser.trace {
		defer el.Trace("press " + string(key))()
	}

	return el.page.Keyboard.PressE(key)
}

// SelectTextE doc is the same as the method SelectText
func (el *Element) SelectTextE(regex string) error {
	err := el.FocusE()
	if err != nil {
		return err
	}
	_, err = el.EvalE(true, el.page.jsFn("selectText"), Array{regex})
	return err
}

// SelectAllTextE doc is the same as the method SelectAllText
func (el *Element) SelectAllTextE() error {
	err := el.FocusE()
	if err != nil {
		return err
	}
	_, err = el.EvalE(true, el.page.jsFn("selectAllText"), nil)
	return err
}

// InputE doc is the same as the method Input
func (el *Element) InputE(text string) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.FocusE()
	if err != nil {
		return err
	}

	if el.page.browser.trace {
		defer el.Trace("input " + text)()
	}

	err = el.page.Keyboard.InsertTextE(text)
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, el.page.jsFn("inputEvent"), nil)
	return err
}

// SelectE doc is the same as the method Select
func (el *Element) SelectE(selectors []string) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	if el.page.browser.trace {
		defer el.Trace(fmt.Sprintf(
			`<span style="color: #777;">select</span> <code>%s</code>`,
			strings.Join(selectors, "; ")))()
	}

	el.page.browser.trySlowmotion("Input.select")

	_, err = el.EvalE(true, el.page.jsFn("select"), Array{selectors})
	return err
}

// SetFilesE doc is the same as the method SetFiles
func (el *Element) SetFilesE(paths []string) error {
	absPaths := []string{}
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		absPaths = append(absPaths, absPath)
	}

	err := proto.DOMSetFileInputFiles{
		Files:    absPaths,
		ObjectID: el.ObjectID,
	}.Call(el)

	return err
}

// TextE doc is the same as the method Text
func (el *Element) TextE() (string, error) {
	str, err := el.EvalE(true, el.page.jsFn("text"), nil)
	return str.Value.String(), err
}

// HTMLE doc is the same as the method HTML
func (el *Element) HTMLE() (string, error) {
	str, err := el.EvalE(true, `() => this.outerHTML`, nil)
	return str.Value.String(), err
}

// VisibleE doc is the same as the method Visible
func (el *Element) VisibleE() (bool, error) {
	res, err := el.EvalE(true, el.page.jsFn("visible"), nil)
	if err != nil {
		return false, err
	}
	return res.Value.Bool(), nil
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

	for range t.C {
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

// WaitE doc is the same as the method Wait
func (el *Element) WaitE(js string, params Array) error {
	return kit.Retry(el.ctx, el.page.Sleeper(), func() (bool, error) {
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

// WaitVisibleE doc is the same as the method WaitVisible
func (el *Element) WaitVisibleE() error {
	return el.WaitE(el.page.jsFn("visible"), nil)
}

// WaitInvisibleE doc is the same as the method WaitInvisible
func (el *Element) WaitInvisibleE() error {
	return el.WaitE(el.page.jsFn("invisible"), nil)
}

// Box represents the element bounding rect
type Box struct {
	Top    float64 `json:"top"`
	Left   float64 `json:"left"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// BoxE doc is the same as the method Box
func (el *Element) BoxE() (*Box, error) {
	res, err := el.EvalE(true, el.page.jsFn("box"), nil)
	if err != nil {
		return nil, err
	}

	var rect Box
	kit.E(json.Unmarshal([]byte(res.Value.Raw), &rect))

	if el.page.IsIframe() {
		frameRect, err := el.page.element.BoxE() // recursively get the box
		if err != nil {
			return nil, err
		}
		rect.Left += frameRect.Left
		rect.Top += frameRect.Top
	}
	return &rect, nil
}

// ResourceE doc is the same as the method Resource
func (el *Element) ResourceE() ([]byte, error) {
	src, err := el.EvalE(true, el.page.jsFn("resource"), nil)
	if err != nil {
		return nil, err
	}

	res, err := proto.PageGetResourceContent{
		FrameID: el.page.FrameID,
		URL:     src.Value.String(),
	}.Call(el)
	if err != nil {
		return nil, err
	}

	data := res.Content

	var bin []byte
	if res.Base64Encoded {
		bin, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, err
		}
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
			X:      box.Left,
			Y:      box.Top,
			Width:  box.Width,
			Height: box.Height,
			Scale:  1,
		},
	}

	if quality > -1 {
		opts.Quality = int64(quality)
	}

	return el.page.Root().ScreenshotE(opts)
}

// ReleaseE doc is the same as the method Release
func (el *Element) ReleaseE() error {
	return el.page.Context(el.ctx).ReleaseE(el.ObjectID)
}

// CallContext parameters for proto
func (el *Element) CallContext() (context.Context, proto.Client, string) {
	return el.ctx, el.page.browser.client, string(el.page.SessionID)
}

// EvalE doc is the same as the method Eval
func (el *Element) EvalE(byValue bool, js string, params Array) (*proto.RuntimeRemoteObject, error) {
	return el.page.Context(el.ctx).EvalE(byValue, el.ObjectID, js, params)
}
