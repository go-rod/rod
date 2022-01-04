package cdp

import (
	"context"
	"errors"
	"io"
	"net/http"
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

type MockWebSocket struct {
	send func([]byte) error
	read func() ([]byte, error)
}

// Connect interface
func (c *MockWebSocket) Connect(ctx context.Context, url string, header http.Header) error {
	return nil
}

func (c *MockWebSocket) Send(b []byte) error {
	return c.send(b)
}

func (c *MockWebSocket) Read() ([]byte, error) {
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
	cdp.ws = &MockWebSocket{
		send: func([]byte) error { return errors.New("err") },
	}

	_, err := cdp.Call(t.Context(), "", "", nil)
	t.Err(err)
}

func (t T) CancelBeforeSend() {
	cdp := New("")
	cdp.ctx = t.Context()
	_, err := cdp.Call(t.Timeout(0), "", "", nil)
	t.Eq(err, context.DeadlineExceeded)
}

func (t T) CancelBeforeCallback() {
	ctx := t.Context()
	cdp := New("")
	cdp.ws = &MockWebSocket{
		read: func() ([]byte, error) {
			<-ctx.Done() // delay until send finishes
			utils.Sleep(0.1)
			return nil, errors.New("read failed")
		},
		send: func([]byte) error {
			ctx.Cancel() // cancel the request after send
			return nil
		},
	}
	cdp.MustConnect(ctx)

	_, err := cdp.Call(ctx, "", "", nil)
	t.Eq(err.Error(), "context canceled")
}

func (t T) CancelOnReadRes() {
	ctx := t.Context()
	cdp := New("")
	cdp.ws = &MockWebSocket{
		send: func(bytes []byte) error {
			return ctx.Err()
		},
		read: func() ([]byte, error) {
			ctx.Cancel()
			return utils.MustToJSONBytes(&Response{
				ID:     1,
				Result: nil,
				Error:  nil,
			}), nil
		},
	}
	cdp.MustConnect(ctx)

	_, err := cdp.Call(ctx, "", "", nil)
	t.Err(err)
}

func (t T) CallAfterBrowserDone() {
	ctx := t.Context()
	cdp := New("")
	cdp.ws = &MockWebSocket{
		send: func(bytes []byte) error { return io.EOF },
		read: func() ([]byte, error) { return nil, io.EOF },
	}
	cdp.MustConnect(ctx)
	utils.Sleep(0.1)

	_, err := cdp.Call(ctx, "", "", nil)
	t.Err(err)
	t.Is(err, io.EOF)
	t.Eq(err.Error(), "cdp connection closed: EOF")
}

func (t T) CancelOnReadEvent() {
	ctx, cancel := context.WithCancel(t.Context())
	cdp := New("")
	cdp.ws = &MockWebSocket{
		send: func(bytes []byte) error {
			return ctx.Err()
		},
		read: func() ([]byte, error) {
			cancel()
			return utils.MustToJSONBytes(&Event{}), nil
		},
	}
	cdp.MustConnect(ctx)

	_, err := cdp.Call(t.Context(), "", "", nil)
	t.Err(err)
}

func (t T) TestError() {
	t.Is(&Error{Code: -123}, &Error{Code: -123})
}

func (t T) PendingRequests() {
	pending := newPendingRequests()

	err := pending.add(1, newPendingRequest())
	t.Nil(err)
	err = pending.add(2, newPendingRequest())
	t.Nil(err)
	pending.fulfill(1, &Response{})

	// resolving something where no-one is waiting is fine
	pending.delete(2)
	pending.fulfill(2, &Response{})
	pending.fulfill(3, &Response{})

	pending.close(io.EOF)
	pending.close(errors.New("this will be ignored"))

	err = pending.add(3, newPendingRequest())
	t.Is(err, io.EOF)
	t.Err(err)

	pending = newPendingRequests()
	pending.close(nil)
	err = pending.add(3, newPendingRequest())
	t.Err(err)
	t.Eq(err.Error(), "browser has shut down")
}
