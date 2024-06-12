package cdp

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
)

var _ WebSocketable = &WebSocket{}

// WebSocket client for chromium. It only implements a subset of WebSocket protocol.
// Both the Read and Write are thread-safe.
// Limitation: https://bugs.chromium.org/p/chromium/issues/detail?id=1069431
// Ref: https://tools.ietf.org/html/rfc6455
type WebSocket struct {
	// Dialer is usually used for proxy
	Dialer Dialer

	lock sync.Mutex
	conn net.Conn
	r    *bufio.Reader
}

// Connect to browser.
func (ws *WebSocket) Connect(ctx context.Context, wsURL string, header http.Header) error {
	if ws.conn != nil {
		panic("duplicated connection: " + wsURL)
	}

	u, err := url.Parse(wsURL)
	if err != nil {
		return err
	}

	ws.initDialer(u)

	conn, err := ws.Dialer.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return err
	}

	ws.conn = conn
	ws.r = bufio.NewReader(conn)
	return ws.handshake(ctx, u, header)
}

// Close the underlying connection.
func (ws *WebSocket) Close() error {
	return ws.conn.Close()
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

// Send a message to browser.
// Because we use zero-copy design, it will modify the content of the msg.
// It won't allocate new memory.
func (ws *WebSocket) Send(msg []byte) error {
	err := ws.send(msg)
	if err != nil {
		_ = ws.Close()
	}
	return err
}

func (ws *WebSocket) send(msg []byte) error {
	// FIN is alway true, Opcode is always text frame.
	header := [18]byte{0b1000_0001, 0b1000_0000}
	mask := []byte{0, 1, 2, 3}

	size := len(msg)
	fieldLen := 0
	switch {
	case size <= 125:
		header[1] |= byte(size)
	case size < 65536:
		header[1] |= 126
		fieldLen = 2
	default:
		header[1] |= 127
		fieldLen = 8
	}

	var i int
	for i = 0; i < fieldLen; i++ {
		digit := (fieldLen - i - 1) * 8
		header[i+2] = byte((size >> digit) & 0xff)
	}

	copy(header[i+2:], mask)

	for i := range msg {
		msg[i] ^= mask[i%4]
	}

	data := make([]byte, i+6+len(msg))
	copy(data, header[:i+6])
	copy(data[i+6:], msg)

	_, err := ws.conn.Write(data)
	return err
}

// Read a message from browser.
func (ws *WebSocket) Read() ([]byte, error) {
	b, err := ws.read()
	if err != nil {
		_ = ws.Close()
		return nil, err
	}
	return b, nil
}

func (ws *WebSocket) read() ([]byte, error) {
	ws.lock.Lock()
	defer ws.lock.Unlock()

	_, err := ws.r.ReadByte()
	if err != nil {
		return nil, err
	}

	b, err := ws.r.ReadByte()
	if err != nil {
		return nil, err
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
			return nil, err
		}

		size = size<<8 + int(b)
	}

	data := make([]byte, size)
	_, err = io.ReadFull(ws.r, data)
	return data, err
}

// BadHandshakeError type.
type BadHandshakeError struct {
	Status string
	Body   string
}

func (e *BadHandshakeError) Error() string {
	return fmt.Sprintf(
		"websocket bad handshake: %s. %s",
		e.Status, e.Body,
	)
}

func verifyWebSocketAccept(responseHeaders http.Header, websocketKey string) bool {
	expectedKey := websocketKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hash := sha1.New()
	hash.Write([]byte(expectedKey))
	expectedAccept := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	return responseHeaders.Get("Sec-WebSocket-Accept") == expectedAccept
}

func (ws *WebSocket) handshake(ctx context.Context, u *url.URL, header http.Header) error {
	defaultSecKey := "nil"
	req := (&http.Request{Method: http.MethodGet, URL: u, Header: http.Header{
		"Upgrade":               {"websocket"},
		"Connection":            {"Upgrade"},
		"Sec-WebSocket-Key":     {defaultSecKey},
		"Sec-WebSocket-Version": {"13"},
	}}).WithContext(ctx)

	secKey := defaultSecKey
	for k, vs := range header {
		switch {
		case k == "Host" && len(vs) > 0:
			req.Host = vs[0]
		case k == "Sec-WebSocket-Key" && len(vs) > 0:
			secKey = vs[0]
			req.Header[k] = vs
		default:
			req.Header[k] = vs
		}
	}

	err := req.Write(ws.conn)
	if err != nil {
		return err
	}

	res, err := http.ReadResponse(ws.r, req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusSwitchingProtocols || !verifyWebSocketAccept(res.Header, secKey) {
		body, _ := io.ReadAll(res.Body)
		return &BadHandshakeError{
			Status: res.Status,
			Body:   string(body),
		}
	}

	return nil
}
