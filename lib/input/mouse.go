package input

// MouseKeys is the map for mouse keys
var MouseKeys = map[string]int{
	"left":    1,
	"right":   2,
	"middle":  4,
	"back":    8,
	"forward": 16,
}

// EncodeMouseButton into button flag
func EncodeMouseButton(buttons []string) (string, int) {
	flag := 0
	for _, btn := range buttons {
		flag |= MouseKeys[btn]
	}
	btn := "none"
	if len(buttons) > 0 {
		btn = buttons[0]
	}
	return btn, flag
}
