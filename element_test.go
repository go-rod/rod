package rod_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func (t T) Click() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	t.True(p.MustHas("[a=ok]"))

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	t.Panic(func() {
		t.mc.stubErr(8, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
}

func (t T) ClickWrapped() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click-wrapped.html")).MustWaitLoad()
	el := p.MustElement("#target")

	shape := el.MustShape()
	t.Len(shape.Quads, 2)

	el.MustClick()
	t.True(p.MustHas("[a=ok]"))
}

func (t T) Tap() {
	page := t.newPage()

	page.MustEmulate(devices.IPad).
		MustNavigate(t.srcFile("fixtures/touch.html")).
		MustWaitLoad()
	el := page.MustElement("button")

	el.MustTap()

	t.True(page.MustHas("[tapped=true]"))

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustTap()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustTap()
	})
	t.Panic(func() {
		t.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
		el.MustTap()
	})
	t.Panic(func() {
		t.mc.stubErr(7, proto.RuntimeCallFunctionOn{})
		el.MustTap()
	})
}

func (t T) Interactable() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	t.True(el.MustInteractable())

	t.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
	t.Err(el.Interactable())
}

func (t T) NotInteractable() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	// cover the button with a green div
	p.MustWaitLoad().MustEval(`() => {
		let div = document.createElement('div')
		div.style = 'position: absolute; left: 0; top: 0; width: 500px; height: 500px;'
		document.body.append(div)
	}`)
	_, err := el.Interactable()
	t.Has(err.Error(), "element covered by: <div>")
	t.Is(err, &rod.ErrNotInteractable{})
	t.Is(err, &rod.ErrCovered{})
	t.False(el.MustInteractable())
	var ee *rod.ErrNotInteractable
	t.True(errors.As(err, &ee))
	t.Eq(ee.Error(), "element is not cursor interactable")

	p.MustElement("div").MustRemove()

	t.mc.stubErr(1, proto.DOMGetContentQuads{})
	_, err = el.Interactable()
	t.Err(err)

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	t.Err(el.Interactable())

	t.mc.stubErr(1, proto.DOMDescribeNode{})
	t.Err(el.Interactable())

	t.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
	t.Err(el.Interactable())
}

func (t T) InteractableWithNoShape() {
	p := t.page.MustNavigate(t.srcFile("fixtures/interactable.html"))

	el := p.MustElement("#no-shape")
	_, err := el.Interactable()
	t.Is(err, &rod.ErrInvisibleShape{})
	t.Is(err, &rod.ErrNotInteractable{})
	t.Eq(err.Error(), "element has no visible shape or outside the viewport: <div#no-shape>")

	el = p.MustElement("#outside")
	_, err = el.Interactable()
	t.Is(err, &rod.ErrInvisibleShape{})

	el = p.MustElement("#invisible")
	_, err = el.Interactable()
	t.Is(err, &rod.ErrInvisibleShape{})
}

func (t T) NotInteractableWithNoPointerEvents() {
	p := t.page.MustNavigate(t.srcFile("fixtures/interactable.html"))
	_, err := p.MustElementR("#no-pointer-events", "click me").Interactable()
	t.Is(err, &rod.ErrNoPointerEvents{})
	t.Is(err, &rod.ErrNotInteractable{})
	t.Eq(err.Error(), "element's pointer-events is none: <span#no-pointer-events>")
}

func (t T) WaitInteractable() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	start := time.Now()

	// cover the button with a green div for 1sec
	p.MustWaitLoad().MustEval(`() => {
		let div = document.createElement('div')
		div.style = 'position: absolute; left: 0; top: 0; width: 500px; height: 500px;'
		document.body.append(div)
		setTimeout(() => div.remove(), 1000)
	}`)

	el.MustWaitInteractable()

	t.Gt(time.Since(start), time.Second)

	t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
	t.Err(el.WaitInteractable())
}

func (t T) Hover() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustEval(`this.onmouseenter = () => this.dataset['a'] = 1`)
	el.MustHover()
	t.Eq("1", el.MustEval(`this.dataset['a']`).String())

	t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
	t.Err(el.Hover())

	t.mc.stubErr(1, proto.DOMGetContentQuads{})
	t.Err(el.Hover())

	t.mc.stubErr(3, proto.DOMGetContentQuads{})
	t.Err(el.Hover())

	t.mc.stubErr(1, proto.InputDispatchMouseEvent{})
	t.Err(el.Hover())
}

func (t T) ElementMoveMouseOut() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	btn := p.MustElement("button")
	btn.MustEval(`this.onmouseout = () => this.setAttribute('name', 'mouse moved.')`)
	t.Eq("mouse moved.", *btn.MustHover().MustMoveMouseOut().MustAttribute("name"))

	t.mc.stubErr(1, proto.DOMGetContentQuads{})
	t.Err(btn.MoveMouseOut())
}

func (t T) MouseMoveErr() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	t.mc.stubErr(1, proto.InputDispatchMouseEvent{})
	t.Err(p.Mouse.Move(10, 10, 1))
}

func (t T) ElementContext() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button").Timeout(time.Hour).CancelTimeout()
	el, cancel := el.WithCancel()
	defer cancel()
	el.Sleeper(rod.DefaultSleeper).MustClick()
}

func (t T) Iframes() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click-iframes.html"))

	frame01 := p.MustElement("iframe").MustFrame()

	frame02 := frame01.MustElement("iframe").MustFrame()
	el := frame02.MustElement("button")
	el.MustClick()

	t.Eq(frame01.MustEval(`testIsolation()`).Str(), "ok")
	t.True(frame02.MustHas("[a=ok]"))
}

func (t T) Contains() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	a := p.MustElement("button")

	b := p.MustElementFromNode(a.MustDescribe())
	t.True(a.MustContainsElement(b))

	pt := a.MustShape().OnePointInside()
	el := p.MustElementFromPoint(int(pt.X), int(pt.Y))
	t.True(a.MustContainsElement(el))

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	t.Err(a.ContainsElement(el))
}

func (t T) ShadowDOM() {
	p := t.page.MustNavigate(t.srcFile("fixtures/shadow-dom.html")).MustWaitLoad()
	el := p.MustElement("#container")
	t.Eq("inside", el.MustShadowRoot().MustElement("p").MustText())

	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMDescribeNode{})
		el.MustShadowRoot()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMResolveNode{})
		el.MustShadowRoot()
	})
}

func (t T) Press() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("[type=text]")

	el.MustPress('1', '2', input.Backspace, ' ')
	el.MustPress([]rune("A b")...)

	t.Eq("1 A b", el.MustText())

	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustPress(' ')
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectAllText()
	})
}

func (t T) KeyDown() {
	p := t.page.MustNavigate(t.srcFile("fixtures/keys.html"))
	p.MustElement("body")
	p.Keyboard.MustDown('j')

	t.True(p.MustHas("body[event=key-down-j]"))
}

func (t T) KeyUp() {
	p := t.page.MustNavigate(t.srcFile("fixtures/keys.html"))
	p.MustElement("body")
	p.Keyboard.MustUp('x')

	t.True(p.MustHas("body[event=key-up-x]"))
}

func (t T) Input() {
	text := "雲の上は\nいつも晴れ"

	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))

	{
		el := p.MustElement("[contenteditable=true]").MustInput(text)
		t.Eq(text, el.MustText())
	}

	el := p.MustElement("textarea")
	el.MustInput(text)

	t.Eq(text, el.MustText())
	t.True(p.MustHas("[event=textarea-change]"))

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustText()
	})
	t.Panic(func() {
		t.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
		el.MustInput("")
	})
	t.Panic(func() {
		t.mc.stubErr(5, proto.RuntimeCallFunctionOn{})
		el.MustInput("")
	})
	t.Panic(func() {
		t.mc.stubErr(6, proto.RuntimeCallFunctionOn{})
		el.MustInput("")
	})
	t.Panic(func() {
		t.mc.stubErr(7, proto.RuntimeCallFunctionOn{})
		el.MustInput("")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.InputInsertText{})
		el.MustInput("")
	})
}

func (t T) InputTime() {
	now := time.Now()

	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))

	var el *rod.Element
	{
		el = p.MustElement("[type=date]")
		el.MustInputTime(now)

		t.Eq(el.MustText(), now.Format("2006-01-02"))
		t.True(p.MustHas("[event=input-date-change]"))
	}

	{
		el = p.MustElement("[type=datetime-local]")
		el.MustInputTime(now)

		t.Eq(el.MustText(), now.Format("2006-01-02T15:04"))
		t.True(p.MustHas("[event=input-datetime-local-change]"))
	}

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
	t.Panic(func() {
		t.mc.stubErr(5, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
	t.Panic(func() {
		t.mc.stubErr(6, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
	t.Panic(func() {
		t.mc.stubErr(7, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
}

func (t T) Checkbox() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("[type=checkbox]")
	t.True(el.MustClick().MustProperty("checked").Bool())
}

func (t T) SelectText() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	el.MustInput("test")
	el.MustSelectAllText()
	el.MustInput("test")
	t.Eq("test", el.MustText())

	el.MustSelectText(`es`)
	el.MustInput("__")

	t.Eq("t__t", el.MustText())

	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectText("")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectAllText()
	})

	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustInput("")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.InputInsertText{})
		el.MustInput("")
	})
}

func (t T) Blur() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("#blur").MustInput("test").MustBlur()

	t.Eq("ok", *el.MustAttribute("a"))
}

func (t T) SelectQuery() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("select")
	err := el.Select([]string{`[value="c"]`}, true, rod.SelectorTypeCSSSector)
	t.E(err)

	t.Eq(2, el.MustEval("this.selectedIndex").Int())
}

func (t T) SelectOptions() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("select")
	el.MustSelect("B", "C")
	t.Eq("B,C", el.MustText())
	t.Eq(1, el.MustProperty("selectedIndex").Int())

	// unselect with regex
	err := el.Select([]string{`^B$`}, false, rod.SelectorTypeRegex)
	t.E(err)
	t.Eq("C", el.MustText())

	// unselect with css selector
	err = el.Select([]string{`[value="c"]`}, false, rod.SelectorTypeCSSSector)
	t.E(err)
	t.Eq("", el.MustText())

	// option not found error
	t.Is(el.Select([]string{"not-exists"}, true, rod.SelectorTypeCSSSector), &rod.ErrElementNotFound{})

	{
		t.mc.stubErr(5, proto.RuntimeCallFunctionOn{})
		t.Err(el.Select([]string{"B"}, true, rod.SelectorTypeText))
	}

	{
		t.mc.stubErr(6, proto.RuntimeCallFunctionOn{})
		t.Err(el.Select([]string{"B"}, true, rod.SelectorTypeText))
	}
}

func (t T) Matches() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	t.True(el.MustMatches(`[cols="30"]`))

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustMatches("")
	})
}

func (t T) Attribute() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	cols := el.MustAttribute("cols")
	rows := el.MustAttribute("rows")

	t.Eq("30", *cols)
	t.Eq("10", *rows)

	p = t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el = p.MustElement("button").MustClick()

	t.Eq("ok", *el.MustAttribute("a"))
	t.Nil(el.MustAttribute("b"))

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustAttribute("")
	})
}

func (t T) Property() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	cols := el.MustProperty("cols")
	rows := el.MustProperty("rows")

	t.Eq(float64(30), cols.Num())
	t.Eq(float64(10), rows.Num())

	p = t.page.MustNavigate(t.srcFile("fixtures/open-page.html"))
	el = p.MustElement("a")

	t.Eq("link", el.MustProperty("id").Str())
	t.Eq("_blank", el.MustProperty("target").Str())
	t.True(el.MustProperty("test").Nil())

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustProperty("")
	})
}

func (t T) SetFiles() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement(`[type=file]`)
	el.MustSetFiles(
		slash("fixtures/click.html"),
		slash("fixtures/alert.html"),
	)

	list := el.MustEval("Array.from(this.files).map(f => f.name)").Arr()
	t.Len(list, 2)
	t.Eq("alert.html", list[1].String())
}

func (t T) Enter() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("[type=submit]")
	el.MustPress(input.Enter)

	t.True(p.MustHas("[event=submit]"))
}

func (t T) WaitInvisible() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	h4 := p.MustElement("h4")
	btn := p.MustElement("button")

	t.True(h4.MustVisible())

	h4.MustWaitVisible()

	go func() {
		utils.Sleep(0.03)
		h4.MustEval(`this.remove()`)
		utils.Sleep(0.03)
		btn.MustEval(`this.style.visibility = 'hidden'`)
	}()

	h4.MustWaitInvisible()
	btn.MustWaitInvisible()

	t.False(p.MustHas("h4"))
}

func (t T) WaitEnabled() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	p.MustElement("button").MustWaitEnabled()
}

func (t T) WaitWritable() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	p.MustElement("input").MustWaitWritable()
}

func (t T) WaitStable() {
	p := t.page.MustNavigate(t.srcFile("fixtures/wait-stable.html"))
	el := p.MustElement("button")
	el.MustEval(`this.classList.add("play")`)
	start := time.Now()
	el.MustWaitStable()
	t.Gt(time.Since(start), time.Second)

	ctx := t.Context()
	t.mc.stub(1, proto.DOMGetContentQuads{}, func(send StubSend) (gson.JSON, error) {
		go func() {
			utils.Sleep(0.1)
			ctx.Cancel()
		}()
		return send()
	})
	t.Err(el.Context(ctx).WaitStable(time.Minute))

	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMGetContentQuads{})
		el.MustWaitStable()
	})
	t.Panic(func() {
		t.mc.stubErr(2, proto.DOMGetContentQuads{})
		el.MustWaitStable()
	})
}

func (t T) WaitStableRAP() {
	p := t.page.MustNavigate(t.srcFile("fixtures/wait-stable.html"))
	el := p.MustElement("button")
	el.MustEval(`this.classList.add("play")`)
	start := time.Now()
	t.E(el.WaitStableRAF())
	t.Gt(time.Since(start), time.Second)

	t.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
	t.Err(el.WaitStableRAF())

	t.mc.stubErr(1, proto.DOMGetContentQuads{})
	t.Err(el.WaitStableRAF())
}

func (t T) CanvasToImage() {
	p := t.page.MustNavigate(t.srcFile("fixtures/canvas.html"))
	src, err := png.Decode(bytes.NewBuffer(p.MustElement("#canvas").MustCanvasToImage()))
	t.E(err)
	t.Eq(src.At(50, 50), color.NRGBA{0xFF, 0x00, 0x00, 0xFF})
}

func (t T) ElementWaitLoad() {
	p := t.page.MustNavigate(t.srcFile("fixtures/resource.html"))
	p.MustElement("img").MustWaitLoad()
}

func (t T) Resource() {
	p := t.page.MustNavigate(t.srcFile("fixtures/resource.html"))
	el := p.MustElement("img")
	t.Eq(len(el.MustResource()), 22661)

	t.mc.stub(1, proto.PageGetResourceContent{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(proto.PageGetResourceContentResult{
			Content:       "ok",
			Base64Encoded: false,
		}), nil
	})
	t.Eq([]byte("ok"), el.MustResource())

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustResource()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageGetResourceContent{})
		el.MustResource()
	})
}

func (t T) BackgroundImage() {
	p := t.page.MustNavigate(t.srcFile("fixtures/resource.html"))
	el := p.MustElement("div")
	t.Eq(len(el.MustBackgroundImage()), 22661)

	{
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		t.Err(el.BackgroundImage())
	}
}

func (t T) ElementScreenshot() {
	f := filepath.Join("tmp", "screenshots", t.Srand(16)+".png")
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("h4")

	data := el.MustScreenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	t.E(err)
	t.Eq(200, img.Bounds().Dx())
	t.Eq(30, img.Bounds().Dy())
	t.Nil(os.Stat(f))

	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustScreenshot()
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.PageCaptureScreenshot{})
		el.MustScreenshot()
	})
	t.Panic(func() {
		t.mc.stubErr(3, proto.DOMGetContentQuads{})
		el.MustScreenshot()
	})
}

func (t T) UseReleasedElement() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	btn := p.MustElement("button")
	btn.MustRelease()
	t.Err(btn.Click("left"))

	btn = p.MustElement("button")
	t.E(proto.RuntimeReleaseObject{ObjectID: btn.Object.ObjectID}.Call(p))
	t.Is(btn.Click("left"), cdp.ErrObjNotFound)
}

func (t T) ElementRemove() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	btn := p.MustElement("button")

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	t.Err(btn.Remove())
}

func (t T) ElementMultipleTimes() {
	// To see whether chrome will reuse the remote object ID or not.
	// Seems like it will not.

	page := t.page.MustNavigate(t.srcFile("fixtures/click.html"))

	btn01 := page.MustElement("button")
	btn02 := page.MustElement("button")

	t.Eq(btn01.MustText(), btn02.MustText())
	t.Neq(btn01.Object, btn02.Object)
}

func (t T) FnErr() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	_, err := el.Eval("foo()")
	t.Err(err)
	t.Has(err.Error(), "ReferenceError: foo is not defined")
	var e *rod.ErrEval
	t.True(errors.As(err, &e))
	t.Eq(proto.RuntimeRemoteObjectSubtypeError, e.Exception.Subtype)

	_, err = el.ElementByJS(rod.Eval("foo()"))
	t.Err(err)
	t.Has(err.Error(), "ReferenceError: foo is not defined")
	t.True(errors.Is(err, &rod.ErrEval{}))
}

func (t T) ElementEWithDepth() {
	checkStr := `green tea`
	p := t.page.MustNavigate(t.srcFile("fixtures/describe.html"))

	ulDOMNode, err := p.MustElement(`ul`).Describe(-1, true)
	t.Nil(errors.Unwrap(err))

	data, err := json.Marshal(ulDOMNode)
	t.Nil(errors.Unwrap(err))
	// The depth is -1, should contain checkStr
	t.Has(string(data), checkStr)
}

func (t T) ElementOthers() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("form")
	el.MustFocus()
	el.MustScrollIntoView()
	t.Eq("submit", el.MustElement("[type=submit]").MustText())
	t.Eq("<input type=\"submit\" value=\"submit\">", el.MustElement("[type=submit]").MustHTML())
	el.MustWait(`true`)
	t.Eq("form", el.MustElementByJS(`this`).MustDescribe().LocalName)
	t.Len(el.MustElementsByJS(`[]`), 0)
}

func (t T) ElementEqual() {
	p := t.page.MustNavigate(t.srcFile("fixtures/describe.html"))
	el1 := p.MustElement("body > ul")
	el2 := p.MustElement("html > body > ul")
	t.True(el1.MustEqual(el2))

	el3 := p.MustElement("ul ul")
	t.False(el1.MustEqual(el3))
}

func (t T) ElementFromPointErr() {
	t.mc.stubErr(1, proto.DOMGetNodeForLocation{})
	t.Err(t.page.ElementFromPoint(10, 10))
}

func (t T) ElementFromNodeErr() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElementX("//button/text()")

	t.mc.stubErr(3, proto.RuntimeCallFunctionOn{})
	t.Err(p.ElementFromNode(el.MustDescribe()))
}

func (t T) ElementErrors() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("form")

	ctx := t.Timeout(0)

	_, err := el.Context(ctx).Describe(-1, true)
	t.Err(err)

	_, err = el.Context(ctx).Frame()
	t.Err(err)

	err = el.Context(ctx).Focus()
	t.Err(err)

	err = el.Context(ctx).Press('a')
	t.Err(err)

	err = el.Context(ctx).Input("a")
	t.Err(err)

	err = el.Context(ctx).Select([]string{"a"}, true, rod.SelectorTypeText)
	t.Err(err)

	err = el.Context(ctx).WaitStable(0)
	t.Err(err)

	_, err = el.Context(ctx).Resource()
	t.Err(err)

	err = el.Context(ctx).Input("a")
	t.Err(err)

	err = el.Context(ctx).Input("a")
	t.Err(err)

	_, err = el.Context(ctx).HTML()
	t.Err(err)

	_, err = el.Context(ctx).Visible()
	t.Err(err)

	_, err = el.Context(ctx).CanvasToImage("", 0)
	t.Err(err)

	err = el.Context(ctx).Release()
	t.Err(err)
}
