package cdp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
)

type wsMockConn struct {
	send func([]byte) error
	read func() ([]byte, error)
}

func (c *wsMockConn) Send(b []byte) error {
	return c.send(b)
}

func (c *wsMockConn) Read() ([]byte, error) {
	return c.read()
}

func TestCancelCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cdp := New("").Context(ctx, cancel)
	go func() {
		<-cdp.chReq
	}()
	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.Error(t, err)
}

func TestReqErr(t *testing.T) {
	cdp := New("")
	cdp.wsConn = &wsMockConn{
		send: func([]byte) error { return errors.New("err") },
	}

	go cdp.consumeMsg()

	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.Error(t, err)
}

func TestCancelOnReq(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cdp := New("")

	go func() {
		kit.Sleep(0.1)
		cancel()
	}()

	_, err := cdp.Call(ctx, "", "", nil)
	assert.EqualError(t, err, "context canceled")

	go func() {
		kit.Sleep(0.1)
		cdp.ctxCancel()
	}()

	_, err = cdp.Call(context.Background(), "", "", nil)
	assert.EqualError(t, err, "context canceled")
}

func TestCancelBeforeSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cdp := New("")

	go func() {
		<-cdp.chReq
		cancel()
	}()

	_, err := cdp.Call(ctx, "", "", nil)
	assert.EqualError(t, err, "context canceled")
}

func TestCancelOnCallback(t *testing.T) {
	cdp := New("")

	go cdp.consumeMsg()

	cdp.callbacks.Store(uint64(1), make(chan *Response))
	cdp.chRes <- &Response{
		ID:     1,
		Result: nil,
		Error:  nil,
	}
	kit.Sleep(0.1)
	cdp.ctxCancel()
}

func TestCancelOnReadRes(t *testing.T) {
	cdp := New("")
	cdp.wsConn = &wsMockConn{
		read: func() ([]byte, error) {
			cdp.ctxCancel()
			return kit.MustToJSONBytes(&Response{
				ID:     1,
				Result: nil,
				Error:  nil,
			}), nil
		},
	}

	go cdp.readMsgFromBrowser()

	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.Error(t, err)
}

func TestCancelOnReadEvent(t *testing.T) {
	cdp := New("")
	cdp.wsConn = &wsMockConn{
		read: func() ([]byte, error) {
			cdp.ctxCancel()
			return kit.MustToJSONBytes(&Event{}), nil
		},
	}

	go cdp.readMsgFromBrowser()

	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.Error(t, err)
}
