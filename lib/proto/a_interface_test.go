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

var (
	_ proto.Client      = &Client{}
	_ proto.Sessionable = &Client{}
	_ proto.Contextable = &Client{}
)

func (c *Client) Call(_ context.Context, sessionID, methodName string, params interface{}) (res []byte, err error) {
	c.sessionID = sessionID
	c.methodName = methodName
	c.params = params
	return utils.MustToJSONBytes(c.ret), c.err
}

func (c *Client) GetSessionID() proto.TargetSessionID { return "" }

func (c *Client) GetContext() context.Context { return nil }

func (t T) CallErr() {
	client := &Client{err: errors.New("err")}
	t.Eq(proto.PageEnable{}.Call(client).Error(), "err")
}

func (t T) ParseMethodName() {
	d, n := proto.ParseMethodName("Page.enable")
	t.Eq("Page", d)
	t.Eq("enable", n)
}

func (t T) GetType() {
	method := proto.GetType("Page.enable")
	t.Eq(reflect.TypeOf(proto.PageEnable{}), method)
}

func (t T) TimeCodec() {
	raw := []byte("123.123")
	var duration proto.MonotonicTime
	t.E(json.Unmarshal(raw, &duration))

	t.Eq(123123, duration.Duration().Milliseconds())
	t.Eq("2m3.123s", duration.String())

	data, err := json.Marshal(duration)
	t.E(err)
	t.Eq(raw, data)

	raw = []byte("1234567890")
	var datetime proto.TimeSinceEpoch
	t.E(json.Unmarshal(raw, &datetime))

	t.Eq(1234567890, datetime.Time().Unix())
	t.Has(datetime.String(), "2009-02")

	data, err = json.Marshal(datetime)
	t.E(err)
	t.Eq(raw, data)
}

func (t T) Rect() {
	rect := proto.DOMQuad{
		336, 382, 361, 382, 361, 421, 336, 412,
	}

	t.Eq(348.5, rect.Center().X)
	t.Eq(399.25, rect.Center().Y)

	res := &proto.DOMGetContentQuadsResult{}
	t.Nil(res.OnePointInside())

	res = &proto.DOMGetContentQuadsResult{Quads: []proto.DOMQuad{{1, 1, 2, 1, 2, 1, 1, 1}}}
	t.Nil(res.OnePointInside())

	res = &proto.DOMGetContentQuadsResult{Quads: []proto.DOMQuad{rect}}
	pt := res.OnePointInside()
	t.Eq(348.5, pt.X)
	t.Eq(399.25, pt.Y)
}

func (t T) Area() {
	t.Eq(proto.DOMQuad{1, 1, 2, 1, 2, 1, 1, 1}.Area(), 0)
	t.Eq(proto.DOMQuad{1, 1, 2, 1, 2, 2, 1, 2}.Area(), 1)
	t.Eq(proto.DOMQuad{1, 1, 2, 1, 2, 4, 1, 3}.Area(), 2.5)
}

func (t T) Box() {
	res := &proto.DOMGetContentQuadsResult{Quads: []proto.DOMQuad{
		{1, 1, 2, 1, 2, 2, 1, 2},
		{2, 0, 3, 0, 3, 1, 2, 1},
		{0, 2, 1, 2, 1, 3, 0, 3},
	}}
	t.Eq(res.Box(), &proto.DOMRect{
		X:      0,
		Y:      0,
		Width:  3,
		Height: 3,
	})

	t.Nil((&proto.DOMGetContentQuadsResult{}).Box())
}

func (t T) InputTouchPointMoveTo() {
	p := &proto.InputTouchPoint{}
	p.MoveTo(1, 2)

	t.Eq(1, p.X)
	t.Eq(2, p.Y)
}

func (t T) CookiesToParams() {
	list := proto.CookiesToParams([]*proto.NetworkCookie{{
		Name:  "name",
		Value: "val",
	}})

	t.Eq(list[0].Name, "name")
	t.Eq(list[0].Value, "val")
}

func (t T) GeneratorOptimize() {
	var _ proto.TargetTargetInfoType = proto.TargetTargetInfoTypeBackgroundPage
	var _ proto.TargetTargetInfoType = proto.TargetTargetInfoTypePage

	var _ proto.PageLifecycleEventName = proto.PageLifecycleEventNameInit
	var _ proto.PageLifecycleEventName = proto.PageLifecycleEventNameFirstContentfulPaint
	var _ proto.PageLifecycleEventName = proto.PageLifecycleEventNameFirstImagePaint

	a := proto.InputDispatchKeyEvent{}
	var _ proto.TimeSinceEpoch = a.Timestamp
	b := proto.NetworkCookie{}
	var _ proto.TimeSinceEpoch = b.Expires

	c := proto.NetworkDataReceived{}
	var _ proto.MonotonicTime = c.Timestamp

	d := proto.NetworkCookie{}
	var _ proto.TimeSinceEpoch = d.Expires
}
