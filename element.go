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
	"github.com/ysmood/rod/lib/cdp"
)

// Element represents the DOM element
type Element struct {
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	page *Page

	ObjectID string
}

// DescribeE doc is the same as the method Describe
func (el *Element) DescribeE() (kit.JSONResult, error) {
	val, err := el.page.Context(el.ctx).CallE(
		"DOM.describeNode",
		cdp.Object{
			"objectId": el.ObjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	node := val.Get("node")
	return &node, nil
}

// FrameE doc is the same as the method Frame
func (el *Element) FrameE() (*Page, error) {
	node, err := el.DescribeE()
	if err != nil {
		return nil, err
	}

	newPage := *el.page
	newPage.FrameID = node.Get("frameId").String()
	newPage.element = el
	newPage.windowObjectID = ""

	return &newPage, nil
}

// FocusE doc is the same as the method Focus
func (el *Element) FocusE() error {
	err := el.ScrollIntoViewIfNeededE()
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, `() => this.focus()`, nil)
	return err
}

// ScrollIntoViewIfNeededE doc is the same as the method ScrollIntoViewIfNeeded
func (el *Element) ScrollIntoViewIfNeededE() error {
	_, err := el.EvalE(true, el.page.jsFn("scrollIntoViewIfNeeded"), nil)
	return err
}

// ClickE doc is the same as the method Click
func (el *Element) ClickE(button string) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.ScrollIntoViewIfNeededE()
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
		defer el.Trace(button + " click")()
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
	_, err = el.EvalE(true, el.page.jsFn("selectText"), cdp.Array{regex})
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

	_, err = el.EvalE(true, el.page.jsFn("select"), cdp.Array{selectors})
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

	_, err := el.page.Context(el.ctx).CallE("DOM.setFileInputFiles", cdp.Object{
		"files":    absPaths,
		"objectId": el.ObjectID,
	})
	return err
}

// TextE doc is the same as the method Text
func (el *Element) TextE() (string, error) {
	str, err := el.EvalE(true, `() => this.innerText`, nil)
	return str.String(), err
}

// HTMLE doc is the same as the method HTML
func (el *Element) HTMLE() (string, error) {
	str, err := el.EvalE(true, `() => this.outerHTML`, nil)
	return str.String(), err
}

// VisibleE doc is the same as the method Visible
func (el *Element) VisibleE() (bool, error) {
	res, err := el.EvalE(true, el.page.jsFn("visible"), nil)
	if err != nil {
		return false, err
	}
	return res.Bool(), nil
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
func (el *Element) WaitE(js string, params cdp.Array) error {
	return kit.Retry(el.ctx, el.page.Sleeper(), func() (bool, error) {
		res, err := el.EvalE(true, js, params)
		if err != nil {
			return true, err
		}

		if res.Bool() {
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
	kit.E(json.Unmarshal([]byte(res.String()), &rect))

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

	res, err := el.page.Context(el.ctx).CallE("Page.getResourceContent", cdp.Object{
		"frameId": el.page.FrameID,
		"url":     src.String(),
	})
	if err != nil {
		return nil, err
	}

	data := res.Get("content").String()

	var bin []byte
	if res.Get("base64Encoded").Bool() {
		bin, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, err
		}
	} else {
		bin = []byte(data)
	}

	return bin, nil
}

// ReleaseE doc is the same as the method Release
func (el *Element) ReleaseE() error {
	return el.page.Context(el.ctx).ReleaseE(el.ObjectID)
}

// EvalE doc is the same as the method Eval
func (el *Element) EvalE(byValue bool, js string, params cdp.Array) (kit.JSONResult, error) {
	return el.page.Context(el.ctx).EvalE(byValue, el.ObjectID, js, params)
}
