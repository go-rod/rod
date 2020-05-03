package rod

import (
	"sync"

	"github.com/ysmood/rod/lib/input"
	"github.com/ysmood/rod/lib/proto"
)

// Mouse represents the mouse on a page, it's always related the main frame
type Mouse struct {
	page *Page
	sync.Mutex

	x float64
	y float64

	// the buttons is currently beening pressed, reflects the press order
	buttons []proto.InputMouseButton
}

// MoveE doc is the same as the method Move
func (m *Mouse) MoveE(x, y float64, steps int) error {
	if steps < 1 {
		steps = 1
	}

	m.Lock()
	defer m.Unlock()

	stepX := (x - m.x) / float64(steps)
	stepY := (y - m.y) / float64(steps)

	button, buttons := input.EncodeMouseButton(m.buttons)

	for i := 0; i < steps; i++ {
		toX := m.x + stepX
		toY := m.y + stepY

		err := proto.InputDispatchMouseEvent{
			Type:      proto.InputDispatchMouseEventTypeMouseMoved,
			X:         toX,
			Y:         toY,
			Button:    button,
			Buttons:   buttons,
			Modifiers: m.page.Keyboard.modifiers,
		}.Call(m.page)
		if err != nil {
			return err
		}

		// to make sure set only when call is successful
		m.x = toX
		m.y = toY
	}

	return nil
}

// ScrollE doc is the same as the method Scroll
func (m *Mouse) ScrollE(x, y float64, steps int) error {
	if steps < 1 {
		steps = 1
	}

	button, buttons := input.EncodeMouseButton(m.buttons)

	stepX := x / float64(steps)
	stepY := y / float64(steps)

	for i := 0; i < steps; i++ {
		err := proto.InputDispatchMouseEvent{
			Type:      proto.InputDispatchMouseEventTypeMouseWheel,
			X:         m.x,
			Y:         m.y,
			Button:    button,
			Buttons:   buttons,
			Modifiers: m.page.Keyboard.modifiers,
			DeltaX:    stepX,
			DeltaY:    stepY,
		}.Call(m.page)
		if err != nil {
			return err
		}
	}

	return nil
}

// DownE doc is the same as the method Down
func (m *Mouse) DownE(button proto.InputMouseButton, clicks int64) error {
	m.Lock()
	defer m.Unlock()

	toButtons := append(m.buttons, button)

	_, buttons := input.EncodeMouseButton(toButtons)

	err := proto.InputDispatchMouseEvent{
		Type:       proto.InputDispatchMouseEventTypeMousePressed,
		Button:     button,
		Buttons:    buttons,
		ClickCount: clicks,
		Modifiers:  m.page.Keyboard.modifiers,
		X:          m.x,
		Y:          m.y,
	}.Call(m.page)
	if err != nil {
		return err
	}
	m.buttons = toButtons
	return nil
}

// UpE doc is the same as the method Up
func (m *Mouse) UpE(button proto.InputMouseButton, clicks int64) error {
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

// ClickE doc is the same as the method Click
func (m *Mouse) ClickE(button proto.InputMouseButton) error {
	err := m.DownE(button, 1)
	if err != nil {
		return err
	}

	return m.UpE(button, 1)
}
