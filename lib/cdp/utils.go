package cdp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/ysmood/kit"
)

// JSON helper
type JSON struct {
	kit.JSONResult
}

// UnmarshalJSON interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	j.JSONResult = kit.JSON(data)
	return nil
}

// Object is the json object
type Object map[string]interface{}

// Array is the json array
type Array []interface{}

func prettyJSON(s *JSON) string {
	if s == nil {
		return ""
	}
	var val interface{}
	kit.E(json.Unmarshal([]byte(s.Raw), &val))
	return kit.Sdump(val)
}

func (cdp *Client) debugLog(obj interface{}) {
	if !cdp.debug {
		return
	}

	switch val := obj.(type) {
	case *Request:
		kit.E(fmt.Fprintf(
			kit.Stdout,
			"[cdp] %s %d %s %s %s\n",
			kit.C("req", "green"),
			val.ID,
			val.Method,
			val.SessionID,
			kit.Sdump(val.Params),
		))
	case *response:
		kit.E(fmt.Fprintf(kit.Stdout,
			"[cdp] %s %d %s %s\n",
			kit.C("res", "yellow"),
			val.ID,
			prettyJSON(val.Result),
			kit.Sdump(val.Error),
		))
	case *Event:
		kit.E(fmt.Fprintf(kit.Stdout,
			"[cdp] %s %s %s %s\n",
			kit.C("evt", "blue"),
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		))

	default:
		kit.Err(kit.Sdump(obj))
	}
}

type defaultWsClient struct {
	conn *websocket.Conn
}

func newDefaultWsClient(ctx context.Context, url string) Websocketable {
	dialer := *websocket.DefaultDialer
	dialer.ReadBufferSize = 25 * 1024 * 1024
	dialer.WriteBufferSize = 10 * 1024 * 1024

	conn, _, err := dialer.DialContext(ctx, url, nil)
	kit.E(err)

	return &defaultWsClient{conn: conn}
}

func (c *defaultWsClient) Send(data []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *defaultWsClient) Read() (data []byte, err error) {
	var msgType = -1
	for msgType != websocket.TextMessage && err == nil {
		msgType, data, err = c.conn.ReadMessage()
	}
	return
}
