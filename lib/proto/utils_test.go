package proto_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

type C struct {
	got.Assertion
}

func Test(t *testing.T) {
	got.Each(t, C{})
}

type Client struct {
	sessionID  string
	methodName string
	params     interface{}
	err        error
	ret        interface{}
}

var _ proto.Client = &Client{}

func (c *Client) Call(ctx context.Context, sessionID, methodName string, params json.RawMessage) (res []byte, err error) {
	c.sessionID = sessionID
	c.methodName = methodName
	c.params = params
	return utils.MustToJSONBytes(c.ret), c.err
}

type Caller struct {
	*Client
}

var _ proto.Caller = &Caller{}

func (c *Caller) CallContext() (context.Context, proto.Client, string) {
	return context.Background(), c.Client, c.Client.sessionID
}

type normalizeErr struct {
}

func (n normalizeErr) Normalize() (json.RawMessage, error) {
	return nil, errors.New("err")
}

func (c C) Call() {
	err := proto.Call("", normalizeErr{}, "", &Caller{&Client{}})
	c.Err(err)

	err = proto.Call("", "", "", &Caller{&Client{err: errors.New("err")}})
	c.Err(err)

	err = proto.Call("", "", func() {}, &Caller{&Client{}})
	c.Err(err)
}

func (c C) ParseMethodName() {
	d, n := proto.ParseMethodName("Page.enable")
	c.Eq("Page", d)
	c.Eq("enable", n)
}

func (c C) GetType() {
	method := proto.GetType("Page.enable")
	c.Eq(reflect.TypeOf(proto.PageEnable{}), method)
}

func (c C) JSON() {
	var j proto.JSON
	c.E(json.Unmarshal([]byte("10"), &j))
	c.Eq(10, j.Int())

	c.Eq("null", utils.MustToJSON(proto.JSON{}))
	c.Eq("true", utils.MustToJSON(proto.NewJSON(true)))

	c.Eq("1 2 3", proto.NewJSON([]int{1, 2, 3}).Join(" "))

	j = proto.NewJSON([]byte("{}"))
	j, err := j.Set("a", 1)
	c.Nil(err)
	c.Eq(j.Get("a").Num, 1)
	c.Err(j.Set("", 1))
}

func (c C) TimeCodec() {
	raw := []byte("123.123")
	var duration proto.MonotonicTime
	c.E(json.Unmarshal(raw, &duration))

	c.Eq(123123, duration.Milliseconds())

	data, err := json.Marshal(duration)
	c.E(err)
	c.Eq(raw, data)

	raw = []byte("123")
	var datetime proto.TimeSinceEpoch
	c.E(json.Unmarshal(raw, &datetime))

	c.Eq(123, datetime.Unix())

	data, err = json.Marshal(datetime)
	c.E(err)
	c.Eq(raw, data)
}

func (c C) NormalizeInputDispatchMouseEvent() {
	e := proto.InputDispatchMouseEvent{
		Type: proto.InputDispatchMouseEventTypeMouseWheel,
	}

	data, err := e.Normalize()
	c.E(err)

	c.Eq(`{"type":"mouseWheel","x":0,"y":0,"deltaX":0,"deltaY":0}`, string(data))
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

func (c C) Rect() {
	rect := proto.DOMQuad{
		336, 382, 361, 382, 361, 421, 336, 412,
	}

	c.Eq(348.5, rect.Center().X)
	c.Eq(399.25, rect.Center().Y)

	res := &proto.DOMGetContentQuadsResult{}
	c.Nil(res.OnePointInside())

	res = &proto.DOMGetContentQuadsResult{Quads: []proto.DOMQuad{rect}}
	pt := res.OnePointInside()
	c.Eq(348.5, pt.X)
	c.Eq(399.25, pt.Y)
}

func (c C) InputTouchPointMoveTo() {
	p := &proto.InputTouchPoint{}
	p.MoveTo(1, 2)

	c.Eq(1, p.X)
	c.Eq(2, p.Y)
}
