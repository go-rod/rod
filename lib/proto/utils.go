package proto

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/ysmood/kit"
)

// Client interface to send the request.
// So that this lib doesn't handle anything has side effect.
type Client interface {
	Call(ctx context.Context, sessionID, methodName string, params json.RawMessage) (res []byte, err error)
}

// Payload interface returns the name of the event, such as "Page.loadEventFired"
type Payload interface {
	// MethodName is called method name is because the json-schema definition of it is "method".
	// And "eventName" is already used by a lot of existing fields.
	MethodName() string
}

// Caller interface to get the context of the request
type Caller interface {
	// CallContext returns ctx, client, and the sessionID
	CallContext() (context.Context, Client, string)
}

// Call method with request and response containers.
func Call(method string, req, res interface{}, caller Caller) error {
	ctx, client, id := caller.CallContext()

	payload, err := Normalize(req)
	if err != nil {
		return err
	}

	bin, err := client.Call(ctx, id, method, payload)
	if err != nil {
		return err
	}

	if res != nil {
		err = json.Unmarshal(bin, res)
		if err != nil {
			return err
		}
	}

	return nil
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

// Normalizable interface to transform the params into the correct data structure before being sent by the client.
// Because the json-schema doesn't cover all the type constrains of the protocol, we need this extra layer to do
// the normalization.
// Such as when send mouse wheel events, the deltaX and deltaY can't be omitted. The json-schema is wrong for them.
type Normalizable interface {
	Normalize() (json.RawMessage, error)
}

// Normalize the method payload
func Normalize(m interface{}) (json.RawMessage, error) {
	n, ok := m.(Normalizable)
	if ok {
		return n.Normalize()
	}
	return json.Marshal(m)
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

// TimeSinceEpoch UTC time in seconds, counted from January 1, 1970.
type TimeSinceEpoch struct {
	time.Time
}

// UnmarshalJSON interface
func (t *TimeSinceEpoch) UnmarshalJSON(b []byte) error {
	t.Time = (time.Unix(0, 0)).Add(
		time.Duration(gjson.ParseBytes(b).Float() * float64(time.Second)),
	)
	return nil
}

// MarshalJSON interface
func (t TimeSinceEpoch) MarshalJSON() ([]byte, error) {
	d := float64(t.Time.UnixNano()) / float64(time.Second)
	return json.Marshal(d)
}

// MonotonicTime Monotonically increasing time in seconds since an arbitrary point in the past.
type MonotonicTime struct {
	time.Duration
}

// UnmarshalJSON interface
func (t *MonotonicTime) UnmarshalJSON(b []byte) error {
	t.Duration = time.Duration(gjson.ParseBytes(b).Float() * float64(time.Second))
	return nil
}

// MarshalJSON interface
func (t MonotonicTime) MarshalJSON() ([]byte, error) {
	d := float64(t.Duration) / float64(time.Second)
	return json.Marshal(d)
}

var _ Normalizable = InputDispatchMouseEvent{}

// Normalize interface
func (e InputDispatchMouseEvent) Normalize() (json.RawMessage, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	if e.Type == InputDispatchMouseEventTypeMouseWheel {
		data, err = sjson.SetBytes(data, "deltaX", e.DeltaX)
		if err != nil {
			return nil, err
		}
		data, err = sjson.SetBytes(data, "deltaY", e.DeltaY)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}
