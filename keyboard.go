package rod

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/keys"
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
func (k *Keyboard) DownE(key rune) error {
	actions := keys.Encode(key)
	_, err := k.page.Call(k.ctx, "Input.dispatchKeyEvent", actions[0])
	return err
}

// Down holds key down
func (k *Keyboard) Down(key rune) {
	kit.E(k.DownE(key))
}

// UpE ...
func (k *Keyboard) UpE(key rune) error {
	actions := keys.Encode(key)
	_, err := k.page.Call(k.ctx, "Input.dispatchKeyEvent", actions[len(actions)-1])
	return err
}

// Up releases the key
func (k *Keyboard) Up(key rune) {
	kit.E(k.UpE(key))
}

// PressE ...
func (k *Keyboard) PressE(key rune) error {
	actions := keys.Encode(key)

	for _, action := range actions {
		_, err := k.page.Call(k.ctx, "Input.dispatchKeyEvent", action)
		if err != nil {
			return err
		}
	}
	return nil
}

// Press a key
func (k *Keyboard) Press(key rune) {
	kit.E(k.PressE(key))
}

// InsertTextE ...
func (k *Keyboard) InsertTextE(text string) error {
	_, err := k.page.Call(k.ctx, "Input.insertText", cdp.Object{
		"text": text,
	})
	return err
}

// InsertText like paste text into the page
func (k *Keyboard) InsertText(text string) {
	kit.E(k.InsertTextE(text))
}
