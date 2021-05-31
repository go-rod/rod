package cdp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

// Dialer interface for WebSocket connection
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

var _ WebSocketable = &WebSocket{}

// WebSocket client for chromium. It only implements a subset of WebSocket protocol.
// Limitation: https://bugs.chromium.org/p/chromium/issues/detail?id=1069431
// Ref: https://tools.ietf.org/html/rfc6455
type WebSocket struct {
	// Dialer is usually used for proxy
	Dialer Dialer

	close  func()
	conn   net.Conn
	r      *bufio.Reader
	header [18]byte // Send is thread-safe, so we can safely share a header for all frames
	mask   []byte
}

// Connect to browser
func (ws *WebSocket) Connect(ctx context.Context, wsURL string, header http.Header) error {
	if ws.conn != nil {
		panic("duplicated connection: " + wsURL)
	}

	ctx, cancel := context.WithCancel(ctx)
	ws.close = cancel

	u, err := url.Parse(wsURL)
	if err != nil {
		return err
	}

	ws.initDialer(u)

	conn, err := ws.Dialer.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	ws.initConstants()

	ws.conn = conn
	ws.r = bufio.NewReader(conn)
	return ws.handshake(ctx, u, header)
}

func (ws *WebSocket) initDialer(u *url.URL) {
	if ws.Dialer != nil {
		return
	}

	if u.Scheme == "wss" {
		ws.Dialer = &tlsDialer{}
		if u.Port() == "" {
			u.Host += ":443"
		}
	} else {
		ws.Dialer = &net.Dialer{}
	}
}

func (ws *WebSocket) initConstants() {
	// FIN is alway true, Opcode is always text frame.
	ws.header = [18]byte{0b1000_0001}

	ws.mask = []byte{0, 1, 2, 3}
}

// Send a message to browser.
// Because we use zero-copy design, it will modify the content of the msg.
// It won't allocate new memory.
func (ws *WebSocket) Send(msg []byte) error {
	ws.header[1] = 0b1000_0000

	size := len(msg)
	fieldLen := 0
	switch {
	case size <= 125:
		ws.header[1] |= byte(size)
	case size < 65536:
		ws.header[1] |= 126
		fieldLen = 2
	default:
		ws.header[1] |= 127
		fieldLen = 8
	}

	var i int
	for i = 0; i < fieldLen; i++ {
		digit := (fieldLen - i - 1) * 8
		ws.header[i+2] = byte((size >> digit) & 0xff)
	}

	copy(ws.header[i+2:], ws.mask)
	_, err := ws.conn.Write(ws.header[:i+6])
	if err != nil {
		return ws.checkClose(err)
	}

	for i := range msg {
		msg[i] = msg[i] ^ ws.mask[i%4]
	}

	_, err = ws.conn.Write(msg)
	return ws.checkClose(err)
}

// Read a message from browser
func (ws *WebSocket) Read() ([]byte, error) {
	_, _ = ws.r.ReadByte()
	b, err := ws.r.ReadByte()
	if err != nil {
		return nil, ws.checkClose(err)
	}

	size := 0
	fieldLen := 0

	b &= 0x7f
	switch {
	case b <= 125:
		size = int(b)
	case b == 126:
		fieldLen = 2
	case b == 127:
		fieldLen = 8
	}

	for i := 0; i < fieldLen; i++ {
		b, err := ws.r.ReadByte()
		if err != nil {
			return nil, ws.checkClose(err)
		}

		size = size<<8 + int(b)
	}

	data := make([]byte, size)
	_, err = io.ReadFull(ws.r, data)
	return data, ws.checkClose(err)
}

// ErrBadHandshake type
type ErrBadHandshake struct {
	Status string
	Body   string
}

func (e *ErrBadHandshake) Error() string {
	return fmt.Sprintf(
		"websocket bad handshake: %s. %s",
		e.Status, e.Body,
	)
}

func (ws *WebSocket) handshake(ctx context.Context, u *url.URL, header http.Header) error {
	req := (&http.Request{Method: http.MethodGet, URL: u, Header: http.Header{
		"Upgrade":               {"websocket"},
		"Connection":            {"Upgrade"},
		"Sec-WebSocket-Key":     {"nil"},
		"Sec-WebSocket-Version": {"13"},
	}}).WithContext(ctx)

	for k, vs := range header {
		if k == "Host" && len(vs) > 0 {
			req.Host = vs[0]
		} else {
			req.Header[k] = vs
		}
	}

	err := req.Write(ws.conn)
	if err != nil {
		return ws.checkClose(err)
	}

	res, err := http.ReadResponse(ws.r, req)
	if err != nil {
		return ws.checkClose(err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusSwitchingProtocols ||
		res.Header.Get("Sec-Websocket-Accept") != "Q67D9eATKx531lK8F7u2rqQNnNI=" {
		body, _ := ioutil.ReadAll(res.Body)
		return &ErrBadHandshake{
			Status: res.Status,
			Body:   string(body),
		}
	}

	return nil
}

func (ws *WebSocket) checkClose(err error) error {
	if err != nil {
		ws.close()
	}
	return err
}

// TODO: replace it with tls.Dialer once golang v1.15 is widely used.
type tlsDialer struct{}

func (d *tlsDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return tls.Dial(network, address, nil)
}
