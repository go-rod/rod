package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func main() {
	w := NewWebSocket(launcher.New().MustLaunch())

	client := cdp.New().Start(w)

	p := rod.New().Client(client).MustConnect().MustPage("http://example.com")

	fmt.Println(p.MustInfo().Title)
}

// WebSocket is a custom websocket that uses gobwas/ws as the transport layer.
type WebSocket struct {
	conn net.Conn
}

// NewWebSocket ...
func NewWebSocket(u string) *WebSocket {
	conn, _, _, err := ws.Dial(context.Background(), u)
	if err != nil {
		log.Fatal(err)
	}
	return &WebSocket{conn}
}

// Send ...
func (w *WebSocket) Send(b []byte) error {
	return wsutil.WriteClientText(w.conn, b)
}

// Read ...
func (w *WebSocket) Read() ([]byte, error) {
	return wsutil.ReadServerText(w.conn)
}
