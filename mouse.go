package rod

import (
	"fmt"
	"sync"

	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// Mouse represents the mouse on a page, it's always related the main frame
type Mouse struct {
	lock *sync.Mutex

	page *Page

	id string // mouse svg dom element id

	x float64
	y float64

	// the buttons is currently beening pressed, reflects the press order
	buttons []proto.InputMouseButton
}

// MoveE to the absolute position with specified steps
func (m *Mouse) MoveE(x, y float64, steps int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

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
			Modifiers: m.page.Keyboard.modifiers,
		}.Call(m.page)
		if err != nil {
			return err
		}

		// to make sure set only when call is successful
		m.x = toX
		m.y = toY

		if m.page.browser.trace {
			js, jsArgs := m.page.jsHelper("updateMouseTracer", Array{m.id, m.x, m.y})
			_, err := m.page.EvalE(true, "", js, jsArgs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ScrollE the relative offset with specified steps
func (m *Mouse) ScrollE(offsetX, offsetY float64, steps int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

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

// DownE doc is similar to the method Down
func (m *Mouse) DownE(button proto.InputMouseButton, clicks int64) error {
	m.lock.Lock()
	defer m.lock.Unlock()

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

// UpE doc is similar to the method Up
func (m *Mouse) UpE(button proto.InputMouseButton, clicks int64) error {
	m.lock.Lock()
	defer m.lock.Unlock()

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

// ClickE doc is similar to the method Click
func (m *Mouse) ClickE(button proto.InputMouseButton) error {
	if m.page.browser.trace {
		defer m.page.Overlay(0, 0, 200, 0, "click "+string(button))()
	}
	m.page.browser.trySlowmotion()

	err := m.DownE(button, 1)
	if err != nil {
		return err
	}

	return m.UpE(button, 1)
}
