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

var setup = got.Setup(nil)

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

func TestCancelCall(t *testing.T) {
	g := setup(t)

	cdp := New("")
	go func() {
		<-cdp.chReq
	}()
	cdp.ctx = g.Context()
	_, err := cdp.Call(g.Timeout(0), "", "", nil)
	g.Err(err)
}

func TestReqErr(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
	cdp := New("")
	cdp.ctx = ctx
	cdp.close = ctx.Cancel
	cdp.ws = &MockWebSocket{
		send: func([]byte) error { return errors.New("err") },
	}

	_, err := cdp.Call(g.Context(), "", "", nil)
	g.Err(err)
}

func TestCancelBeforeSend(t *testing.T) {
	g := setup(t)

	cdp := New("")
	cdp.ctx = g.Context()
	_, err := cdp.Call(g.Timeout(0), "", "", nil)
	g.Eq(err, context.DeadlineExceeded)
}

func TestCancelBeforeCallback(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
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
	g.Eq(err.Error(), "context canceled")
}

func TestCancelOnReadRes(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
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
	g.Err(err)
}

func TestCallAfterBrowserDone(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
	cdp := New("")
	cdp.ws = &MockWebSocket{
		send: func(bytes []byte) error { return io.EOF },
		read: func() ([]byte, error) { return nil, io.EOF },
	}
	cdp.MustConnect(ctx)
	utils.Sleep(0.1)

	_, err := cdp.Call(ctx, "", "", nil)
	g.Err(err)
	g.Is(err, io.EOF)
	g.Eq(err.Error(), "cdp connection closed: EOF")
}

func TestCancelOnReadEvent(t *testing.T) {
	g := setup(t)

	ctx, cancel := context.WithCancel(g.Context())
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

	_, err := cdp.Call(g.Context(), "", "", nil)
	g.Err(err)
}

func TestTestError(t *testing.T) {
	g := setup(t)

	g.Is(&Error{Code: -123}, &Error{Code: -123})
}

func TestPendingRequests(t *testing.T) {
	g := setup(t)

	pending := newPendingRequests()

	err := pending.add(1, newPendingRequest())
	g.Nil(err)
	err = pending.add(2, newPendingRequest())
	g.Nil(err)
	pending.fulfill(1, &Response{})

	// resolving something where no-one is waiting is fine
	pending.delete(2)
	pending.fulfill(2, &Response{})
	pending.fulfill(3, &Response{})

	pending.close(io.EOF)
	pending.close(errors.New("this will be ignored"))

	err = pending.add(3, newPendingRequest())
	g.Is(err, io.EOF)
	g.Err(err)

	pending = newPendingRequests()
	pending.close(nil)
	err = pending.add(3, newPendingRequest())
	g.Err(err)
	g.Eq(err.Error(), "browser has shut down")
}
