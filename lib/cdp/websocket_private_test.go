package cdp

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/url"
	"testing"
	"time"
)

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

	g.Err(ws.handshake(g.Timeout(0), nil, nil))

	mc.errOnCount = 1
	g.Err(ws.handshake(g.Context(), u, nil))

	tls := &tlsDialer{}
	g.Err(tls.DialContext(context.Background(), "", ""))
}

type MockConn struct {
	errOnCount int
	frame      []byte
}

func (c *MockConn) Read(b []byte) (n int, err error) {
	if c.errOnCount == 0 {
		return 0, errors.New("err")
	}
	c.errOnCount--
	return copy(b, c.frame), nil
}

func (c *MockConn) Write(b []byte) (n int, err error) {
	if c.errOnCount == 0 {
		return 0, errors.New("err")
	}
	c.errOnCount--
	return len(b), nil
}

func (c *MockConn) Close() error {
	return nil
}

func (c *MockConn) LocalAddr() net.Addr {
	return nil
}

func (c *MockConn) RemoteAddr() net.Addr {
	return nil
}

func (c *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}
