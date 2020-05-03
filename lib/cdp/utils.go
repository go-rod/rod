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

func prettyJSON(s []byte) string {
	var val interface{}
	kit.E(json.Unmarshal(s, &val))
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
