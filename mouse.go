package rod

import (
	"sync"

	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/input"
)

const defaultMouseButton = "left"

// Mouse represents the mouse on a page, it's always related the main frame
type Mouse struct {
	page *Page
	sync.Mutex

	x float64
	y float64

	// the buttons is currently beening pressed, reflects the press order
	buttons []string
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

		_, err := m.page.CallE("Input.dispatchMouseEvent", cdp.Object{
			"type":      "mouseMoved",
			"x":         toX,
			"y":         toY,
			"button":    button,
			"buttons":   buttons,
			"modifiers": m.page.Keyboard.modifiers,
		})
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
func (m *Mouse) ScrollE(x, y, steps int64) error {
	if steps < 1 {
		steps = 1
	}

	button, buttons := input.EncodeMouseButton(m.buttons)

	stepX := x / steps
	stepY := y / steps

	for i := int64(0); i < steps; i++ {
		_, err := m.page.CallE("Input.dispatchMouseEvent", cdp.Object{
			"type":      "mouseWheel",
			"x":         m.x,
			"y":         m.y,
			"button":    button,
			"buttons":   buttons,
			"modifiers": m.page.Keyboard.modifiers,
			"deltaX":    stepX,
			"deltaY":    stepY,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// DownE doc is the same as the method Down
func (m *Mouse) DownE(button string, clicks int64) error {
	m.Lock()
	defer m.Unlock()

	toButtons := append(m.buttons, button)

	_, buttons := input.EncodeMouseButton(toButtons)

	_, err := m.page.CallE("Input.dispatchMouseEvent", cdp.Object{
		"type":       "mousePressed",
		"button":     button,
		"buttons":    buttons,
		"clickCount": clicks,
		"modifiers":  m.page.Keyboard.modifiers,
		"x":          m.x,
		"y":          m.y,
	})
	if err != nil {
		return err
	}
	m.buttons = toButtons
	return nil
}

// UpE doc is the same as the method Up
func (m *Mouse) UpE(button string, clicks int64) error {
	m.Lock()
	defer m.Unlock()

	toButtons := []string{}
	for _, btn := range m.buttons {
		if btn == button {
			continue
		}
		toButtons = append(toButtons, btn)
	}

	_, buttons := input.EncodeMouseButton(toButtons)

	_, err := m.page.CallE("Input.dispatchMouseEvent", cdp.Object{
		"type":       "mouseReleased",
		"button":     button,
		"buttons":    buttons,
		"clickCount": clicks,
		"x":          m.x,
		"y":          m.y,
	})
	if err != nil {
		return err
	}
	m.buttons = toButtons
	return nil
}

// ClickE doc is the same as the method Click
func (m *Mouse) ClickE(button string) error {
	if button == "" {
		button = defaultMouseButton
	}

	err := m.DownE(button, 1)
	if err != nil {
		return err
	}

	return m.UpE(button, 1)
}
