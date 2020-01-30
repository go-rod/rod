package rod

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

const defaultMouseButton = "left"

// Mouse represents the mouse on a page, it's always related the main frame
type Mouse struct {
	ctx  context.Context
	page *Page

	x int64
	y int64
}

// Ctx sets the context for later operation
func (m *Mouse) Ctx(ctx context.Context) *Mouse {
	newObj := *m
	newObj.ctx = ctx
	return &newObj
}

// MoveToE ...
func (m *Mouse) MoveToE(x, y int64) error {
	m.x = x
	m.y = y
	_, err := m.page.Call(m.ctx, "Input.dispatchMouseEvent", cdp.Object{
		"type": "mouseMoved",
		"x":    m.x,
		"y":    m.y,
	})
	return err
}

// Move to the location
func (m *Mouse) Move(x, y int64) {
	kit.E(m.MoveToE(x, y))
}

// DownE ...
func (m *Mouse) DownE(button string) error {
	_, err := m.page.Call(m.ctx, "Input.dispatchMouseEvent", cdp.Object{
		"type":       "mousePressed",
		"button":     button,
		"clickCount": 1,
		"x":          m.x,
		"y":          m.y,
	})
	return err
}

// Down button
func (m *Mouse) Down(button string) {
	kit.E(m.DownE(button))
}

// UpE ...
func (m *Mouse) UpE(button string) error {
	_, err := m.page.Call(m.ctx, "Input.dispatchMouseEvent", cdp.Object{
		"type":       "mouseReleased",
		"button":     button,
		"clickCount": 1,
		"x":          m.x,
		"y":          m.y,
	})
	return err
}

// Up button
func (m *Mouse) Up(button string) {
	kit.E(m.UpE(button))
}

// ClickE ...
func (m *Mouse) ClickE(button string) error {
	if button == "" {
		button = defaultMouseButton
	}

	err := m.DownE(button)
	if err != nil {
		return err
	}

	return m.UpE(button)
}

// Click button
func (m *Mouse) Click(button string) {
	kit.E(m.ClickE(button))
}
