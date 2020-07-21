package proto_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
)

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
	return kit.MustToJSONBytes(c.ret), c.err
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

func TestCall(t *testing.T) {
	err := proto.Call("", normalizeErr{}, "", &Caller{&Client{}})
	assert.Error(t, err)

	err = proto.Call("", "", "", &Caller{&Client{err: errors.New("err")}})
	assert.Error(t, err)

	err = proto.Call("", "", func() {}, &Caller{&Client{}})
	assert.Error(t, err)
}

func TestE(t *testing.T) {
	assert.Panics(t, func() {
		proto.E(errors.New("err"))
	})
}

func TestParseMethodName(t *testing.T) {
	d, n := proto.ParseMethodName("Page.enable")
	assert.Equal(t, "Page", d)
	assert.Equal(t, "enable", n)
}

func TestGetType(t *testing.T) {
	method := proto.GetType("Page.enable")
	assert.Equal(t, reflect.TypeOf(proto.PageEnable{}), method)
}

func TestJSON(t *testing.T) {
	var j proto.JSON
	kit.E(json.Unmarshal([]byte("10"), &j))
	assert.EqualValues(t, 10, j.Int())

	assert.Equal(t, "true", kit.MustToJSON(proto.NewJSON(true)))

	assert.Equal(t, "1 2 3", proto.NewJSON([]int{1, 2, 3}).Join(" "))
}

func TestTimeCodec(t *testing.T) {
	raw := []byte("123.123")
	var duration proto.MonotonicTime
	kit.E(json.Unmarshal(raw, &duration))

	assert.EqualValues(t, 123123, duration.Milliseconds())

	data, err := json.Marshal(duration)
	kit.E(err)
	assert.Equal(t, raw, data)

	raw = []byte("123")
	var datetime proto.TimeSinceEpoch
	kit.E(json.Unmarshal(raw, &datetime))

	assert.EqualValues(t, 123, datetime.Unix())

	data, err = json.Marshal(datetime)
	kit.E(err)
	assert.Equal(t, raw, data)
}

func TestNormalizeInputDispatchMouseEvent(t *testing.T) {
	e := proto.InputDispatchMouseEvent{
		Type: proto.InputDispatchMouseEventTypeMouseWheel,
	}

	data, err := e.Normalize()
	kit.E(err)

	assert.Equal(t, `{"type":"mouseWheel","x":0,"y":0,"deltaX":0,"deltaY":0}`, string(data))
}

func TestPatternToReg(t *testing.T) {
	assert.Equal(t, ``, proto.PatternToReg(""))
	assert.Equal(t, `\A.*\z`, proto.PatternToReg("*"))
	assert.Equal(t, `\A.\z`, proto.PatternToReg("?"))
	assert.Equal(t, `\Aa\z`, proto.PatternToReg("a"))
	assert.Equal(t, `\Aa.com/.*/test\z`, proto.PatternToReg("a.com/*/test"))
	assert.Equal(t, `\A\?\*\z`, proto.PatternToReg(`\?\*`))
	assert.Equal(t, `\Aa.com\?a=10&b=\*\z`, proto.PatternToReg(`a.com\?a=10&b=\*`))
}

func TestRect(t *testing.T) {
	b := &proto.DOMBoxModel{Content: proto.DOMQuad{
		336, 382, 361, 382, 361, 421, 336, 412,
	}}

	assert.Equal(t, proto.DOMRect{X: 336, Y: 382, Width: 25, Height: 30}, *b.Rect())
}
