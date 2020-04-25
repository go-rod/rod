// This file defines the helpers to develop automation.
// Such as when running automation we can use trace to visually
// see where the mouse going to click.

package rod

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
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
func (p *Page) Overlay(left, top, width, height float64, msg string) (remove func()) {
	root := p.Root()
	id := "rod-" + kit.RandString(8)

	_, err := root.EvalE(true, "", root.jsFn("overlay"), cdp.Array{
		id,
		left,
		top,
		width,
		height,
		msg,
	})
	CancelPanic(err)

	remove = func() {
		_, _ = root.EvalE(true, "", root.jsFn("removeOverlay"), cdp.Array{id})
	}

	return
}

// Trace with an overlay on the element
func (el *Element) Trace(htmlMessage string) (removeOverlay func()) {
	id := "rod-" + kit.RandString(8)

	_, err := el.EvalE(true, el.page.jsFn("elementOverlay"), cdp.Array{
		id,
		htmlMessage,
	})
	CancelPanic(err)

	removeOverlay = func() {
		_, _ = el.EvalE(true, el.page.jsFn("removeOverlay"), cdp.Array{id})
	}

	el.page.Trace(el.page.stripHTML(htmlMessage))

	return
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

	index := make([]byte, 8)
	binary.BigEndian.PutUint64(index, uint64(time.Now().UnixNano()))

	name := kit.Escape(hex.EncodeToString(index) + " " + msg)
	path := filepath.Join(dir, name+".jpg")

	kit.E(kit.OutputFile(path, img, nil))
}

func (p *Page) stripHTML(str string) string {
	return p.Eval(p.jsFn("stripHTML"), str).String()
}

func (p *Page) traceFn(js string, params cdp.Array) func() {
	fnName := strings.Replace(js, p.jsFnPrefix(), "rod.", 1)
	paramsStr := p.stripHTML(kit.MustToJSON(params))
	msg := fmt.Sprintf("retry <code>%s(%s)</code>", fnName, paramsStr[1:len(paramsStr)-1])
	return p.Overlay(0, 0, 500, 0, msg)
}
