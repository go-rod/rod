package input

import (
	"runtime"

	"github.com/go-rod/rod/lib/proto"
)

// Key contains information for generating a key press based off the unicode
// value.
//
// Example data for the following runes:
// 									'\r'  '\n'  | ','  '<'    | 'a'   'A'  | '\u0a07'
// 									_____________________________________________________
type Key struct {
	// Code is the key code:
	// 								"Enter"     | "Comma"     | "KeyA"     | "MediaStop"
	Code string

	// Key is the key value:
	// 								"Enter"     | ","   "<"   | "a"   "A"  | "MediaStop"
	Key string

	// Text is the text for printable keys:
	// 								"\r"  "\r"  | ","   "<"   | "a"   "A"  | ""
	Text string

	// Unmodified is the unmodified text for printable keys:
	// 								"\r"  "\r"  | ","   ","   | "a"   "a"  | ""
	Unmodified string

	// Native is the native scan code.
	// 								0x13  0x13  | 0xbc  0xbc  | 0x61  0x41 | 0x00ae
	Native int

	// Windows is the windows scan code.
	// 								0x13  0x13  | 0xbc  0xbc  | 0x61  0x41 | 0xe024
	Windows int

	// Shift indicates whether or not the Shift modifier should be sent.
	// 								false false | false true  | false true | false
	Shift bool

	// Print indicates whether or not the character is a printable character
	// (ie, should a "char" event be generated).
	// 								true  true  | true  true  | true  true | false
	Print bool
}

// Encode encodes a keyDown, char, and keyUp sequence for the specified rune.
func Encode(r rune) []*proto.InputDispatchKeyEvent {
	// force \n -> \r
	if r == '\n' {
		r = '\r'
	}

	// if not known key, encode as unidentified
	v := Keys[r]

	// create
	keyDown := proto.InputDispatchKeyEvent{
		Type:                  "keyDown",
		Key:                   v.Key,
		Code:                  v.Code,
		NativeVirtualKeyCode:  v.Native,
		WindowsVirtualKeyCode: v.Windows,
	}
	if runtime.GOOS == "darwin" {
		keyDown.NativeVirtualKeyCode = 0
	}
	if v.Shift {
		keyDown.Modifiers |= 8
	}

	keyUp := keyDown
	keyUp.Type = "keyUp"

	// printable, so create char event
	if v.Print {
		keyChar := keyDown
		keyChar.Type = "char"
		keyChar.Text = v.Text
		keyChar.UnmodifiedText = v.Unmodified

		// the virtual key code for char events for printable characters will
		// be different than the defined keycode when not shifted...
		//
		// specifically, it always sends the ascii value as the scan code,
		// which is available as the rune.
		keyChar.NativeVirtualKeyCode = int(r)
		keyChar.WindowsVirtualKeyCode = int(r)

		return []*proto.InputDispatchKeyEvent{&keyDown, &keyChar, &keyUp}
	}

	return []*proto.InputDispatchKeyEvent{&keyDown, &keyUp}
}
