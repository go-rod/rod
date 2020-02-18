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
	ctx  context.Context
	page *Page

	ObjectID string

	timeoutCancel func()
}

// Context sets the context for chained sub-operations
func (el *Element) Context(ctx context.Context) *Element {
	newObj := *el
	newObj.ctx = ctx
	return &newObj
}

// Timeout sets the timeout for chained sub-operations
func (el *Element) Timeout(d time.Duration) *Element {
	ctx, cancel := context.WithTimeout(el.ctx, d)
	el.timeoutCancel = cancel
	return el.Context(ctx)
}

// CancelTimeout ...
func (el *Element) CancelTimeout() *Element {
	if el.timeoutCancel != nil {
		el.timeoutCancel()
	}
	return el
}

// DescribeE ...
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

// FrameE ...
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

// FocusE ...
func (el *Element) FocusE() error {
	err := el.ScrollIntoViewIfNeededE()
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, `() => this.focus()`)
	return err
}

// ScrollIntoViewIfNeededE ...
func (el *Element) ScrollIntoViewIfNeededE() error {
	_, err := el.EvalE(true, el.page.jsFn("scrollIntoViewIfNeeded"))
	return err
}

// ClickE ...
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

	x := box.Get("left").Int() + box.Get("width").Int()/2
	y := box.Get("top").Int() + box.Get("height").Int()/2

	err = el.page.Mouse.MoveE(x, y, 1)
	if err != nil {
		return err
	}

	defer el.Trace(button + " click")()

	return el.page.Mouse.ClickE(button)
}

// PressE ...
func (el *Element) PressE(key rune) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.FocusE()
	if err != nil {
		return err
	}

	defer el.Trace("press " + string(key))()

	return el.page.Keyboard.PressE(key)
}

// InputE ...
func (el *Element) InputE(text string) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	err = el.FocusE()
	if err != nil {
		return err
	}

	defer el.Trace("input " + text)()

	err = el.page.Keyboard.InsertTextE(text)
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, el.page.jsFn("inputEvent"))
	return err
}

// SelectE ...
func (el *Element) SelectE(selectors ...string) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	defer el.Trace(fmt.Sprintf(
		`<span style="color: #777;">select</span> <code>%s</code>`,
		strings.Join(selectors, "; ")))()
	el.page.browser.trySlowmotion("Input.select")

	_, err = el.EvalE(true, el.page.jsFn("select"), selectors)
	return err
}

// SetFilesE ...
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

// TextE ...
func (el *Element) TextE() (string, error) {
	str, err := el.EvalE(true, `() => this.innerText`)
	return str.String(), err
}

// HTMLE ...
func (el *Element) HTMLE() (string, error) {
	str, err := el.EvalE(true, `() => this.outerHTML`)
	return str.String(), err
}

// WaitStableE not using requestAnimation here because it can trigger to many checks,
// or miss checks for jQuery css animation.
func (el *Element) WaitStableE(interval time.Duration) error {
	err := el.WaitVisibleE()
	if err != nil {
		return err
	}

	box := el.Box().Raw

	t := time.NewTicker(interval)
	defer t.Stop()

	for range t.C {
		select {
		case <-t.C:
		case <-el.ctx.Done():
			return el.ctx.Err()
		}
		current := el.Box().Raw
		if box == current {
			break
		}
		box = current
	}
	return nil
}

// WaitE ...
func (el *Element) WaitE(js string, params ...interface{}) error {
	return kit.Retry(el.ctx, el.page.Sleeper(), func() (bool, error) {
		res, err := el.EvalE(true, js, params...)
		if err != nil {
			return true, err
		}

		if res.Bool() {
			return true, nil
		}

		return false, nil
	})
}

// WaitVisibleE ...
func (el *Element) WaitVisibleE() error {
	return el.WaitE(el.page.jsFn("waitVisible"))
}

// WaitInvisibleE ...
func (el *Element) WaitInvisibleE() error {
	return el.WaitE(el.page.jsFn("waitInvisible"))
}

// BoxE ...
func (el *Element) BoxE() (kit.JSONResult, error) {
	box, err := el.EvalE(true, el.page.jsFn("box"))
	if err != nil {
		return nil, err
	}

	var j map[string]interface{}
	kit.E(json.Unmarshal([]byte(box.String()), &j))

	if el.page.IsIframe() {
		frameRect, err := el.page.element.BoxE() // recursively get the box
		if err != nil {
			return nil, err
		}
		j["left"] = box.Get("left").Int() + frameRect.Get("left").Int()
		j["top"] = box.Get("top").Int() + frameRect.Get("top").Int()
	}
	return kit.JSON(kit.MustToJSON(j)), nil
}

// ResourceE ...
func (el *Element) ResourceE() ([]byte, error) {
	src, err := el.EvalE(true, el.page.jsFn("resource"))
	if err != nil {
		return nil, err
	}

	res, err := el.page.CallE("Page.getResourceContent", cdp.Object{
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

// ReleaseE ...
func (el *Element) ReleaseE() error {
	return el.page.ReleaseE(el.ObjectID)
}

// EvalE ...
func (el *Element) EvalE(byValue bool, js string, params ...interface{}) (kit.JSONResult, error) {
	return el.page.EvalE(byValue, el.ObjectID, js, params)
}
