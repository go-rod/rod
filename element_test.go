package rod_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func TestGetElementPage(t *testing.T) {
	g := setup(t)

	el := g.page.MustNavigate(g.blank()).MustElement("html")
	g.Eq(el.Page().SessionID, g.page.SessionID)
}

func TestClick(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	g.True(p.MustHas("[a=ok]"))

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	g.Panic(func() {
		g.mc.stubErr(8, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
}

func TestClickWrapped(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click-wrapped.html")).MustWaitLoad()
	el := p.MustElement("#target")

	shape := el.MustShape()
	g.Len(shape.Quads, 2)

	el.MustClick()
	g.True(p.MustHas("[a=ok]"))
}

func TestTap(t *testing.T) {
	g := setup(t)

	page := g.newPage()

	page.MustEmulate(devices.IPad).
		MustNavigate(g.srcFile("fixtures/touch.html")).
		MustWaitLoad()
	el := page.MustElement("button")

	el.MustTap()

	g.True(page.MustHas("[tapped=true]"))

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustTap()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustTap()
	})
	g.Panic(func() {
		g.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
		el.MustTap()
	})
	g.Panic(func() {
		g.mc.stubErr(7, proto.RuntimeCallFunctionOn{})
		el.MustTap()
	})
}

func TestInteractable(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	g.True(el.MustInteractable())

	g.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
	g.Err(el.Interactable())
}

func TestNotInteractable(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	// cover the button with a green div
	p.MustWaitLoad().MustEval(`() => {
		let div = document.createElement('div')
		div.style = 'position: absolute; left: 0; top: 0; width: 500px; height: 500px;'
		document.body.append(div)
	}`)
	_, err := el.Interactable()
	g.Has(err.Error(), "element covered by: <div>")
	g.Is(err, &rod.ErrNotInteractable{})
	g.Is(err, &rod.ErrCovered{})
	g.False(el.MustInteractable())
	var ee *rod.ErrNotInteractable
	g.True(errors.As(err, &ee))
	g.Eq(ee.Error(), "element is not cursor interactable")

	p.MustElement("div").MustRemove()

	g.mc.stubErr(1, proto.DOMGetContentQuads{})
	_, err = el.Interactable()
	g.Err(err)

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(el.Interactable())

	g.mc.stubErr(1, proto.DOMDescribeNode{})
	g.Err(el.Interactable())

	g.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
	g.Err(el.Interactable())
}

func TestInteractableWithNoShape(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/interactable.html"))

	el := p.MustElement("#no-shape")
	_, err := el.Interactable()
	g.Is(err, &rod.ErrInvisibleShape{})
	g.Is(err, &rod.ErrNotInteractable{})
	g.Eq(err.Error(), "element has no visible shape or outside the viewport: <div#no-shape>")

	el = p.MustElement("#outside")
	_, err = el.Interactable()
	g.Is(err, &rod.ErrInvisibleShape{})

	el = p.MustElement("#invisible")
	_, err = el.Interactable()
	g.Is(err, &rod.ErrInvisibleShape{})
}

func TestNotInteractableWithNoPointerEvents(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/interactable.html"))
	_, err := p.MustElementR("#no-pointer-events", "click me").Interactable()
	g.Is(err, &rod.ErrNoPointerEvents{})
	g.Is(err, &rod.ErrNotInteractable{})
	g.Eq(err.Error(), "element's pointer-events is none: <span#no-pointer-events>")
}

func TestWaitInteractable(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
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

	g.Gt(time.Since(start), time.Second)

	g.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
	g.Err(el.WaitInteractable())
}

func TestHover(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustEval(`() => this.onmouseenter = () => this.dataset['a'] = 1`)
	el.MustHover()
	g.Eq("1", el.MustEval(`() => this.dataset['a']`).String())

	g.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
	g.Err(el.Hover())

	g.mc.stubErr(1, proto.DOMGetContentQuads{})
	g.Err(el.Hover())

	g.mc.stubErr(3, proto.DOMGetContentQuads{})
	g.Err(el.Hover())

	g.mc.stubErr(1, proto.InputDispatchMouseEvent{})
	g.Err(el.Hover())
}

func TestElementMoveMouseOut(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	btn := p.MustElement("button")
	btn.MustEval(`() => this.onmouseout = () => this.setAttribute('name', 'mouse moved.')`)
	g.Eq("mouse moved.", *btn.MustHover().MustMoveMouseOut().MustAttribute("name"))

	g.mc.stubErr(1, proto.DOMGetContentQuads{})
	g.Err(btn.MoveMouseOut())
}

func TestElementContext(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button").Timeout(time.Hour).CancelTimeout()
	el, cancel := el.WithCancel()
	defer cancel()
	el.Sleeper(rod.DefaultSleeper).MustClick()
}

func TestElementCancelContext(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.Timeout(time.Second).MustElement("button")
	el = el.CancelTimeout()
	utils.Sleep(1.1)
	el.MustClick()
}

func TestIframes(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click-iframes.html"))

	frame01 := p.MustElement("iframe").MustFrame()

	frame02 := frame01.MustElement("iframe").MustFrame()
	el := frame02.MustElement("button")
	el.MustClick()

	g.Eq(frame01.MustEval(`() => testIsolation()`).Str(), "ok")
	g.True(frame02.MustHas("[a=ok]"))
}

func TestContains(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	a := p.MustElement("button")

	b := p.MustElementFromNode(a.MustDescribe())
	g.True(a.MustContainsElement(b))

	pt := a.MustShape().OnePointInside()
	el := p.MustElementFromPoint(int(pt.X), int(pt.Y))
	g.True(a.MustContainsElement(el))

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(a.ContainsElement(el))
}

func TestShadowDOM(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/shadow-dom.html")).MustWaitLoad()
	el := p.MustElement("#container")
	g.Eq("inside", el.MustShadowRoot().MustElement("p").MustText())

	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMDescribeNode{})
		el.MustShadowRoot()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMResolveNode{})
		el.MustShadowRoot()
	})

	elNoShadow := p.MustElement("script")
	_, err := elNoShadow.ShadowRoot()
	g.True((&rod.ErrNoShadowRoot{}).Is(err))
	g.Has(err.Error(), "element has no shadow root:")
}

func TestInputTime(t *testing.T) {
	g := setup(t)

	now := time.Now()

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))

	var el *rod.Element
	{
		el = p.MustElement("[type=date]")
		el.MustInputTime(now)

		g.Eq(el.MustText(), now.Format("2006-01-02"))
		g.True(p.MustHas("[event=input-date-change]"))
	}

	{
		el = p.MustElement("[type=datetime-local]")
		el.MustInputTime(now)

		g.Eq(el.MustText(), now.Format("2006-01-02T15:04"))
		g.True(p.MustHas("[event=input-datetime-local-change]"))
	}

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
	g.Panic(func() {
		g.mc.stubErr(5, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
	g.Panic(func() {
		g.mc.stubErr(6, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
	g.Panic(func() {
		g.mc.stubErr(7, proto.RuntimeCallFunctionOn{})
		el.MustInputTime(now)
	})
}

func TestElementInputDate(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	p.MustElement("[type=date]").MustInput("12")
}

func TestCheckbox(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("[type=checkbox]")
	g.True(el.MustClick().MustProperty("checked").Bool())
}

func TestSelectText(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	el.MustInput("test")
	el.MustSelectAllText()
	el.MustInput("test")
	g.Eq("test", el.MustText())

	el.MustSelectText(`es`)
	el.MustInput("__")

	g.Eq("t__t", el.MustText())

	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectText("")
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustSelectAllText()
	})

	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustInput("")
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.InputInsertText{})
		el.MustInput("")
	})
}

func TestBlur(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("#blur").MustInput("test").MustBlur()

	g.Eq("ok", *el.MustAttribute("a"))
}

func TestSelectQuery(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("select")
	err := el.Select([]string{`[value="c"]`}, true, rod.SelectorTypeCSSSector)
	g.E(err)

	g.Eq(2, el.MustEval("() => this.selectedIndex").Int())
}

func TestSelectOptions(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("select")
	el.MustSelect("B", "C")
	g.Eq("B,C", el.MustText())
	g.Eq(1, el.MustProperty("selectedIndex").Int())

	// unselect with regex
	err := el.Select([]string{`^B$`}, false, rod.SelectorTypeRegex)
	g.E(err)
	g.Eq("C", el.MustText())

	// unselect with css selector
	err = el.Select([]string{`[value="c"]`}, false, rod.SelectorTypeCSSSector)
	g.E(err)
	g.Eq("", el.MustText())

	// option not found error
	g.Is(el.Select([]string{"not-exists"}, true, rod.SelectorTypeCSSSector), &rod.ErrElementNotFound{})

	{
		g.mc.stubErr(5, proto.RuntimeCallFunctionOn{})
		g.Err(el.Select([]string{"B"}, true, rod.SelectorTypeText))
	}
}

func TestMatches(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	g.True(el.MustMatches(`[cols="30"]`))

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustMatches("")
	})
}

func TestAttribute(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	cols := el.MustAttribute("cols")
	rows := el.MustAttribute("rows")

	g.Eq("30", *cols)
	g.Eq("10", *rows)

	p = g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el = p.MustElement("button").MustClick()

	g.Eq("ok", *el.MustAttribute("a"))
	g.Nil(el.MustAttribute("b"))

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustAttribute("")
	})
}

func TestProperty(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	cols := el.MustProperty("cols")
	rows := el.MustProperty("rows")

	g.Eq(float64(30), cols.Num())
	g.Eq(float64(10), rows.Num())

	p = g.page.MustNavigate(g.srcFile("fixtures/open-page.html"))
	el = p.MustElement("a")

	g.Eq("link", el.MustProperty("id").Str())
	g.Eq("_blank", el.MustProperty("target").Str())
	g.True(el.MustProperty("test").Nil())

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustProperty("")
	})
}

func TestDisabled(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))

	g.False(p.MustElement("#EnabledButton").MustDisabled())
	g.True(p.MustElement("#DisabledButton").MustDisabled())

	g.Panic(func() {
		el := p.MustElement("#EnabledButton")
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustDisabled()
	})
}

func TestSetFiles(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement(`[type=file]`)
	el.MustSetFiles(
		slash("fixtures/click.html"),
		slash("fixtures/alert.html"),
	)

	list := el.MustEval("() => Array.from(this.files).map(f => f.name)").Arr()
	g.Len(list, 2)
	g.Eq("alert.html", list[1].String())
}

func TestEnter(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("[type=submit]")
	el.MustType(input.Enter)

	g.True(p.MustHas("[event=submit]"))
}

func TestWaitInvisible(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	h4 := p.MustElement("h4")
	btn := p.MustElement("button")

	g.True(h4.MustVisible())

	h4.MustWaitVisible()

	go func() {
		utils.Sleep(0.03)
		h4.MustEval(`() => this.remove()`)
		utils.Sleep(0.03)
		btn.MustEval(`() => this.style.visibility = 'hidden'`)
	}()

	h4.MustWaitInvisible()
	btn.MustWaitInvisible()

	g.False(p.MustHas("h4"))
}

func TestWaitEnabled(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	p.MustElement("button").MustWaitEnabled()
}

func TestWaitWritable(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	p.MustElement("input").MustWaitWritable()
}

func TestWaitStable(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/wait-stable.html"))
	el := p.MustElement("button")
	go func() {
		utils.Sleep(1)
		el.MustEval(`() => this.classList.remove("play")`)
	}()
	start := time.Now()
	el.MustWaitStable()
	g.Gt(time.Since(start), time.Second)

	ctx := g.Context()
	g.mc.stub(1, proto.DOMGetContentQuads{}, func(send StubSend) (gson.JSON, error) {
		go func() {
			utils.Sleep(0.1)
			ctx.Cancel()
		}()
		return send()
	})
	g.Err(el.Context(ctx).WaitStable(time.Minute))

	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMGetContentQuads{})
		el.MustWaitStable()
	})
	g.Panic(func() {
		g.mc.stubErr(2, proto.DOMGetContentQuads{})
		el.MustWaitStable()
	})
}

func TestWaitStableRAP(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/wait-stable.html"))
	el := p.MustElement("button")
	go func() {
		utils.Sleep(1)
		el.MustEval(`() => this.classList.remove("play")`)
	}()
	start := time.Now()
	g.E(el.WaitStableRAF())
	g.Gt(time.Since(start), time.Second)

	g.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
	g.Err(el.WaitStableRAF())

	g.mc.stubErr(1, proto.DOMGetContentQuads{})
	g.Err(el.WaitStableRAF())
}

func TestCanvasToImage(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/canvas.html"))
	src, err := png.Decode(bytes.NewBuffer(p.MustElement("#canvas").MustCanvasToImage()))
	g.E(err)
	g.Eq(src.At(50, 50), color.NRGBA{0xFF, 0x00, 0x00, 0xFF})
}

func TestElementWaitLoad(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/resource.html"))
	p.MustElement("img").MustWaitLoad()
}

func TestResource(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/resource.html"))
	el := p.MustElement("img")
	g.Eq(len(el.MustResource()), 22661)

	g.mc.stub(1, proto.PageGetResourceContent{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(proto.PageGetResourceContentResult{
			Content:       "ok",
			Base64Encoded: false,
		}), nil
	})
	g.Eq([]byte("ok"), el.MustResource())

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustResource()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.PageGetResourceContent{})
		el.MustResource()
	})
}

func TestBackgroundImage(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/resource.html"))
	el := p.MustElement("div")
	g.Eq(len(el.MustBackgroundImage()), 22661)

	{
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		g.Err(el.BackgroundImage())
	}
}

func TestElementScreenshot(t *testing.T) {
	g := setup(t)

	f := filepath.Join("tmp", "screenshots", g.RandStr(16)+".png")
	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("h4")

	data := el.MustScreenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	g.E(err)
	g.Eq(200, img.Bounds().Dx())
	g.Eq(30, img.Bounds().Dy())
	g.Nil(os.Stat(f))

	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMScrollIntoViewIfNeeded{})
		el.MustScreenshot()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.PageCaptureScreenshot{})
		el.MustScreenshot()
	})
	g.Panic(func() {
		g.mc.stubErr(3, proto.DOMGetContentQuads{})
		el.MustScreenshot()
	})
}

func TestUseReleasedElement(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	btn := p.MustElement("button")
	btn.MustRelease()
	g.Err(btn.Click("left", 1))

	btn = p.MustElement("button")
	g.E(proto.RuntimeReleaseObject{ObjectID: btn.Object.ObjectID}.Call(p))
	g.Is(btn.Click("left", 1), cdp.ErrObjNotFound)
}

func TestElementRemove(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	btn := p.MustElement("button")

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(btn.Remove())
}

func TestElementMultipleTimes(t *testing.T) {
	g := setup(t)

	// To see whether chrome will reuse the remote object ID or not.
	// Seems like it will not.

	page := g.page.MustNavigate(g.srcFile("fixtures/click.html"))

	btn01 := page.MustElement("button")
	btn02 := page.MustElement("button")

	g.Eq(btn01.MustText(), btn02.MustText())
	g.Neq(btn01.Object, btn02.Object)
}

func TestFnErr(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElement("button")

	_, err := el.Eval("foo()")
	g.Err(err)
	g.Has(err.Error(), "ReferenceError: foo is not defined")
	var e *rod.ErrEval
	g.True(errors.As(err, &e))
	g.Eq(proto.RuntimeRemoteObjectSubtypeError, e.Exception.Subtype)

	_, err = el.ElementByJS(rod.Eval("() => foo()"))
	g.Err(err)
	g.Has(err.Error(), "ReferenceError: foo is not defined")
	g.True(errors.Is(err, &rod.ErrEval{}))
}

func TestElementEWithDepth(t *testing.T) {
	g := setup(t)

	checkStr := `green tea`
	p := g.page.MustNavigate(g.srcFile("fixtures/describe.html"))

	ulDOMNode, err := p.MustElement(`ul`).Describe(-1, true)
	g.Nil(errors.Unwrap(err))

	data, err := json.Marshal(ulDOMNode)
	g.Nil(errors.Unwrap(err))
	// The depth is -1, should contain checkStr
	g.Has(string(data), checkStr)
}

func TestElementOthers(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("form")
	el.MustFocus()
	el.MustScrollIntoView()
	g.Eq("submit", el.MustElement("[type=submit]").MustText())
	g.Eq("<input type=\"submit\" value=\"submit\">", el.MustElement("[type=submit]").MustHTML())
	el.MustWait(`() => true`)
	g.Eq("form", el.MustElementByJS(`() => this`).MustDescribe().LocalName)
	g.Len(el.MustElementsByJS(`() => []`), 0)
}

func TestElementEqual(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/describe.html"))
	el1 := p.MustElement("body > ul")
	el2 := p.MustElement("html > body > ul")
	g.True(el1.MustEqual(el2))

	el3 := p.MustElement("ul ul")
	g.False(el1.MustEqual(el3))
}

func TestElementWait(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/describe.html"))
	e1 := p.MustElement("body > ul > li")
	g.Eq(e1.MustText(), "coffee")

	params := []interface{}{1, 3, 4}
	go func() {
		utils.Sleep(0.3)
		e1.MustEval(`(a, b, c) => this.innerText = 'x'.repeat(a + b + c)`, params...)
	}()

	e1.MustWait(`(a, b, c) => this.innerText.length === (a + b + c)`, params...)
	g.Eq(e1.MustText(), "xxxxxxxx")
}

func TestShapeInIframe(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click-iframe.html"))
	pt := p.MustElement("iframe").MustFrame().MustElement("button").MustShape().OnePointInside()

	g.InDelta(pt.X, 238, 1)
	g.InDelta(pt.Y, 287, 1)
}

func TestElementFromPointErr(t *testing.T) {
	g := setup(t)

	g.mc.stubErr(1, proto.DOMGetNodeForLocation{})
	g.Err(g.page.ElementFromPoint(10, 10))
}

func TestElementFromNodeErr(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	el := p.MustElementX("//button/text()")

	g.mc.stubErr(3, proto.RuntimeCallFunctionOn{})
	g.Err(p.ElementFromNode(el.MustDescribe()))
}

func TestElementErrors(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("form")

	ctx := g.Timeout(0)

	_, err := el.Context(ctx).Describe(-1, true)
	g.Err(err)

	_, err = el.Context(ctx).Frame()
	g.Err(err)

	err = el.Context(ctx).Focus()
	g.Err(err)

	_, err = el.Context(ctx).KeyActions()
	g.Err(err)

	err = el.Context(ctx).Input("a")
	g.Err(err)

	err = el.Context(ctx).Select([]string{"a"}, true, rod.SelectorTypeText)
	g.Err(err)

	err = el.Context(ctx).WaitStable(0)
	g.Err(err)

	_, err = el.Context(ctx).Resource()
	g.Err(err)

	err = el.Context(ctx).Input("a")
	g.Err(err)

	err = el.Context(ctx).Input("a")
	g.Err(err)

	_, err = el.Context(ctx).HTML()
	g.Err(err)

	_, err = el.Context(ctx).Visible()
	g.Err(err)

	_, err = el.Context(ctx).CanvasToImage("", 0)
	g.Err(err)

	err = el.Context(ctx).Release()
	g.Err(err)
}

func TestElementGetXPath(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("textarea")
	xpath := el.MustGetXPath(true)
	g.Eq(xpath, "/html/body/form/textarea")

	xpath = el.MustGetXPath(false)
	g.Eq(xpath, "/html/body/form/textarea")

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustGetXPath(true)
	})
}
