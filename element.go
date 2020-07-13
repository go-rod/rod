package rod

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	js, jsArgs := el.page.jsHelper("scrollIntoViewIfNeeded", nil)
	_, err := el.EvalE(true, js, jsArgs)
	return err
}

// ClickE doc is similar to the method Click
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

	defer el.tryTrace(string(button) + " click")()

	return el.page.Mouse.ClickE(button)
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

	js, jsArgs := el.page.jsHelper("selectText", Array{regex})
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

	js, jsArgs := el.page.jsHelper("selectAllText", nil)
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

	js, jsArgs := el.page.jsHelper("inputEvent", nil)
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

	js, jsArgs := el.page.jsHelper("select", Array{selectors})
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
		if err != nil {
			return err
		}
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
// But it can choose depth, depth default is 1, -1 to all
// please see https://chromedevtools.github.io/devtools-protocol/tot/DOM/#method-describeNode
func (el *Element) DescribeE(depth int, pierce bool) (*proto.DOMNode, error) {
	var Depth int64
	switch {
	case depth < 0:
		Depth = -1 // -1 to all
	case depth == 0:
		Depth = 1
	default:
		Depth = int64(depth)
	}
	val, err := proto.DOMDescribeNode{ObjectID: el.ObjectID, Depth: Depth, Pierce: pierce}.Call(el)
	if err != nil {
		return nil, err
	}
	return val.Node, nil
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

	return el.page.ElementFromObjectID(shadowNode.Object.ObjectID), nil
}

// FrameE doc is similar to the method Frame
func (el *Element) FrameE() (*Page, error) {
	node, err := el.DescribeE(1, false)
	if err != nil {
		return nil, err
	}

	newPage := *el.page
	newPage.FrameID = node.FrameID
	newPage.element = el
	newPage.windowObjectID = ""

	return &newPage, nil
}

// TextE doc is similar to the method Text
func (el *Element) TextE() (string, error) {
	js, jsArgs := el.page.jsHelper("text", nil)
	str, err := el.EvalE(true, js, jsArgs)
	return str.Value.String(), err
}

// HTMLE doc is similar to the method HTML
func (el *Element) HTMLE() (string, error) {
	str, err := el.EvalE(true, `this.outerHTML`, nil)
	return str.Value.String(), err
}

// VisibleE doc is similar to the method Visible
func (el *Element) VisibleE() (bool, error) {
	js, jsArgs := el.page.jsHelper("visible", nil)
	res, err := el.EvalE(true, js, jsArgs)
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

// WaitE doc is similar to the method Wait
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

// WaitVisibleE doc is similar to the method WaitVisible
func (el *Element) WaitVisibleE() error {
	js, jsArgs := el.page.jsHelper("visible", nil)
	return el.WaitE(js, jsArgs)
}

// WaitInvisibleE doc is similar to the method WaitInvisible
func (el *Element) WaitInvisibleE() error {
	js, jsArgs := el.page.jsHelper("invisible", nil)
	return el.WaitE(js, jsArgs)
}

// Box represents the element bounding rect
type Box struct {
	Top    float64 `json:"top"`
	Left   float64 `json:"left"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// BoxE doc is similar to the method Box
func (el *Element) BoxE() (*Box, error) {
	js, jsArgs := el.page.jsHelper("box", nil)
	res, err := el.EvalE(true, js, jsArgs)
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

// ResourceE doc is similar to the method Resource
func (el *Element) ResourceE() ([]byte, error) {
	js, jsArgs := el.page.jsHelper("resource", nil)
	src, err := el.EvalE(true, js, jsArgs)
	if err != nil {
		return nil, err
	}

	defer el.page.EnableDomain(&proto.PageEnable{})()

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
