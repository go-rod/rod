package proto_test

import "github.com/go-rod/rod/lib/proto"

func (t T) Point() {
	p := proto.NewPoint(1, 2).
		Add(proto.NewPoint(3, 4)).
		Minus(proto.NewPoint(1, 1)).
		Scale(2)

	t.Eq(p.X, 6)
	t.Eq(p.Y, 10)
}
