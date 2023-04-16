package input_test

import (
	"testing"

	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/got"
	"github.com/ysmood/gson"
)

func TestKeyMap(t *testing.T) {
	g := got.T(t)

	k := input.Key('a')
	g.Eq(k.Info(), input.KeyInfo{
		Key:      "a",
		Code:     "KeyA",
		KeyCode:  65,
		Location: 0,
	})

	k = input.Key('A')
	g.Eq(k.Info(), input.KeyInfo{
		Key:      "A",
		Code:     "KeyA",
		KeyCode:  65,
		Location: 0,
	})
	g.True(k.Printable())

	k = input.Enter
	g.Eq(k.Info(), input.KeyInfo{
		Key:      "\r",
		Code:     "Enter",
		KeyCode:  13,
		Location: 0,
	})

	k = input.ShiftLeft
	g.Eq(k.Info(), input.KeyInfo /* len=4 */ {
		Key:      "Shift",
		Code:     "ShiftLeft",
		KeyCode:  16,
		Location: 1,
	})
	g.False(k.Printable())

	k = input.ShiftRight
	g.Eq(k.Info(), input.KeyInfo /* len=4 */ {
		Key:      "Shift",
		Code:     "ShiftRight",
		KeyCode:  16,
		Location: 2,
	})

	k, has := input.Digit1.Shift()
	g.True(has)
	g.Eq(k.Info().Key, "!")

	_, has = input.Enter.Shift()
	g.False(has)

	g.Panic(func() {
		input.Key('\n').Info()
	})
}

func TestKeyModifier(t *testing.T) {
	g := got.T(t)

	check := func(k input.Key, m int) {
		g.Helper()

		g.Eq(k.Modifier(), m)
	}

	check(input.KeyA, 0)
	check(input.AltLeft, 1)
	check(input.ControlLeft, 2)
	check(input.MetaLeft, 4)
	check(input.ShiftLeft, 8)
}

func TestKeyEncode(t *testing.T) {
	g := got.T(t)

	g.Eq(input.Key('a').Encode(proto.InputDispatchKeyEventTypeKeyDown, 0), &proto.InputDispatchKeyEvent{
		Type:                  "keyDown",
		Text:                  "a",
		UnmodifiedText:        "a",
		Code:                  "KeyA",
		Key:                   "a",
		WindowsVirtualKeyCode: 65,
		Location:              gson.Int(0),
	})

	g.Eq(input.Key('a').Encode(proto.InputDispatchKeyEventTypeKeyUp, 0), &proto.InputDispatchKeyEvent{
		Type:                  "keyUp",
		Text:                  "a",
		UnmodifiedText:        "a",
		Code:                  "KeyA",
		Key:                   "a",
		WindowsVirtualKeyCode: 65,
		Location:              gson.Int(0),
	})

	g.Eq(input.AltLeft.Encode(proto.InputDispatchKeyEventTypeKeyDown, 0), &proto.InputDispatchKeyEvent{
		Type:                  "rawKeyDown",
		Code:                  "AltLeft",
		Key:                   "Alt",
		WindowsVirtualKeyCode: 18,
		Location:              gson.Int(1),
	})

	g.Eq(input.Numpad1.Encode(proto.InputDispatchKeyEventTypeKeyDown, 0), &proto.InputDispatchKeyEvent{
		Type:                  "keyDown",
		Code:                  "Numpad1",
		Key:                   "1",
		Text:                  "1",
		UnmodifiedText:        "1",
		WindowsVirtualKeyCode: 35,
		IsKeypad:              true,
	})
}

func TestMac(t *testing.T) {
	g := got.T(t)

	old := input.IsMac
	input.IsMac = true
	defer func() { input.IsMac = old }()

	zero := 0

	g.Eq(input.ArrowDown.Encode(proto.InputDispatchKeyEventTypeKeyDown, 0), &proto.InputDispatchKeyEvent{
		Type:                  "rawKeyDown",
		Code:                  "ArrowDown",
		Key:                   "ArrowDown",
		WindowsVirtualKeyCode: 40,
		AutoRepeat:            false,
		IsKeypad:              false,
		IsSystemKey:           false,
		Location:              &zero,
		Commands: []string{
			"moveDown",
		},
	})
}
