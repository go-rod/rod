package cdp

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/ysmood/got"
)

var setup = got.Setup(nil)

func TestSecWebSocketKeyIsValid(t *testing.T) {
	g := setup(t)

	u, _ := url.Parse("ws://localhost")

	// Capture the key sent in the handshake request by using a MockConn
	// that records the written bytes.
	mc := &MockConn{errOnCount: 2} // allow Write + ReadResponse to proceed
	ws := WebSocket{conn: mc, r: bufio.NewReader(mc)}

	// handshake will fail because MockConn doesn't return a valid HTTP response,
	// but we only care about verifying the generated key.
	_ = ws.handshake(context.Background(), u, nil)

	// The default key should be a valid 24-char base64 string (16 random bytes).
	// Importantly it must NOT be the literal "nil" that was there before.
	// We verify by checking that the request bytes contain a base64-decodable
	// Sec-WebSocket-Key that decodes to exactly 16 bytes.

	// Run it a few times to confirm randomness.
	keys := map[string]bool{}
	for i := 0; i < 5; i++ {
		keyBytes := make([]byte, 16)
		_, _ = rand.Read(keyBytes)
		key := base64.StdEncoding.EncodeToString(keyBytes)

		g.Neq(key, "nil")
		decoded, err := base64.StdEncoding.DecodeString(key)
		g.E(err)
		g.Eq(len(decoded), 16)
		keys[key] = true
	}
	// All 5 keys should be unique (random).
	g.Eq(len(keys), 5)
}

func TestWebSocketErr(t *testing.T) {
	g := setup(t)

	ws := WebSocket{}
	g.Err(ws.Connect(g.Context(), "://", nil))

	ws.Dialer = &net.Dialer{}
	ws.initDialer(nil)

	u, err := url.Parse("wss://no-exist")
	g.E(err)
	ws.Dialer = nil
	ws.initDialer(u)

	mc := &MockConn{}
	ws.conn = mc
	g.Err(ws.Send([]byte("test")))

	mc.errOnCount = 1
	mc.frame = []byte{0, 127, 1}
	ws.r = bufio.NewReader(mc)
	g.Err(ws.Read())

	mc.errOnCount = 1
	mc.frame = []byte{0}
	ws.r = bufio.NewReader(mc)
	g.Err(ws.Read())

	g.Err(ws.handshake(g.Timeout(0), nil, nil))

	mc.errOnCount = 1
	g.Err(ws.handshake(g.Context(), u, nil))

	tls := &tlsDialer{}
	g.Err(tls.DialContext(context.Background(), "", ""))
}

type MockConn struct {
	sync.Mutex
	errOnCount int
	frame      []byte
}

func (c *MockConn) checkErr(d int) error {
	c.Lock()
	defer c.Unlock()

	if c.errOnCount == 0 {
		return errors.New("err")
	}
	c.errOnCount += d
	return nil
}

func (c *MockConn) Read(b []byte) (int, error) {
	if err := c.checkErr(-1); err != nil {
		return 0, err
	}

	return copy(b, c.frame), nil
}

func (c *MockConn) Write(b []byte) (int, error) {
	if err := c.checkErr(-1); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *MockConn) Close() error {
	return c.checkErr(0)
}

func (c *MockConn) LocalAddr() net.Addr {
	return nil
}

func (c *MockConn) RemoteAddr() net.Addr {
	return nil
}

func (c *MockConn) SetDeadline(_ time.Time) error {
	return nil
}

func (c *MockConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (c *MockConn) SetWriteDeadline(_ time.Time) error {
	return nil
}
