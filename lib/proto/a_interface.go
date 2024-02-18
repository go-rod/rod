// Package proto is a lib to encode/decode the data of the cdp protocol.
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

// Sessionable type has a proto.TargetSessionID for its methods.
type Sessionable interface {
	GetSessionID() TargetSessionID
}

// Contextable type has a context.Context for its methods.
type Contextable interface {
	GetContext() context.Context
}

// Request represents a cdp.Request.Method.
type Request interface {
	// ProtoReq returns the cdp.Request.Method
	ProtoReq() string
}

// Event represents a cdp.Event.Params.
type Event interface {
	// ProtoEvent returns the cdp.Event.Method
	ProtoEvent() string
}

// GetType from method name of this package,
// such as proto.GetType("Page.enable") will return the type of proto.PageEnable.
func GetType(methodName string) reflect.Type {
	return types[methodName]
}

// ParseMethodName to domain and name.
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
	if tsa, ok := c.(Sessionable); ok {
		sessionID = string(tsa.GetSessionID())
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
