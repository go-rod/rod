package cdp_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/launcher"
)

func TestBasic(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	ob := kit.NewObservable()

	url := os.Getenv("chrome")
	_, err := launcher.GetWebSocketDebuggerURL(url)
	if err != nil {
		url = launcher.New().Launch()
	}

	client, err := cdp.New(ctx, url)
	kit.E(err)

	go func() {
		for msg := range client.Event() {
			ob.Publish(msg)
		}
	}()

	defer func() {
		kit.E(client.Call(ctx, &cdp.Request{Method: "Browser.close"}))
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	kit.E(err)

	res, err := client.Call(ctx, &cdp.Request{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": "file://" + file,
		},
	})
	kit.E(err)

	targetID := res.Get("targetId").String()

	res, err = client.Call(ctx, &cdp.Request{
		Method: "Target.attachToTarget",
		Params: cdp.Object{
			"targetId": targetID,
			"flatten":  true, // if it's not set no response will return
		},
	})
	kit.E(err)

	sessionID := res.Get("sessionId").String()

	_, err = client.Call(ctx, &cdp.Request{
		SessionID: sessionID,
		Method:    "Page.enable",
	})
	kit.E(err)

	_, err = client.Call(ctx, &cdp.Request{
		Method: "Target.attachToTarget",
		Params: cdp.Object{
			"targetId": "abc",
		},
	})
	assert.Error(t, err)

	timeout, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	sleeper := func() kit.Sleeper {
		return kit.MergeSleepers(
			kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil),
			func(ctx context.Context) error {
				_, err := ob.Until(ctx, func(_ kit.Event) bool {
					return true
				})
				return err
			},
		)
	}

	kit.E(kit.Retry(timeout, sleeper(), func() (bool, error) {
		res, err = client.Call(ctx, &cdp.Request{
			SessionID: sessionID,
			Method:    "Runtime.evaluate",
			Params: cdp.Object{
				"expression": `document.querySelector('iframe')`,
			},
		})

		return err == nil && res.Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, &cdp.Request{
		SessionID: sessionID,
		Method:    "DOM.describeNode",
		Params: cdp.Object{
			"objectId": res.Get("result.objectId").String(),
		},
	})
	kit.E(err)

	frameId := res.Get("node.frameId").String()

	timeout, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	kit.E(kit.Retry(timeout, sleeper(), func() (bool, error) {
		// we might need to recreate the world because world can be
		// destroyed after the frame is reloaded
		res, err = client.Call(ctx, &cdp.Request{
			SessionID: sessionID,
			Method:    "Page.createIsolatedWorld",
			Params: cdp.Object{
				"frameId": frameId,
			},
		})
		kit.E(err)

		res, err = client.Call(ctx, &cdp.Request{
			SessionID: sessionID,
			Method:    "Runtime.evaluate",
			Params: cdp.Object{
				"contextId":  res.Get("executionContextId").Int(),
				"expression": `document.querySelector('h4')`,
			},
		})

		return err == nil && res.Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, &cdp.Request{
		SessionID: sessionID,
		Method:    "DOM.getOuterHTML",
		Params: cdp.Object{
			"objectId": res.Get("result.objectId").String(),
		},
	})
	kit.E(err)

	assert.Equal(t, "<h4>it works</h4>", res.Get("outerHTML").String())
}
