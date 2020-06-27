package rod

import (
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// Keyboard represents the keyboard on a page, it's always related the main frame
type Keyboard struct {
	page *Page

	// modifiers are currently beening pressed
	modifiers int64
}

// DownE doc is similar to the method Down
func (k *Keyboard) DownE(key rune) error {
	actions := input.Encode(key)

	err := actions[0].Call(k.page)
	if err != nil {
		return err
	}
	k.modifiers = actions[0].Modifiers
	return nil
}

// UpE doc is similar to the method Up
func (k *Keyboard) UpE(key rune) error {
	actions := input.Encode(key)

	err := actions[len(actions)-1].Call(k.page)
	if err != nil {
		return err
	}
	k.modifiers = 0
	return nil
}

// PressE doc is similar to the method Press
func (k *Keyboard) PressE(key rune) error {
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

// InsertTextE doc is similar to the method InsertText
func (k *Keyboard) InsertTextE(text string) error {
	if k.page.browser.trace {
		defer k.page.Overlay(0, 0, 200, 0, "insert text "+text)()
	}
	k.page.browser.trySlowmotion()

	err := proto.InputInsertText{Text: text}.Call(k.page)
	return err
}
