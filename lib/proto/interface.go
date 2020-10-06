package proto

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
)

// Client interface to send the request.
// So that this lib doesn't handle anything has side effect.
type Client interface {
	Call(ctx context.Context, sessionID, methodName string, params interface{}) (res []byte, err error)
}

// TargetSessionable type has a proto.TargetSessionID for its methods
type TargetSessionable interface {
	GetTargetSessionID() TargetSessionID
}

// Contextable type has a context.Context for its methods
type Contextable interface {
	GetContext() context.Context
}

// Payload represents a cdp.Response.Result or cdp.Event.Params
type Payload interface {
	// ProtoName is the cdp.Response.Method or cdp.Event.Method
	ProtoName() string
}

// GetType from method name of this package,
// such as proto.GetType("Page.enable") will return the type of proto.PageEnable
func GetType(methodName string) reflect.Type {
	return types[methodName]
}

// ParseMethodName to domain and name
func ParseMethodName(method string) (domain, name string) {
	arr := strings.Split(method, ".")
	return arr[0], arr[1]
}

// call method with request and response containers.
func call(method string, req, res interface{}, c Client) error {
	ctx := context.Background()
	if cta, ok := c.(Contextable); ok {
		ctx = cta.GetContext()
	}

	sessionID := ""
	if tsa, ok := c.(TargetSessionable); ok {
		sessionID = string(tsa.GetTargetSessionID())
	}

	bin, err := c.Call(ctx, sessionID, method, req)
	if err != nil {
		return err
	}
	if res == nil {
		return nil
	}
	return json.Unmarshal(bin, res)
}
