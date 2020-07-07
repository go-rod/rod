package cdp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

type wsWriteErrConn struct {
}

func (c *wsWriteErrConn) Send(_ []byte) error {
	return errors.New("err")
}

func (c *wsWriteErrConn) Read() ([]byte, error) {
	return nil, nil
}

func TestWriteError(t *testing.T) {
	cdp := New("")
	cdp.wsConn = &wsWriteErrConn{}
	go cdp.consumeMsg()
	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.EqualError(t, err, "context canceled")
}
