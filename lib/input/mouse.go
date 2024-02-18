package input

import "github.com/go-rod/rod/lib/proto"

// MouseKeys is the map for mouse keys.
var MouseKeys = map[proto.InputMouseButton]int{
	proto.InputMouseButtonLeft:    1,
	proto.InputMouseButtonRight:   2,
	proto.InputMouseButtonMiddle:  4,
	proto.InputMouseButtonBack:    8,
	proto.InputMouseButtonForward: 16,
}

// EncodeMouseButton into button flag.
func EncodeMouseButton(buttons []proto.InputMouseButton) (proto.InputMouseButton, int) {
	flag := int(0)
	for _, btn := range buttons {
		flag |= MouseKeys[btn]
	}
	btn := proto.InputMouseButton("none")
	if len(buttons) > 0 {
		btn = buttons[0]
	}
	return btn, flag
}
