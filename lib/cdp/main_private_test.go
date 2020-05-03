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
	cdp := New().Context(ctx)
	go func() {
		<-cdp.chReqMsg
	}()
	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.Error(t, err)
}

type wsWriteErr struct {
}

func (c *wsWriteErr) Send(_ []byte) error {
	return errors.New("err")
}

func (c *wsWriteErr) Read() ([]byte, error) {
	return nil, nil
}

func TestWriteError(t *testing.T) {
	cdp := New()
	cdp.ws = &wsWriteErr{}
	go cdp.consumeMsg()
	_, err := cdp.Call(context.Background(), "", "", nil)
	assert.EqualError(t, err, "err")
}
