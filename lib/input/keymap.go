package input

// Key names
// Reference: https://github.com/microsoft/playwright/blob/main/packages/playwright-core/src/server/usKeyboardLayout.ts
var (
	// Functions row
	//
	Escape = AddKey("Escape", "", "Escape", 27, 0)
	F1     = AddKey("F1", "", "F1", 112, 0)
	F2     = AddKey("F2", "", "F2", 113, 0)
	F3     = AddKey("F3", "", "F3", 114, 0)
	F4     = AddKey("F4", "", "F4", 115, 0)
	F5     = AddKey("F5", "", "F5", 116, 0)
	F6     = AddKey("F6", "", "F6", 117, 0)
	F7     = AddKey("F7", "", "F7", 118, 0)
	F8     = AddKey("F8", "", "F8", 119, 0)
	F9     = AddKey("F9", "", "F9", 120, 0)
	F10    = AddKey("F10", "", "F10", 121, 0)
	F11    = AddKey("F11", "", "F11", 122, 0)
	F12    = AddKey("F12", "", "F12", 123, 0)

	// Numbers row
	//
	Backquote = AddKey("`", "~", "Backquote", 192, 0)
	Digit1    = AddKey("1", "!", "Digit1", 49, 0)
	Digit2    = AddKey("2", "@", "Digit2", 50, 0)
	Digit3    = AddKey("3", "#", "Digit3", 51, 0)
	Digit4    = AddKey("4", "$", "Digit4", 52, 0)
	Digit5    = AddKey("5", "%", "Digit5", 53, 0)
	Digit6    = AddKey("6", "^", "Digit6", 54, 0)
	Digit7    = AddKey("7", "&", "Digit7", 55, 0)
	Digit8    = AddKey("8", "*", "Digit8", 56, 0)
	Digit9    = AddKey("9", "(", "Digit9", 57, 0)
	Digit0    = AddKey("0", ")", "Digit0", 48, 0)
	Minus     = AddKey("-", "_", "Minus", 189, 0)
	Equal     = AddKey("=", "+", "Equal", 187, 0)
	Backslash = AddKey(`\`, "|", "Backslash", 220, 0)
	Backspace = AddKey("Backspace", "", "Backspace", 8, 0)

	// First row
	//
	Tab          = AddKey("\t", "", "Tab", 9, 0)
	KeyQ         = AddKey("q", "Q", "KeyQ", 81, 0)
	KeyW         = AddKey("w", "W", "KeyW", 87, 0)
	KeyE         = AddKey("e", "E", "KeyE", 69, 0)
	KeyR         = AddKey("r", "R", "KeyR", 82, 0)
	KeyT         = AddKey("t", "T", "KeyT", 84, 0)
	KeyY         = AddKey("y", "Y", "KeyY", 89, 0)
	KeyU         = AddKey("u", "U", "KeyU", 85, 0)
	KeyI         = AddKey("i", "I", "KeyI", 73, 0)
	KeyO         = AddKey("o", "O", "KeyO", 79, 0)
	KeyP         = AddKey("p", "P", "KeyP", 80, 0)
	BracketLeft  = AddKey("[", "{", "BracketLeft", 219, 0)
	BracketRight = AddKey("]", "}", "BracketRight", 221, 0)

	// Second row
	//
	CapsLock  = AddKey("CapsLock", "", "CapsLock", 20, 0)
	KeyA      = AddKey("a", "A", "KeyA", 65, 0)
	KeyS      = AddKey("s", "S", "KeyS", 83, 0)
	KeyD      = AddKey("d", "D", "KeyD", 68, 0)
	KeyF      = AddKey("f", "F", "KeyF", 70, 0)
	KeyG      = AddKey("g", "G", "KeyG", 71, 0)
	KeyH      = AddKey("h", "H", "KeyH", 72, 0)
	KeyJ      = AddKey("j", "J", "KeyJ", 74, 0)
	KeyK      = AddKey("k", "K", "KeyK", 75, 0)
	KeyL      = AddKey("l", "L", "KeyL", 76, 0)
	Semicolon = AddKey(";", ":", "Semicolon", 186, 0)
	Quote     = AddKey("'", `"`, "Quote", 222, 0)
	Enter     = AddKey("\r", "", "Enter", 13, 0)

	// Third row
	//
	ShiftLeft  = AddKey("Shift", "", "ShiftLeft", 16, 1)
	KeyZ       = AddKey("z", "Z", "KeyZ", 90, 0)
	KeyX       = AddKey("x", "X", "KeyX", 88, 0)
	KeyC       = AddKey("c", "C", "KeyC", 67, 0)
	KeyV       = AddKey("v", "V", "KeyV", 86, 0)
	KeyB       = AddKey("b", "B", "KeyB", 66, 0)
	KeyN       = AddKey("n", "N", "KeyN", 78, 0)
	KeyM       = AddKey("m", "M", "KeyM", 77, 0)
	Comma      = AddKey(",", "<", "Comma", 188, 0)
	Period     = AddKey(".", ">", "Period", 190, 0)
	Slash      = AddKey("/", "?", "Slash", 191, 0)
	ShiftRight = AddKey("Shift", "", "ShiftRight", 16, 2)

	// Last row
	//
	ControlLeft  = AddKey("Control", "", "ControlLeft", 17, 1)
	MetaLeft     = AddKey("Meta", "", "MetaLeft", 91, 1)
	AltLeft      = AddKey("Alt", "", "AltLeft", 18, 1)
	Space        = AddKey(" ", "", "Space", 32, 0)
	AltRight     = AddKey("Alt", "", "AltRight", 18, 2)
	AltGraph     = AddKey("AltGraph", "", "AltGraph", 225, 0)
	MetaRight    = AddKey("Meta", "", "MetaRight", 92, 2)
	ContextMenu  = AddKey("ContextMenu", "", "ContextMenu", 93, 0)
	ControlRight = AddKey("Control", "", "ControlRight", 17, 2)

	// Center block
	//
	PrintScreen = AddKey("PrintScreen", "", "PrintScreen", 44, 0)
	ScrollLock  = AddKey("ScrollLock", "", "ScrollLock", 145, 0)
	Pause       = AddKey("Pause", "", "Pause", 19, 0)
	PageUp      = AddKey("PageUp", "", "PageUp", 33, 0)
	PageDown    = AddKey("PageDown", "", "PageDown", 34, 0)
	Insert      = AddKey("Insert", "", "Insert", 45, 0)
	Delete      = AddKey("Delete", "", "Delete", 46, 0)
	Home        = AddKey("Home", "", "Home", 36, 0)
	End         = AddKey("End", "", "End", 35, 0)
	ArrowLeft   = AddKey("ArrowLeft", "", "ArrowLeft", 37, 0)
	ArrowUp     = AddKey("ArrowUp", "", "ArrowUp", 38, 0)
	ArrowRight  = AddKey("ArrowRight", "", "ArrowRight", 39, 0)
	ArrowDown   = AddKey("ArrowDown", "", "ArrowDown", 40, 0)

	// Numpad
	//
	NumLock        = AddKey("NumLock", "", "NumLock", 144, 0)
	NumpadDivide   = AddKey("/", "", "NumpadDivide", 111, 3)
	NumpadMultiply = AddKey("*", "", "NumpadMultiply", 106, 3)
	NumpadSubtract = AddKey("-", "", "NumpadSubtract", 109, 3)
	Numpad7        = AddKey("7", "", "Numpad7", 36, 3)
	Numpad8        = AddKey("8", "", "Numpad8", 38, 3)
	Numpad9        = AddKey("9", "", "Numpad9", 33, 3)
	Numpad4        = AddKey("4", "", "Numpad4", 37, 3)
	Numpad5        = AddKey("5", "", "Numpad5", 12, 3)
	Numpad6        = AddKey("6", "", "Numpad6", 39, 3)
	NumpadAdd      = AddKey("+", "", "NumpadAdd", 107, 3)
	Numpad1        = AddKey("1", "", "Numpad1", 35, 3)
	Numpad2        = AddKey("2", "", "Numpad2", 40, 3)
	Numpad3        = AddKey("3", "", "Numpad3", 34, 3)
	Numpad0        = AddKey("0", "", "Numpad0", 45, 3)
	NumpadDecimal  = AddKey(".", "", "NumpadDecimal", 46, 3)
	NumpadEnter    = AddKey("\r", "", "NumpadEnter", 13, 3)
)
