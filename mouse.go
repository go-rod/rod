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

// Move to the absolute position with specified steps
func (m *Mouse) Move(x, y float64, steps int) error {
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

// Down doc is similar to the method MustDown
func (m *Mouse) Down(button proto.InputMouseButton, clicks int64) error {
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

// Up doc is similar to the method MustUp
func (m *Mouse) Up(button proto.InputMouseButton, clicks int64) error {
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

// Click doc is similar to the method MustClick
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
