package cdp

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
)

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

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

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
