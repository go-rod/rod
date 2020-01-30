package rod

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Keyboard represents the keyboard on a page, it's always related the main frame
type Keyboard struct {
	ctx  context.Context
	page *Page
}

// Ctx sets the context for later operation
func (k *Keyboard) Ctx(ctx context.Context) *Keyboard {
	newObj := *k
	newObj.ctx = ctx
	return &newObj
}

// DownE ...
func (k *Keyboard) DownE(key string) error {
	text := ""
	if len(key) < 2 {
		text = key
	}

	_, err := k.page.Call(k.ctx, "Input.dispatchKeyEvent", cdp.Object{
		"type": "keyDown",
		"key":  key,
		"text": text,
	})
	return err
}

// Down button
func (k *Keyboard) Down(key string) {
	kit.E(k.DownE(key))
}

// UpE ...
func (k *Keyboard) UpE(key string) error {
	text := ""
	if len(key) < 2 {
		text = key
	}

	_, err := k.page.Call(k.ctx, "Input.dispatchKeyEvent", cdp.Object{
		"type": "keyUp",
		"key":  key,
		"text": text,
	})
	return err
}

// Up button
func (k *Keyboard) Up(key string) {
	kit.E(k.UpE(key))
}

// PressE ...
func (k *Keyboard) PressE(key string) error {

	if key == "" {
		key = defaultMouseButton
	}

	err := k.DownE(key)
	if err != nil {
		return err
	}

	return k.UpE(key)
}

// Press button
func (k *Keyboard) Press(key string) {
	kit.E(k.PressE(key))
}

// TextE ...
func (k *Keyboard) TextE(text string) error {
	_, err := k.page.Call(k.ctx, "Input.insertText", cdp.Object{
		"text": text,
	})
	return err
}

// Text inset text
func (k *Keyboard) Text(text string) {
	kit.E(k.TextE(text))
}
