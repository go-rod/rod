package cdp

import (
	"context"
	"errors"
	"testing"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

func Test(t *testing.T) {
	got.Each(t, T{})
}

type T struct {
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

func (t T) CancelCall() {
	cdp := New("")
	go func() {
		<-cdp.chReq
	}()
	cdp.ctx = t.Context()
	_, err := cdp.Call(t.Timeout(0), "", "", nil)
	t.Err(err)
}

func (t T) ReqErr() {
	ctx := t.Context()
	cdp := New("")
	cdp.ctx = ctx
	cdp.close = ctx.Cancel
	cdp.wsConn = &wsMockConn{
		send: func([]byte) error { return errors.New("err") },
	}

	go cdp.consumeMsg()

	_, err := cdp.Call(t.Context(), "", "", nil)
	t.Err(err)
}

func (t T) CancelOnReq() {
	ctx := t.Context()
	cdp := New("")
	cdp.ctx = ctx

	go func() {
		utils.Sleep(0.1)
		ctx.Cancel()
	}()

	_, err := cdp.Call(ctx, "", "", nil)
	t.Eq(err.Error(), "context canceled")

	go func() {
		utils.Sleep(0.1)
		ctx.Cancel()
	}()

	_, err = cdp.Call(t.Context(), "", "", nil)
	t.Eq(err.Error(), "context canceled")
}

func (t T) CancelBeforeSend() {
	cdp := New("")
	cdp.ctx = t.Context()
	_, err := cdp.Call(t.Timeout(0), "", "", nil)
	t.Eq(err, context.DeadlineExceeded)
}

func (t T) CancelBeforeCallback() {
	cdp := New("")
	cdp.ctx = t.Context()

	ctx := t.Context()

	go func() {
		<-cdp.chReq
		ctx.Cancel()
	}()

	_, err := cdp.Call(ctx, "", "", nil)
	t.Eq(err.Error(), "context canceled")
}

func (t T) CancelOnCallback() {
	ctx := t.Context()
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
	ctx.Cancel()
}

func (t T) CancelOnReadRes() {
	ctx := t.Context()
	cdp := New("")
	cdp.ctx = ctx
	cdp.wsConn = &wsMockConn{
		read: func() ([]byte, error) {
			ctx.Cancel()
			return utils.MustToJSONBytes(&Response{
				ID:     1,
				Result: nil,
				Error:  nil,
			}), nil
		},
	}

	go cdp.readMsgFromBrowser()

	_, err := cdp.Call(t.Context(), "", "", nil)
	t.Err(err)
}

func (t T) CancelOnReadEvent() {
	ctx, cancel := context.WithCancel(t.Context())
	cdp := New("")
	cdp.ctx = ctx
	cdp.wsConn = &wsMockConn{
		read: func() ([]byte, error) {
			cancel()
			return utils.MustToJSONBytes(&Event{}), nil
		},
	}

	go cdp.readMsgFromBrowser()

	_, err := cdp.Call(t.Context(), "", "", nil)
	t.Err(err)
}
