// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// check method and sleep if needed
func (b *Browser) trySlowmotion(method string) {
	if b.slowmotion == 0 {
		return
	}

	if strings.HasPrefix(method, "Input.") {
		time.Sleep(b.slowmotion)
	}
}

// Overlay a rectangle on the main frame with specified message
func (p *Page) Overlay(left, top, width, height float64, msg string) func() {
	root := p.Root()
	id := "rod-" + kit.RandString(8)

	_, err := root.EvalE(true, "", root.jsFn("overlay"), []interface{}{
		id,
		left,
		top,
		width,
		height,
		msg,
	})
	CancelPanic(err)

	clean := func() {
		_, err := root.EvalE(true, "", root.jsFn("removeOverlay"), []interface{}{id})
		CancelPanic(err)
	}

	return clean
}

// Trace with an overlay on the element
func (el *Element) Trace(msg string) func() {
	var removeOverlay func()
	if el.page.browser.trace {
		box, err := el.BoxE()
		CancelPanic(err)
		removeOverlay = el.page.Overlay(
			box.Get("left").Float(),
			box.Get("top").Float(),
			box.Get("width").Float(),
			box.Get("height").Float(),
			msg,
		)
	}

	el.page.Trace(msg)

	return func() {
		if removeOverlay != nil {
			removeOverlay()
		}
	}
}

// Trace screenshot to TraceDir
func (p *Page) Trace(msg string) {
	dir := p.traceDir
	if dir == "" {
		return
	}

	img, err := p.Root().ScreenshotE(cdp.Object{
		"format":  "jpeg",
		"quality": 80,
	})
	CancelPanic(err)

	time := time.Now().Format(time.RFC3339Nano)
	name := kit.Escape(time + " " + msg)
	path := filepath.Join(dir, name+".jpg")

	kit.E(kit.OutputFile(path, img, nil))
}
