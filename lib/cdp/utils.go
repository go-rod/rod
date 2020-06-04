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

func prettyJSON(s interface{}) string {
	raw, ok := s.(json.RawMessage)
	if ok {
		var val interface{}
		_ = json.Unmarshal(raw, &val)
		return kit.Sdump(val)
	}

	return kit.Sdump(raw)
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
			prettyJSON(val.Params),
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

// DefaultWsClient is the default websocket client
type DefaultWsClient struct{}

// DefaultWsConn is the default websocket connection type
type DefaultWsConn struct {
	conn *websocket.Conn
}

// Connect interface
func (c DefaultWsClient) Connect(ctx context.Context, url string, header http.Header) (WebsocketableConn, error) {
	dialer := *websocket.DefaultDialer
	dialer.ReadBufferSize = 25 * 1024 * 1024
	dialer.WriteBufferSize = 10 * 1024 * 1024

	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return nil, err
	}

	return &DefaultWsConn{conn: conn}, nil

}

// Send a message
func (c *DefaultWsConn) Send(data []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Read a message
func (c *DefaultWsConn) Read() (data []byte, err error) {
	var msgType = -1
	for msgType != websocket.TextMessage && err == nil {
		msgType, data, err = c.conn.ReadMessage()
	}
	return
}
