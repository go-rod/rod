package rod_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/tidwall/gjson"
)

func (c C) Click() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	c.True(p.MustHas("[a=ok]"))

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
}

func (c C) ClickWrapped() {
	p := c.page.MustNavigate(srcFile("fixtures/click-wrapped.html")).MustWaitLoad()
	el := p.MustElement("#target")

	shape := el.MustShape()
	c.Len(shape.Quads, 2)

	el.MustClick()
	c.True(p.MustHas("[a=ok]"))
}

func (c C) Tap() {
	page := c.browser.MustPage("")
	defer page.MustClose()

	page.MustEmulate(devices.IPad).
		MustNavigate(srcFile("fixtures/touch.html")).
		MustWaitLoad()
	el := page.MustElement("button")

	c.browser.Trace(true)
	el.MustTap()
	c.browser.Trace(false)

	c.True(page.MustHas("[tapped=true]"))

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustTap()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustTap()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMGetContentQuads{})
		el.MustTap()
	})
}

func (c C) Interactable() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	c.True(p.MustElement("button").MustInteractable())
}

func (c C) NotInteractable() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	// cover the button with a green div
	p.MustWaitLoad().MustEval(`() => {
		let div = document.createElement('div')
		div.style = 'position: absolute; left: 0; top: 0; width: 500px; height: 500px;'
		document.body.append(div)
	}`)
	ok, _ := el.Interactable()
	c.Is(ok, rod.ErrNotInteractable)
	c.False(el.MustInteractable())
	p.MustElement("div").MustRemove()

	c.mc.stubErr(1, proto.DOMGetContentQuads{})
	_, err := el.Interactable()
	c.Err(err)

	c.mc.stub(1, proto.DOMGetContentQuads{}, func(send StubSend) (proto.JSON, error) {
		res, _ := send()
		return res.Set("quads", nil)
	})
	_, err = el.Interactable()
	c.Err(err)

	c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	c.Err(el.Interactable())

	c.mc.stubErr(1, proto.DOMDescribeNode{})
	c.Err(el.Interactable())

	c.mc.stubErr(3, proto.RuntimeCallFunctionOn{})
	c.Err(el.Interactable())
}

func (c C) Hover() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustEval(`this.onmouseenter = () => this.dataset['a'] = 1`)
	el.MustHover()
	c.Eq("1", el.MustEval(`this.dataset['a']`).String())

	c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
	c.Err(el.Hover())

	c.mc.stubErr(1, proto.DOMGetContentQuads{})
	c.Err(el.Hover())

	c.mc.stubErr(1, proto.InputDispatchMouseEvent{})
	c.Err(el.Hover())
}

func (c C) MouseMoveErr() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	c.mc.stubErr(1, proto.InputDispatchMouseEvent{})
	c.Err(p.Mouse.Move(10, 10, 1))
}

func (c C) ElementContext() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("button").Timeout(time.Hour).CancelTimeout()
	el, cancel := el.WithCancel()
	defer cancel()
	el.Sleeper(rod.DefaultSleeper).MustClick()
}

func (c C) Iframes() {
	p := c.page.MustNavigate(srcFile("fixtures/click-iframes.html"))
	frame := p.MustElement("iframe").MustFrame().MustElement("iframe").MustFrame()
	el := frame.MustElement("button")
	el.MustClick()
	c.True(frame.MustHas("[a=ok]"))

	id := el.MustNodeID()
	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		p.MustElementFromNode(id)
	})

	c.Panic(func() {
		c.mc.stub(1, proto.RuntimeGetProperties{}, func(send StubSend) (proto.JSON, error) {
			d, _ := send()
			return d.Set("result", []interface{}{})
		})
		p.MustElementFromNode(id).MustText()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMDescribeNode{})
		p.MustElementFromNode(id)
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeEvaluate{})
		p.MustElementFromNode(id)
	})
	c.Panic(func() {
		c.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
		p.MustElementFromNode(id)
	})
	c.Panic(func() {
		c.mc.stubErr(4, proto.RuntimeEvaluate{})
		p.MustElementFromNode(id)
	})
}

func (c C) Contains() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	a := p.MustElement("button")

	b := p.MustElementFromNode(a.MustNodeID())
	c.True(a.MustContainsElement(b))

	pt := a.MustShape().OnePointInside()
	el := p.MustElementFromPoint(int(pt.X), int(pt.Y))
	c.True(a.MustContainsElement(el))
}

func (c C) ShadowDOM() {
	p := c.page.MustNavigate(srcFile("fixtures/shadow-dom.html")).MustWaitLoad()
	el := p.MustElement("#container")
	c.Eq("inside", el.MustShadowRoot().MustElement("p").MustText())

	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMDescribeNode{})
		el.MustShadowRoot()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMResolveNode{})
		el.MustShadowRoot()
	})
}

func (c C) Press() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("[type=text]")
	el.MustPress('A')
	el.MustPress(' ')
	el.MustPress('b')

	c.Eq("A b", el.MustText())

	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustPress(' ')
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectAllText()
	})
}

func (c C) KeyDown() {
	p := c.page.MustNavigate(srcFile("fixtures/keys.html"))
	p.MustElement("body")
	p.Keyboard.MustDown('j')

	c.True(p.MustHas("body[event=key-down-j]"))
}

func (c C) KeyUp() {
	p := c.page.MustNavigate(srcFile("fixtures/keys.html"))
	p.MustElement("body")
	p.Keyboard.MustUp('x')

	c.True(p.MustHas("body[event=key-up-x]"))
}

func (c C) Text() {
	text := "雲の上は\nいつも晴れ"

	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	el.MustInput(text)

	c.Eq(text, el.MustText())
	c.True(p.MustHas("[event=textarea-change]"))

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustText()
	})
}

func (c C) Checkbox() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("[type=checkbox]")
	c.True(el.MustClick().MustProperty("checked").Bool())
}

func (c C) SelectText() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	el.MustInput("test")
	el.MustSelectAllText()
	el.MustInput("test")
	c.Eq("test", el.MustText())

	el.MustSelectText(`es`)
	el.MustInput("__")

	c.Eq("t__t", el.MustText())

	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectText("")
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectAllText()
	})

	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustInput("")
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.InputInsertText{})
		el.MustInput("")
	})
}

func (c C) Blur() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("#blur").MustInput("test").MustBlur()

	c.Eq("ok", *el.MustAttribute("a"))
}

func (c C) SelectQuery() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("select")
	err := el.Select([]string{`[value="c"]`}, true, rod.SelectorTypeCSSSector)
	c.E(err)

	c.Eq(2, el.MustEval("this.selectedIndex").Int())
}

func (c C) SelectQueryNum() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("select")
	el.MustSelect("123")

	c.Eq(-1, el.MustEval("this.selectedIndex").Int())
}

func (c C) SelectOptions() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("select")
	el.MustSelect("B", "C")
	c.Eq("B,C", el.MustText())
	c.Eq(1, el.MustProperty("selectedIndex").Int())

	// unselect with regex
	err := el.Select([]string{`^B$`}, false, rod.SelectorTypeRegex)
	c.E(err)
	c.Eq("C", el.MustText())

	// unselect with css selector
	err = el.Select([]string{`[value="c"]`}, false, rod.SelectorTypeCSSSector)
	c.E(err)
	c.Eq("", el.MustText())
}

func (c C) Matches() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	c.True(el.MustMatches(`[cols="30"]`))

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustMatches("")
	})
}

func (c C) Attribute() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	cols := el.MustAttribute("cols")
	rows := el.MustAttribute("rows")

	c.Eq("30", *cols)
	c.Eq("10", *rows)

	p = c.page.MustNavigate(srcFile("fixtures/click.html"))
	el = p.MustElement("button").MustClick()

	c.Eq("ok", *el.MustAttribute("a"))
	c.Nil(el.MustAttribute("b"))

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustAttribute("")
	})
}

func (c C) Property() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	cols := el.MustProperty("cols")
	rows := el.MustProperty("rows")

	c.Eq(float64(30), cols.Num)
	c.Eq(float64(10), rows.Num)

	p = c.page.MustNavigate(srcFile("fixtures/open-page.html"))
	el = p.MustElement("a")

	c.Eq("link", el.MustProperty("id").Str)
	c.Eq("_blank", el.MustProperty("target").Str)
	c.Eq(gjson.Null, el.MustProperty("test").Type)

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustProperty("")
	})
}

func (c C) SetFiles() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement(`[type=file]`)
	el.MustSetFiles(
		slash("fixtures/click.html"),
		slash("fixtures/alert.html"),
	)

	list := el.MustEval("Array.from(this.files).map(f => f.name)").Array()
	c.Len(list, 2)
	c.Eq("alert.html", list[1].String())
}

func (c C) Enter() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("[type=submit]")
	el.MustPress(input.Enter)

	c.True(p.MustHas("[event=submit]"))
}

func (c C) WaitInvisible() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	h4 := p.MustElement("h4")
	btn := p.MustElement("button")
	timeout := 3 * time.Second

	c.True(h4.MustVisible())

	h4t := h4.Timeout(timeout)
	h4t.MustWaitVisible()
	h4t.CancelTimeout()

	go func() {
		utils.Sleep(0.03)
		h4.MustEval(`this.remove()`)
		utils.Sleep(0.03)
		btn.MustEval(`this.style.visibility = 'hidden'`)
	}()

	h4.Timeout(timeout).MustWaitInvisible()
	btn.Timeout(timeout).MustWaitInvisible()

	c.False(p.MustHas("h4"))
}

func (c C) WaitStable() {
	p := c.page.MustNavigate(srcFile("fixtures/wait-stable.html"))
	el := p.MustElement("button")
	start := time.Now()
	el.MustWaitStable().MustClick()
	c.Gt(time.Since(start), time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	c.mc.stub(1, proto.DOMGetContentQuads{}, func(send StubSend) (proto.JSON, error) {
		go func() {
			utils.Sleep(0.1)
			cancel()
		}()
		return send()
	})
	c.Err(el.Context(ctx).WaitStable(time.Minute))

	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMGetContentQuads{})
		el.MustWaitStable()
	})
	c.Panic(func() {
		c.mc.stubErr(2, proto.DOMGetContentQuads{})
		el.MustWaitStable()
	})
}

func (c C) CanvasToImage() {
	p := c.page.MustNavigate(srcFile("fixtures/canvas.html"))
	src, err := png.Decode(bytes.NewBuffer(p.MustElement("#canvas").MustCanvasToImage()))
	c.E(err)
	c.Eq(src.At(50, 50), color.NRGBA{0xFF, 0x00, 0x00, 0xFF})
}

func (c C) Resource() {
	p := c.page.MustNavigate(srcFile("fixtures/resource.html"))
	el := p.MustElement("img").MustWaitLoad()
	c.Eq(15456, len(el.MustResource()))

	c.mc.stub(1, proto.PageGetResourceContent{}, func(send StubSend) (proto.JSON, error) {
		return proto.NewJSON(proto.PageGetResourceContentResult{
			Content:       "ok",
			Base64Encoded: false,
		}), nil
	})
	c.Eq([]byte("ok"), el.MustResource())

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustResource()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.PageGetResourceContent{})
		el.MustResource()
	})
}

func (c C) ElementScreenshot() {
	f := filepath.Join("tmp", "screenshots", utils.RandString(8)+".png")
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("h4")

	data := el.MustScreenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	c.E(err)
	c.Eq(200, img.Bounds().Dx())
	c.Eq(30, img.Bounds().Dy())
	c.Nil(os.Stat(f))

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustScreenshot()
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustScreenshot()
	})
	c.Panic(func() {
		c.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
		el.MustScreenshot()
	})
}

func (c C) UseReleasedElement() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	btn := p.MustElement("button")
	btn.MustRelease()
	c.Err(btn.Click("left"))

	btn = p.MustElement("button")
	c.E(proto.RuntimeReleaseObject{ObjectID: btn.Object.ObjectID}.Call(p))
	c.Eq(btn.Click("left").Error(), "{\"code\":-32000,\"message\":\"Could not find object with given id\",\"data\":\"\"}")
}

func (c C) ElementRemove() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	btn := p.MustElement("button")

	c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	c.Err(btn.Remove())
}

func (c C) ElementMultipleTimes() {
	// To see whether chrome will reuse the remote object ID or not.
	// Seems like it will not.

	page := c.page.MustNavigate(srcFile("fixtures/click.html"))

	btn01 := page.MustElement("button")
	btn02 := page.MustElement("button")

	c.Eq(btn01.MustText(), btn02.MustText())
	c.Neq(btn01.Object, btn02.Object)
}

func (c C) FnErr() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	_, err := el.Eval("foo()")
	c.Err(err)
	c.Has(err.Error(), "ReferenceError: foo is not defined")
	c.True(errors.Is(err, rod.ErrEval))
	c.Eq(proto.RuntimeRemoteObjectSubtypeError, rod.AsError(err).Details.(*proto.RuntimeRemoteObject).Subtype)

	_, err = el.ElementByJS(rod.NewEval("foo()"))
	c.Err(err)
	c.Has(err.Error(), "ReferenceError: foo is not defined")
	c.True(errors.Is(err, rod.ErrEval))
}

func (c C) ElementEWithDepth() {
	checkStr := `green tea`
	p := c.page.MustNavigate(srcFile("fixtures/describe.html"))

	ulDOMNode, err := p.MustElement(`ul`).Describe(-1, true)
	c.Nil(errors.Unwrap(err))

	data, err := json.Marshal(ulDOMNode)
	c.Nil(errors.Unwrap(err))
	// The depth is -1, should contain checkStr
	c.Has(string(data), checkStr)
}

func (c C) ElementOthers() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("form")
	el.MustFocus()
	el.MustScrollIntoView()
	c.Eq("submit", el.MustElement("[type=submit]").MustText())
	c.Eq("<input type=\"submit\" value=\"submit\">", el.MustElement("[type=submit]").MustHTML())
	el.MustWait(`true`)
	c.Eq("form", el.MustElementByJS(`this`).MustDescribe().LocalName)
	c.Len(el.MustElementsByJS(`[]`), 0)
}

func (c C) ElementFromPointErr() {
	c.mc.stubErr(1, proto.DOMGetNodeForLocation{})
	c.Err(c.page.ElementFromPoint(10, 10))
}

func (c C) ElementErrors() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("form")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := el.Context(ctx).Describe(-1, true)
	c.Err(err)

	_, err = el.Context(ctx).Frame()
	c.Err(err)

	err = el.Context(ctx).Focus()
	c.Err(err)

	err = el.Context(ctx).Press('a')
	c.Err(err)

	err = el.Context(ctx).Input("a")
	c.Err(err)

	err = el.Context(ctx).Select([]string{"a"}, true, rod.SelectorTypeText)
	c.Err(err)

	err = el.Context(ctx).WaitStable(0)
	c.Err(err)

	_, err = el.Context(ctx).Resource()
	c.Err(err)

	err = el.Context(ctx).Input("a")
	c.Err(err)

	err = el.Context(ctx).Input("a")
	c.Err(err)

	_, err = el.Context(ctx).HTML()
	c.Err(err)

	_, err = el.Context(ctx).Visible()
	c.Err(err)

	_, err = el.Context(ctx).CanvasToImage("", 0)
	c.Err(err)

	err = el.Context(ctx).Release()
	c.Err(err)

	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMRequestNode{})
		el.MustNodeID()
	})
}
