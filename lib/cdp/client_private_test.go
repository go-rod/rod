package cdp

import (
	"context"
	"errors"
	"testing"

	"github.com/go-rod/rod/lib/utils"
	"github.com/stretchr/testify/assert"
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
		utils.Sleep(0.1)
		cancel()
	}()

	_, err := cdp.Call(ctx, "", "", nil)
	assert.EqualError(t, err, "context canceled")

	go func() {
		utils.Sleep(0.1)
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
	utils.Sleep(0.1)
	cdp.ctxCancel()
}

func TestCancelOnReadRes(t *testing.T) {
	cdp := New("")
	cdp.wsConn = &wsMockConn{
		read: func() ([]byte, error) {
			cdp.ctxCancel()
			return utils.MustToJSONBytes(&Response{
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
			return utils.MustToJSONBytes(&Event{}), nil
		},
	}

	go cdp.readMsgFromBrowser()

	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.Error(t, err)
}
