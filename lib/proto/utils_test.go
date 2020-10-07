package proto_test

import (
	"testing"

	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/got"
)

type C struct {
	got.G
}

func Test(t *testing.T) {
	got.Each(t, C{})
}

func (c C) PatternToReg() {
	c.Eq(``, proto.PatternToReg(""))
	c.Eq(`\A.*\z`, proto.PatternToReg("*"))
	c.Eq(`\A.\z`, proto.PatternToReg("?"))
	c.Eq(`\Aa\z`, proto.PatternToReg("a"))
	c.Eq(`\Aa.com/.*/test\z`, proto.PatternToReg("a.com/*/test"))
	c.Eq(`\A\?\*\z`, proto.PatternToReg(`\?\*`))
	c.Eq(`\Aa.com\?a=10&b=\*\z`, proto.PatternToReg(`a.com\?a=10&b=\*`))
}
