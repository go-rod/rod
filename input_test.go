package rod_test

import (
	"testing"

	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

func TestKeyActions(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/keys.html"))
	body := p.MustElement("body")

	p.KeyActions().Press(input.ControlLeft).Type(input.Enter).MustDo()
	g.Eq(body.MustText(), `↓ "Control" ControlLeft 17 modifiers(ctrl)
↓ "Enter" Enter 13 modifiers(ctrl)
↑ "Enter" Enter 13 modifiers(ctrl)
↑ "Control" ControlLeft 17 modifiers()
`)

	body.MustEval("() => this.innerText = ''")
	body.MustKeyActions().
		Press(input.ShiftLeft).Type('A', 'X').Release(input.ShiftLeft).
		Type('a').MustDo()
	g.Eq(body.MustText(), `↓ "Shift" ShiftLeft 16 modifiers(shift)
↓ "A" KeyA 65 modifiers(shift)
↑ "A" KeyA 65 modifiers(shift)
↓ "X" KeyX 88 modifiers(shift)
↑ "X" KeyX 88 modifiers(shift)
↑ "Shift" ShiftLeft 16 modifiers()
↓ "a" KeyA 65 modifiers()
↑ "a" KeyA 65 modifiers()
`)

	g.Nil(p.Keyboard.Release('a'))
}

func TestKeyType(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("[type=text]")

	el.MustKeyActions().Type('1', '2', input.Backspace, ' ').MustDo()
	el.MustKeyActions().Type('A', ' ', 'b').MustDo()
	p.MustInsertText(" test")
	p.Keyboard.MustType(input.Tab)

	g.Eq("1 A b test", el.MustText())
}

func TestKeyTypeErr(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/keys.html"))
	body := p.MustElement("body")

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(body.Type('a'))

	g.mc.stubErr(1, proto.InputDispatchKeyEvent{})
	g.Err(p.Keyboard.Type('a'))

	g.mc.stubErr(2, proto.InputDispatchKeyEvent{})
	g.Err(p.Keyboard.Type('a'))

	g.mc.stubErr(1, proto.InputDispatchKeyEvent{})
	g.Err(p.KeyActions().Press('a').Do())
}

func TestInput(t *testing.T) {
	g := setup(t)

	text := "雲の上は\nいつも晴れ"

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))

	{
		el := p.MustElement("[contenteditable=true]").MustInput(text)
		g.Eq(text, el.MustText())
	}

	el := p.MustElement("textarea")
	el.MustInput(text)

	g.Eq(text, el.MustText())
	g.True(p.MustHas("[event=textarea-change]"))

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustText()
	})
	g.Panic(func() {
		g.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
		el.MustInput("")
	})
	g.Panic(func() {
		g.mc.stubErr(5, proto.RuntimeCallFunctionOn{})
		el.MustInput("")
	})
	g.Panic(func() {
		g.mc.stubErr(6, proto.RuntimeCallFunctionOn{})
		el.MustInput("")
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.InputInsertText{})
		el.MustInput("")
	})
}

func TestMouse(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse

	mouse.MustScroll(0, 10)
	mouse.MustMoveTo(140, 160)
	mouse.MustDown("left")
	mouse.MustUp("left")

	g.True(page.MustHas("[a=ok]"))

	g.Panic(func() {
		g.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustScroll(0, 10)
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustDown(proto.InputMouseButtonLeft)
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustUp(proto.InputMouseButtonLeft)
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.InputDispatchMouseEvent{})
		mouse.MustClick(proto.InputMouseButtonLeft)
	})
}

func TestMouseHoldMultiple(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.blank())

	p.Mouse.MustDown("left")
	defer p.Mouse.MustUp("left")
	p.Mouse.MustDown("right")
	defer p.Mouse.MustUp("right")
}

func TestMouseClick(t *testing.T) {
	g := setup(t)

	g.browser.SlowMotion(1)
	defer func() { g.browser.SlowMotion(0) }()

	page := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	page.MustElement("button")
	mouse := page.Mouse
	mouse.MustMoveTo(140, 160)
	mouse.MustClick("left")
	g.True(page.MustHas("[a=ok]"))
}

func TestMouseDoubleClick(t *testing.T) {
	g := setup(t)

	g.browser.SlowMotion(1)
	defer func() { g.browser.SlowMotion(0) }()

	page := g.page.MustNavigate(g.srcFile("fixtures/double-click.html"))
	el := page.MustElement("button")
	el.MustDoubleClick()
	g.Eq(el.MustText(), "ok")
}

func TestMouseDrag(t *testing.T) {
	g := setup(t)

	page := g.newPage().MustNavigate(g.srcFile("fixtures/drag.html")).MustWaitLoad()
	mouse := page.Mouse

	mouse.MustMoveTo(3, 3)
	mouse.MustDown("left")
	g.E(mouse.MoveLinear(proto.NewPoint(60, 80), 3))
	mouse.MustUp("left")

	utils.Sleep(0.3)
	g.Eq(page.MustEval(`() => dragTrack`).Str(), " move 3 3 down 3 3 move 22 28 move 41 54 move 60 80 up 60 80")
}

func TestMouseScroll(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/scroll.html")).MustWaitLoad()

	p.Mouse.MustMoveTo(30, 30)
	p.Mouse.MustClick(proto.InputMouseButtonLeft)

	p.Mouse.MustScroll(0, 10)
	p.Mouse.MustScroll(100, 190)
	g.E(p.Mouse.Scroll(200, 300, 5))

	p.MustWait(`() => pageXOffset > 200 && pageYOffset > 300`)
}

func TestMouseMoveLinear(t *testing.T) {
	g := setup(t)

	page := g.newPage().MustNavigate(g.srcFile("fixtures/mouse-move.html")).MustWaitLoad()
	mouse := page.Mouse

	mouse.MustMoveTo(1, 2)
	g.E(mouse.MoveLinear(proto.NewPoint(3, 4), 3))

	utils.Sleep(0.3)
	g.Eq(page.MustEval(`() => moveTrack`).Str(), " move 1 2 move 1 2 move 2 3 move 3 4")
}

func TestMouseMoveErr(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	g.mc.stubErr(2, proto.InputDispatchMouseEvent{})
	g.Err(p.Mouse.MoveLinear(proto.NewPoint(10, 10), 3))
}

func TestNativeDrag(t *testing.T) { // devtools doesn't support to use mouse event to simulate it for now
	t.Skip()

	g := setup(t)
	page := g.page.MustNavigate(g.srcFile("fixtures/drag.html"))
	mouse := page.Mouse

	pt := page.MustElement("#draggable").MustShape().OnePointInside()
	toY := page.MustElement(".dropzone:nth-child(2)").MustShape().OnePointInside().Y

	page.Overlay(pt.X, pt.Y, 10, 10, "from")
	page.Overlay(pt.X, toY, 10, 10, "to")

	mouse.MustMoveTo(pt.X, pt.Y)
	mouse.MustDown("left")
	g.E(mouse.MoveLinear(proto.NewPoint(pt.X, toY), 5))
	page.MustScreenshot("")
	mouse.MustUp("left")

	page.MustElement(".dropzone:nth-child(2) #draggable")
}

func TestTouch(t *testing.T) {
	g := setup(t)

	page := g.newPage().MustEmulate(devices.IPad)

	wait := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	page.MustNavigate(g.srcFile("fixtures/touch.html"))
	wait()

	touch := page.Touch

	touch.MustTap(10, 20)

	p := &proto.InputTouchPoint{X: 30, Y: 40}

	touch.MustStart(p).MustEnd()
	touch.MustStart(p)
	p.MoveTo(50, 60)
	touch.MustMove(p).MustCancel()

	page.MustWait(`() => touchTrack == ' start 10 20 end start 30 40 end start 30 40 move 50 60 cancel'`)

	g.Panic(func() {
		g.mc.stubErr(1, proto.InputDispatchTouchEvent{})
		touch.MustTap(1, 2)
	})
}
