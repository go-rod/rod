package rod

import (
	"sync"

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

	_, err := k.page.CallE("Input.dispatchKeyEvent", actions[0])
	if err != nil {
		return err
	}
	k.modifiers = actions[0].Modifiers
	return nil
}

// UpE doc is the same as the method Up
func (k *Keyboard) UpE(key rune) error {
	actions := input.Encode(key)

	k.Lock()
	defer k.Unlock()

	_, err := k.page.CallE("Input.dispatchKeyEvent", actions[len(actions)-1])
	if err != nil {
		return err
	}
	k.modifiers = 0
	return nil
}

// PressE doc is the same as the method Press
func (k *Keyboard) PressE(key rune) error {
	actions := input.Encode(key)

	k.Lock()
	defer k.Unlock()

	k.modifiers = actions[0].Modifiers
	defer func() { k.modifiers = 0 }()

	for _, action := range actions {
		_, err := k.page.CallE("Input.dispatchKeyEvent", action)
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertTextE doc is the same as the method InsertText
func (k *Keyboard) InsertTextE(text string) error {
	_, err := k.page.CallE("Input.insertText", cdp.Object{
		"text": text,
	})
	return err
}
