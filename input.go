package rod

import (
	"fmt"
	"sync"

	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// Keyboard represents the keyboard on a page, it's always related the main frame
type Keyboard struct {
	sync.Mutex

	page *Page

	// modifiers are currently beening pressed
	modifiers int64
}

func (k *Keyboard) getModifiers() int64 {
	k.Lock()
	defer k.Unlock()

	return k.modifiers
}

// Down holds the key down
func (k *Keyboard) Down(key rune) error {
	k.Lock()
	defer k.Unlock()

	actions := input.Encode(key)

	err := actions[0].Call(k.page)
	if err != nil {
		return err
	}
	k.modifiers = actions[0].Modifiers
	return nil
}

// Up releases the key
func (k *Keyboard) Up(key rune) error {
	k.Lock()
	defer k.Unlock()

	actions := input.Encode(key)

	err := actions[len(actions)-1].Call(k.page)
	if err != nil {
		return err
	}
	k.modifiers = 0
	return nil
}

// Press a key. It's a combination of Keyboard.Down and Keyboard.Up
func (k *Keyboard) Press(key rune) error {
	k.Lock()
	defer k.Unlock()

	if k.page.browser.trace {
		defer k.page.Overlay(0, 0, 200, 0, "press "+input.Keys[key].Key)()
	}
	k.page.browser.trySlowmotion()

	actions := input.Encode(key)

	k.modifiers = actions[0].Modifiers
	defer func() { k.modifiers = 0 }()

	for _, action := range actions {
		err := action.Call(k.page)
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertText is like pasting text into the page
func (k *Keyboard) InsertText(text string) error {
	k.Lock()
	defer k.Unlock()

	if k.page.browser.trace {
		defer k.page.Overlay(0, 0, 200, 0, "insert text "+text)()
	}
	k.page.browser.trySlowmotion()

	err := proto.InputInsertText{Text: text}.Call(k.page)
	return err
}

// Mouse represents the mouse on a page, it's always related the main frame
type Mouse struct {
	sync.Mutex

	page *Page

	id string // mouse svg dom element id

	x float64
	y float64

	// the buttons is currently beening pressed, reflects the press order
	buttons []proto.InputMouseButton
}

// Move to the absolute position with specified steps
func (m *Mouse) Move(x, y float64, steps int) error {
	m.Lock()
	defer m.Unlock()

	if steps < 1 {
		steps = 1
	}

	stepX := (x - m.x) / float64(steps)
	stepY := (y - m.y) / float64(steps)

	button, buttons := input.EncodeMouseButton(m.buttons)

	for i := 0; i < steps; i++ {
		m.page.browser.trySlowmotion()

		toX := m.x + stepX
		toY := m.y + stepY

		err := proto.InputDispatchMouseEvent{
			Type:      proto.InputDispatchMouseEventTypeMouseMoved,
			X:         toX,
			Y:         toY,
			Button:    button,
			Buttons:   buttons,
			Modifiers: m.page.Keyboard.getModifiers(),
		}.Call(m.page)
		if err != nil {
			return err
		}

		// to make sure set only when call is successful
		m.x = toX
		m.y = toY

		if m.page.browser.trace {
			if !m.updateMouseTracer() {
				m.initMouseTracer()
				m.updateMouseTracer()
			}
		}
	}

	return nil
}

// Scroll the relative offset with specified steps
func (m *Mouse) Scroll(offsetX, offsetY float64, steps int) error {
	m.Lock()
	defer m.Unlock()

	if m.page.browser.trace {
		defer m.page.Overlay(0, 0, 200, 0, fmt.Sprintf("scroll (%.2f, %.2f)", offsetX, offsetY))()
	}
	m.page.browser.trySlowmotion()

	if steps < 1 {
		steps = 1
	}

	button, buttons := input.EncodeMouseButton(m.buttons)

	stepX := offsetX / float64(steps)
	stepY := offsetY / float64(steps)

	for i := 0; i < steps; i++ {
		err := proto.InputDispatchMouseEvent{
			Type:      proto.InputDispatchMouseEventTypeMouseWheel,
			X:         m.x,
			Y:         m.y,
			Button:    button,
			Buttons:   buttons,
			Modifiers: m.page.Keyboard.getModifiers(),
			DeltaX:    stepX,
			DeltaY:    stepY,
		}.Call(m.page)
		if err != nil {
			return err
		}
	}

	return nil
}

// Down holds the button down
func (m *Mouse) Down(button proto.InputMouseButton, clicks int64) error {
	m.Lock()
	defer m.Unlock()

	toButtons := append(m.buttons, button)

	_, buttons := input.EncodeMouseButton(toButtons)

	err := proto.InputDispatchMouseEvent{
		Type:       proto.InputDispatchMouseEventTypeMousePressed,
		Button:     button,
		Buttons:    buttons,
		ClickCount: clicks,
		Modifiers:  m.page.Keyboard.getModifiers(),
		X:          m.x,
		Y:          m.y,
	}.Call(m.page)
	if err != nil {
		return err
	}
	m.buttons = toButtons
	return nil
}

// Up releases the button
func (m *Mouse) Up(button proto.InputMouseButton, clicks int64) error {
	m.Lock()
	defer m.Unlock()

	toButtons := []proto.InputMouseButton{}
	for _, btn := range m.buttons {
		if btn == button {
			continue
		}
		toButtons = append(toButtons, btn)
	}

	_, buttons := input.EncodeMouseButton(toButtons)

	err := proto.InputDispatchMouseEvent{
		Type:       proto.InputDispatchMouseEventTypeMouseReleased,
		Button:     button,
		Buttons:    buttons,
		ClickCount: clicks,
		X:          m.x,
		Y:          m.y,
	}.Call(m.page)
	if err != nil {
		return err
	}
	m.buttons = toButtons
	return nil
}

// Click the button. It's the combination of Mouse.Down and Mouse.Up
func (m *Mouse) Click(button proto.InputMouseButton) error {
	if m.page.browser.trace {
		defer m.page.Overlay(0, 0, 200, 0, "click "+string(button))()
	}
	m.page.browser.trySlowmotion()

	err := m.Down(button, 1)
	if err != nil {
		return err
	}

	return m.Up(button, 1)
}

// Touch presents a touch device, such as a hand with fingers, each finger is a proto.InputTouchPoint.
// Touch events is stateless, we use the struct here only as a namespace to make the API style unified.
type Touch struct {
	page *Page
}

// Start a touch action
func (t *Touch) Start(points ...*proto.InputTouchPoint) error {
	return proto.InputDispatchTouchEvent{
		Type:        proto.InputDispatchTouchEventTypeTouchStart,
		TouchPoints: points,
		Modifiers:   t.page.Keyboard.getModifiers(),
	}.Call(t.page)
}

// Move touch points. Use the InputTouchPoint.ID (Touch.identifier) to track points.
// Doc: https://developer.mozilla.org/en-US/docs/Web/API/Touch_events
func (t *Touch) Move(points ...*proto.InputTouchPoint) error {
	return proto.InputDispatchTouchEvent{
		Type:        proto.InputDispatchTouchEventTypeTouchMove,
		TouchPoints: points,
		Modifiers:   t.page.Keyboard.getModifiers(),
	}.Call(t.page)
}

// End touch action
func (t *Touch) End() error {
	return proto.InputDispatchTouchEvent{
		Type:        proto.InputDispatchTouchEventTypeTouchEnd,
		TouchPoints: []*proto.InputTouchPoint{},
		Modifiers:   t.page.Keyboard.getModifiers(),
	}.Call(t.page)
}

// Cancel touch action
func (t *Touch) Cancel() error {
	return proto.InputDispatchTouchEvent{
		Type:        proto.InputDispatchTouchEventTypeTouchCancel,
		TouchPoints: []*proto.InputTouchPoint{},
		Modifiers:   t.page.Keyboard.getModifiers(),
	}.Call(t.page)
}

// Tap dispatches a touchstart and touchend event.
func (t *Touch) Tap(x, y float64) error {
	if t.page.browser.trace {
		defer t.page.Overlay(0, 0, 200, 0, "touch")()
	}
	t.page.browser.trySlowmotion()

	p := &proto.InputTouchPoint{X: x, Y: y}

	err := t.Start(p)
	if err != nil {
		return err
	}

	return t.End()
}
