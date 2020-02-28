package rod

import (
	"sync"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/input"
)

// Keyboard represents the keyboard on a page, it's always related the main frame
type Keyboard struct {
	page *Page
	sync.Mutex

	// modifiers are currently beening pressed
	modifiers int64
}

// DownE doc is the same as the method Down
func (k *Keyboard) DownE(key rune) error {
	actions := input.Encode(key)

	k.Lock()
	defer k.Unlock()

	_, err := k.page.CallE(nil, "Input.dispatchKeyEvent", actions[0])
	if err != nil {
		return err
	}
	k.modifiers = actions[0].Modifiers
	return nil
}

// Down holds key down
func (k *Keyboard) Down(key rune) {
	kit.E(k.DownE(key))
}

// UpE doc is the same as the method Up
func (k *Keyboard) UpE(key rune) error {
	actions := input.Encode(key)

	k.Lock()
	defer k.Unlock()

	_, err := k.page.CallE(nil, "Input.dispatchKeyEvent", actions[len(actions)-1])
	if err != nil {
		return err
	}
	k.modifiers = 0
	return nil
}

// Up releases the key
func (k *Keyboard) Up(key rune) {
	kit.E(k.UpE(key))
}

// PressE doc is the same as the method Press
func (k *Keyboard) PressE(key rune) error {
	actions := input.Encode(key)

	k.Lock()
	defer k.Unlock()

	k.modifiers = actions[0].Modifiers
	defer func() { k.modifiers = 0 }()

	for _, action := range actions {
		_, err := k.page.CallE(nil, "Input.dispatchKeyEvent", action)
		if err != nil {
			return err
		}
	}
	return nil
}

// Press a key
func (k *Keyboard) Press(key rune) {
	if k.page.browser.trace {
		defer k.page.Overlay(0, 0, 200, 0, "press "+input.Keys[key].Key)()
	}

	kit.E(k.PressE(key))
}

// InsertTextE doc is the same as the method InsertText
func (k *Keyboard) InsertTextE(text string) error {
	_, err := k.page.CallE(nil, "Input.insertText", cdp.Object{
		"text": text,
	})
	return err
}

// InsertText like paste text into the page
func (k *Keyboard) InsertText(text string) {
	kit.E(k.InsertTextE(text))
}
