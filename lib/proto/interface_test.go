package proto_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

type Client struct {
	sessionID  string
	methodName string
	params     interface{}
	err        error
	ret        interface{}
}

var _ proto.Client = &Client{}

func (c *Client) Call(ctx context.Context, sessionID, methodName string, params interface{}) (res []byte, err error) {
	c.sessionID = sessionID
	c.methodName = methodName
	c.params = params
	return utils.MustToJSONBytes(c.ret), c.err
}

func (c *Client) GetTargetSessionID() proto.TargetSessionID { return "" }

func (c *Client) GetContext() context.Context { return nil }

func (c C) CallErr() {
	client := &Client{err: errors.New("err")}
	c.Eq(proto.PageEnable{}.Call(client).Error(), "err")
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

	data, err := json.Marshal(e)
	c.E(err)

	c.Eq(`{"type":"mouseWheel","x":0,"y":0,"deltaX":0,"deltaY":0}`, string(data))

	ee := proto.InputDispatchMouseEvent{
		Type: proto.InputDispatchMouseEventTypeMouseMoved,
	}

	data, err = json.Marshal(ee)
	c.E(err)

	c.Eq(`{"type":"mouseMoved","x":0,"y":0}`, string(data))
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
