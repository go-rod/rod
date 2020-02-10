// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"context"
	"strings"
	"time"

	"github.com/ysmood/kit"
)

// check method and sleep if needed
func (b *Browser) slowmotion(method string) {
	if b.Slowmotion == 0 {
		return
	}

	if strings.HasPrefix(method, "Input.") {
		time.Sleep(b.Slowmotion)
	}
}

// Overlay a rectangle on the main frame with specified message
func (p *Page) Overlay(left, top, width, height int64, msg string) func() {
	const js = `function foo (id, left, top, width, height, msg) {
		var div = document.createElement('div')
		var msgDiv = document.createElement('div')
		div.id = id
		div.style = 'position: fixed; z-index:2147483647; border: 2px dashed red;'
			+ 'border-radius: 3px; box-shadow: #5f3232 0 0 3px; pointer-events: none;'
			+ 'left:' + (left - 2) + 'px;'
			+ 'top:' + (top - 2) + 'px;'
			+ 'height:' + height + 'px;'
			+ 'width:' + width + 'px;'

		if (height === 0) {
			div.style.border = 'none'
		}
	
		msgDiv.style = 'position: absolute; color: #cc26d6; font-size: 12px; background: #ffffffeb;'
			+ 'box-shadow: #333 0 0 3px; padding: 2px 5px; border-radius: 3px; white-space: nowrap;'
			+ 'top:' + (height + 2) + 'px; '
	
		msgDiv.innerHTML = msg
	
		div.appendChild(msgDiv)
		document.body.appendChild(div)
	}`

	root := p.rootFrame()
	id := kit.RandString(8)

	_, err := root.EvalE(true, "", js, []interface{}{
		id,
		left,
		top,
		width,
		height,
		msg,
	})
	if err != nil && err != context.Canceled {
		p.browser.fatal.Publish(err)
	}

	clean := func() {
		_, err := root.EvalE(true, "", `id => {
			let el = document.getElementById(id)
			el && el.remove()
		}`, []interface{}{id})
		if err != nil && err != context.Canceled {
			p.browser.fatal.Publish(err)
		}
	}

	return clean
}

// Trace with an overlay on the element
func (el *Element) Trace(msg string) func() {
	if !el.page.browser.Trace {
		return func() {}
	}

	el.WaitVisible()

	box, _ := el.BoxE()

	return el.page.Overlay(
		box.Get("left").Int(),
		box.Get("top").Int(),
		box.Get("width").Int(),
		box.Get("height").Int(),
		msg,
	)
}
