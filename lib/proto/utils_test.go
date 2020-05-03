package proto_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/proto"
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
	return kit.MustToJSONBytes(c.ret), c.err
}

type Caller struct {
	*Client
}

var _ proto.Caller = &Caller{}

func (c *Caller) CallContext() (context.Context, proto.Client, string) {
	return context.Background(), c.Client, c.Client.sessionID
}

func TestE(t *testing.T) {
	assert.Panics(t, func() {
		proto.E(errors.New("err"))
	})
}

func TestJSON(t *testing.T) {
	var j proto.JSON
	kit.E(json.Unmarshal([]byte("10"), &j))
	assert.EqualValues(t, 10, j.Int())

	assert.Equal(t, "true", kit.MustToJSON(proto.NewJSON(true)))
}
