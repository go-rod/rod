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
	const js = `function foo (id, left, top, width, height, msg) {
		var div = document.createElement('div')
		var msgDiv = document.createElement('div')
		div.id = id
		div.style = 'position: fixed; z-index:2147483647; border: 2px dashed red;'
			+ 'border-radius: 3px; box-shadow: #5f3232 0 0 3px; pointer-events: none;'
			+ 'box-sizing: border-box;'
			+ 'left:' + left + 'px;'
			+ 'top:' + top + 'px;'
			+ 'height:' + height + 'px;'
			+ 'width:' + width + 'px;'

		if (height === 0) {
			div.style.border = 'none'
		}
	
		msgDiv.style = 'position: absolute; color: #cc26d6; font-size: 12px; background: #ffffffeb;'
			+ 'box-shadow: #333 0 0 3px; padding: 2px 5px; border-radius: 3px; white-space: nowrap;'
			+ 'top:' + height + 'px; '
	
		msgDiv.innerHTML = msg
	
		div.appendChild(msgDiv)
		document.body.appendChild(div)
	}`

	root := p.Root()
	id := "rod-" + kit.RandString(8)

	_, err := root.EvalE(true, "", js, []interface{}{
		id,
		left,
		top,
		width,
		height,
		msg,
	})
	CancelPanic(err)

	clean := func() {
		_, err := root.EvalE(true, "", `id => {
			let el = document.getElementById(id)
			el && el.remove()
		}`, []interface{}{id})
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

	el.page.Trace()

	return func() {
		if removeOverlay != nil {
			removeOverlay()
		}
		el.page.Trace()
	}
}

// Trace screenshot to TraceDir
func (p *Page) Trace() {
	dir := p.traceDir
	if dir == "" {
		return
	}

	img, err := p.Root().ScreenshotE(cdp.Object{
		"format":  "jpeg",
		"quality": 80,
	})
	CancelPanic(err)

	name := strings.ReplaceAll(time.Now().Format(time.RFC3339Nano), ":", "_")
	path := filepath.Join(dir, name+".jpg")

	kit.E(kit.OutputFile(path, img, nil))
}
