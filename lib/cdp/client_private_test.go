package cdp

import (
	"context"
	"errors"
	"testing"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

func Test(t *testing.T) {
	got.Each(t, C{})
}

type C struct {
	got.G
}

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

func (c C) CancelCall() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cdp := New("")
	go func() {
		<-cdp.chReq
	}()
	cdp.ctx = context.Background()
	_, err := cdp.Call(ctx, "", "", nil)
	c.Err(err)
}

func (c C) ReqErr() {
	ctx, cancel := context.WithCancel(context.Background())
	cdp := New("")
	cdp.ctx = ctx
	cdp.close = cancel
	cdp.wsConn = &wsMockConn{
		send: func([]byte) error { return errors.New("err") },
	}

	go cdp.consumeMsg()

	_, err := cdp.Call(context.Background(), "", "", nil)
	c.Err(err)
}

func (c C) CancelOnReq() {
	ctx, cancel := context.WithCancel(context.Background())
	cdp := New("")
	cdp.ctx = ctx

	go func() {
		utils.Sleep(0.1)
		cancel()
	}()

	_, err := cdp.Call(ctx, "", "", nil)
	c.Eq(err.Error(), "context canceled")

	go func() {
		utils.Sleep(0.1)
		cancel()
	}()

	_, err = cdp.Call(context.Background(), "", "", nil)
	c.Eq(err.Error(), "context canceled")
}

func (c C) CancelBeforeSend() {
	cdp := New("")
	cdp.ctx = context.Background()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cdp.Call(ctx, "", "", nil)
	c.Eq(err.Error(), "context canceled")
}

func (c C) CancelBeforeCallback() {
	cdp := New("")
	cdp.ctx = context.Background()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-cdp.chReq
		cancel()
	}()

	_, err := cdp.Call(ctx, "", "", nil)
	c.Eq(err.Error(), "context canceled")
}

func (c C) CancelOnCallback() {
	ctx, cancel := context.WithCancel(context.Background())
	cdp := New("")
	cdp.ctx = ctx

	go cdp.consumeMsg()

	cdp.callbacks.Store(1, make(chan *Response))
	cdp.chRes <- &Response{
		ID:     1,
		Result: nil,
		Error:  nil,
	}
	utils.Sleep(0.1)
	cancel()
}

func (c C) CancelOnReadRes() {
	ctx, cancel := context.WithCancel(context.Background())
	cdp := New("")
	cdp.ctx = ctx
	cdp.wsConn = &wsMockConn{
		read: func() ([]byte, error) {
			cancel()
			return utils.MustToJSONBytes(&Response{
				ID:     1,
				Result: nil,
				Error:  nil,
			}), nil
		},
	}

	go cdp.readMsgFromBrowser()

	_, err := cdp.Call(context.Background(), "", "", nil)
	c.Err(err)
}

func (c C) CancelOnReadEvent() {
	ctx, cancel := context.WithCancel(context.Background())
	cdp := New("")
	cdp.ctx = ctx
	cdp.wsConn = &wsMockConn{
		read: func() ([]byte, error) {
			cancel()
			return utils.MustToJSONBytes(&Event{}), nil
		},
	}

	go cdp.readMsgFromBrowser()

	_, err := cdp.Call(context.Background(), "", "", nil)
	c.Err(err)
}
