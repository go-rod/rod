package cdp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
	"github.com/ysmood/gotrace"
	"github.com/ysmood/gson"
)

var setup = got.Setup(nil)

func TestBasic(t *testing.T) {
	g := setup(t)

	ctx := g.Context()

	client := cdp.New().Logger(defaults.CDP).Start(cdp.MustConnectWS(launcher.New().MustLaunch()))

	defer func() {
		_, _ = client.Call(ctx, "", "Browser.close", nil)
	}()

	go func() {
		for range client.Event() {
			utils.Noop()
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	g.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]string{
		"url": "file://" + file,
	})
	g.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true, // if it's not set no response will return
	})
	g.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	g.E(err)

	_, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": "abc",
	})
	g.Err(err)

	timeout := g.Context()

	sleeper := func() utils.Sleeper {
		return utils.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	}

	// cancel call
	tmpCtx, tmpCancel := context.WithCancel(ctx)
	tmpCancel()
	_, err = client.Call(tmpCtx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `10`,
	})
	g.Eq(err.Error(), context.Canceled.Error())

	g.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"expression": `document.querySelector('iframe')`,
		})

		return err == nil && gson.New(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"objectId": gson.New(res).Get("result.objectId").String(),
	})
	g.E(err)

	frameID := gson.New(res).Get("node.frameId").String()

	timeout = g.Context()

	g.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		// we might need to recreate the world because world can be
		// destroyed after the frame is reloaded
		res, err = client.Call(ctx, sessionID, "Page.createIsolatedWorld", map[string]interface{}{
			"frameId": frameID,
		})
		g.E(err)

		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"contextId":  gson.New(res).Get("executionContextId").Int(),
			"expression": `document.querySelector('h4')`,
		})

		return err == nil && gson.New(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.getOuterHTML", map[string]interface{}{
		"objectId": gson.New(res).Get("result.objectId").String(),
	})
	g.E(err)

	g.Eq("<h4>it works</h4>", gson.New(res).Get("outerHTML").String())
}

func TestError(t *testing.T) {
	g := setup(t)

	cdpErr := cdp.Error{10, "err", "data"}
	g.Eq(cdpErr.Error(), "{10 err data}")
	g.True(cdpErr.Is(&cdpErr))

	g.Panic(func() {
		cdp.MustStartWithURL(context.Background(), "", nil)
	})
}

func TestCrash(t *testing.T) {
	g := setup(t)

	ctx := g.Context()

	client := cdp.MustStartWithURL(ctx, launcher.New().MustLaunch(), nil)

	go func() {
		for range client.Event() {
			utils.Noop()
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	g.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]interface{}{
		"url": "file://" + file,
	})
	g.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	g.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	g.E(err)

	go func() {
		utils.Sleep(1)
		_, err := client.Call(ctx, sessionID, "Browser.crash", nil)
		g.Eq(err, io.EOF)
	}()

	_, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":   `new Promise(() => {})`,
		"awaitPromise": true,
	})
	g.Eq(err, io.EOF)

	_, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `10`,
	})
	g.Has(err.Error(), "use of closed network connection")
}

func TestFormat(t *testing.T) {
	g := setup(t)

	g.Eq(cdp.Request{
		ID:        123,
		SessionID: "000000001234",
		Method:    "test",
		Params:    1,
	}.String(), `=> #123 @00000000 test 1`)

	g.Eq(cdp.Response{
		ID:     0,
		Result: []byte("11"),
	}.String(), "<= #0 11")

	g.Eq(cdp.Response{Error: &cdp.Error{}}.String(), `<= #0 error: {"code":0,"message":"","data":""}`)

	g.Eq(cdp.Event{
		Method: "event",
		Params: []byte("11"),
	}.String(), `<- @00000000 event 11`)
}

func TestSlowSend(t *testing.T) {
	g := setup(t)

	gotrace.CheckLeak(g, 0)

	id := 0
	wait := make(chan int)

	ws := &MockWebSocket{
		send: func([]byte) error {
			close(wait)
			utils.Sleep(0.3)
			return nil
		},
		read: func() ([]byte, error) {
			if id > 0 {
				return nil, io.EOF
			}

			id++
			<-wait

			return json.Marshal(cdp.Response{
				ID:     id,
				Result: json.RawMessage("1"),
				Error:  nil,
			})
		},
	}

	c := cdp.New().Start(ws)
	_, err := c.Call(g.Context(), "1234567890", "method", 1)
	g.E(err)
}

func TestCancelCallLeak(t *testing.T) {
	g := setup(t)

	gotrace.CheckLeak(g, 0)

	for i := 0; i < 30; i++ {
		id := 0
		wait := make(chan int)

		ws := &MockWebSocket{
			send: func([]byte) error {
				close(wait)
				utils.Sleep(0.01)
				return nil
			},
			read: func() ([]byte, error) {
				if id > 0 {
					return nil, io.EOF
				}

				id++
				<-wait

				return json.Marshal(cdp.Response{
					ID:     id,
					Result: json.RawMessage("1"),
					Error:  nil,
				})
			},
		}

		c := cdp.New().Start(ws)
		ctx := g.Context()
		ctx.Cancel()
		_, _ = c.Call(ctx, "1234567890", "method", 1)
	}
}

func TestConcurrentCall(t *testing.T) {
	g := setup(t)

	gotrace.CheckLeak(g, 0)

	req := make(chan []byte, 30)
	t.Cleanup(func() { close(req) })

	ws := &MockWebSocket{
		send: func(data []byte) error {
			req <- data
			return nil
		},
		read: func() ([]byte, error) {
			data, ok := <-req
			if !ok {
				return nil, io.EOF
			}

			var req cdp.Request
			err := json.Unmarshal(data, &req)
			if err != nil {
				return nil, err
			}

			return json.Marshal(cdp.Response{
				ID:     req.ID,
				Result: json.RawMessage(gson.New(req.Params).JSON("", "")),
				Error:  nil,
			})
		},
	}

	c := cdp.New().Start(ws)

	for i := 0; i < 1000; i++ {
		i := i
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			g := setup(t)
			g.Parallel()

			res, err := c.Call(g.Context(), "1234567890", "method", i)
			g.E(err)
			g.Eq(gson.New(res).Int(), i)
		})
	}
}

func TestMassBrowserClose(t *testing.T) {
	t.Skip()

	g := setup(t)
	s := g.Serve()

	for i := 0; i < 50; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			browser := rod.New().MustConnect()
			browser.MustPage(s.URL()).MustWaitLoad().MustClose()
			browser.MustClose()
		})
	}
}

type MockWebSocket struct {
	send func(data []byte) error
	read func() ([]byte, error)
}

func (c *MockWebSocket) Send(data []byte) error {
	return c.send(data)
}

func (c *MockWebSocket) Read() ([]byte, error) {
	return c.read()
}
