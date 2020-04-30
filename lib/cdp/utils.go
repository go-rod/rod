package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

	prefix := time.Now().Format("[cdp] [2006-01-02 15:04:05]")

	switch val := obj.(type) {
	case *Request:
		kit.E(fmt.Fprintf(
			kit.Stdout,
			"%s %s %d %s %s %s\n",
			prefix,
			kit.C("-> req", "green"),
			val.ID,
			val.Method,
			val.SessionID,
			kit.Sdump(val.Params),
		))
	case *response:
		kit.E(fmt.Fprintf(kit.Stdout,
			"%s %s %d %s %s\n",
			prefix,
			kit.C("<- res", "yellow"),
			val.ID,
			prettyJSON(val.Result),
			kit.Sdump(val.Error),
		))
	case *Event:
		kit.E(fmt.Fprintf(kit.Stdout,
			"%s %s %s %s %s\n",
			prefix,
			kit.C("evt", "blue"),
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		))

	default:
		kit.Err(kit.Sdump(obj))
	}
}

// DefaultWsClient for CDP
type DefaultWsClient struct {
	conn *websocket.Conn
}

// NewDefaultWsClient instance
func NewDefaultWsClient(ctx context.Context, url string, header http.Header) Websocketable {
	dialer := *websocket.DefaultDialer
	dialer.ReadBufferSize = 25 * 1024 * 1024
	dialer.WriteBufferSize = 10 * 1024 * 1024

	conn, _, err := dialer.DialContext(ctx, url, header)
	kit.E(err)

	return &DefaultWsClient{conn: conn}
}

// Send a message
func (c *DefaultWsClient) Send(data []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Read a message
func (c *DefaultWsClient) Read() (data []byte, err error) {
	var msgType = -1
	for msgType != websocket.TextMessage && err == nil {
		msgType, data, err = c.conn.ReadMessage()
	}
	return
}
