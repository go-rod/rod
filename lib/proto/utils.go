package proto

import (
	"context"

	"github.com/tidwall/gjson"
	"github.com/ysmood/kit"
)

// Client interface to send the request.
// So that this lib doesn't handle any thing has side effect.
type Client interface {
	Call(ctx context.Context, sessionID, methodName string, params interface{}) (res []byte, err error)
}

// Event interface
type Event interface {
	MethodName() string
}

// Caller interface to get the context of the request
type Caller interface {
	// CallContext returns ctx, client, and the sessionID
	CallContext() (context.Context, Client, string)
}

// E panics err if err not nil
func E(err error) {
	if err != nil {
		panic(err)
	}
}

// JSON value
type JSON struct {
	gjson.Result
}

// NewJSON json object
func NewJSON(val interface{}) JSON {
	j := JSON{}
	j.Raw = kit.MustToJSON(val)
	return j
}

// UnmarshalJSON interface
func (j *JSON) UnmarshalJSON(b []byte) error {
	j.Result = gjson.ParseBytes(b)
	return nil
}

// MarshalJSON interface
func (j JSON) MarshalJSON() ([]byte, error) {
	return []byte(j.Raw), nil
}
