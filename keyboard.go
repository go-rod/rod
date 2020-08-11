package rod

import (
	"sync"

	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// Keyboard represents the keyboard on a page, it's always related the main frame
type Keyboard struct {
	lock *sync.Mutex

	page *Page

	// modifiers are currently beening pressed
	modifiers int64
}

// Down doc is similar to the method MustDown
func (k *Keyboard) Down(key rune) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	actions := input.Encode(key)

	err := actions[0].Call(k.page)
	if err != nil {
		return err
	}
	k.modifiers = actions[0].Modifiers
	return nil
}

// Up doc is similar to the method MustUp
func (k *Keyboard) Up(key rune) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	actions := input.Encode(key)

	err := actions[len(actions)-1].Call(k.page)
	if err != nil {
		return err
	}
	k.modifiers = 0
	return nil
}

// Press doc is similar to the method MustPress
func (k *Keyboard) Press(key rune) error {
	k.lock.Lock()
	defer k.lock.Unlock()

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

// InsertText doc is similar to the method MustInsertText
func (k *Keyboard) InsertText(text string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	if k.page.browser.trace {
		defer k.page.Overlay(0, 0, 200, 0, "insert text "+text)()
	}
	k.page.browser.trySlowmotion()

	err := proto.InputInsertText{Text: text}.Call(k.page)
	return err
}
