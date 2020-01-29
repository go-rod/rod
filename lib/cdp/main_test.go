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
)

func TestBasic(t *testing.T) {
	ctx := context.Background()

	url := os.Getenv("chrome")
	_, err := cdp.GetWebSocketDebuggerURL(url)
	if err != nil {
		url, err = cdp.LaunchBrowser("", true)
		kit.E(err)
	}

	client, err := cdp.New(ctx, url)
	kit.E(err)

	go func() {
		panic(<-client.Fatal())
	}()

	go func() {
		for msg := range client.Event() {
			kit.Log(msg.Method)
		}
	}()

	defer func() {
		kit.E(client.Call(ctx, &cdp.Message{Method: "Browser.close"}))
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	kit.E(err)

	res, err := client.Call(ctx, &cdp.Message{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": "file://" + file,
		},
	})
	kit.E(err)

	targetID := res.Get("targetId").String()

	res, err = client.Call(ctx, &cdp.Message{
		Method: "Target.attachToTarget",
		Params: cdp.Object{
			"targetId": targetID,
			"flatten":  true, // if it's not set no response will return
		},
	})
	kit.E(err)

	_, err = client.Call(ctx, &cdp.Message{
		Method: "Target.attachToTarget",
		Params: cdp.Object{
			"targetId": "abc",
		},
	})
	assert.Error(t, err)

	sessionID := res.Get("sessionId").String()

	timeout, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	kit.E(cdp.Retry(timeout, func() error {
		res, err = client.Call(ctx, &cdp.Message{
			SessionID: sessionID,
			Method:    "Runtime.evaluate",
			Params: cdp.Object{
				"expression": `document.querySelector('iframe')`,
			},
		})

		if err != nil {
			return err
		}
		if res.Get("result.objectId").String() == "" {
			return cdp.ErrNotYet
		}

		return nil
	}))

	res, err = client.Call(ctx, &cdp.Message{
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

	kit.E(cdp.Retry(timeout, func() error {
		// we might need to recreate the world because world can be
		// destroyed after some frame events happens
		res, err = client.Call(ctx, &cdp.Message{
			SessionID: sessionID,
			Method:    "Page.createIsolatedWorld",
			Params: cdp.Object{
				"frameId": frameId,
			},
		})
		kit.E(err)

		res, err = client.Call(ctx, &cdp.Message{
			SessionID: sessionID,
			Method:    "Runtime.evaluate",
			Params: cdp.Object{
				"contextId":  res.Get("executionContextId").Int(),
				"expression": `document.querySelector('h4')`,
			},
		})
		if err != nil {
			return err
		}

		if res.Get("result.subtype").String() == "null" {
			return cdp.ErrNotYet
		}

		return nil
	}))

	res, err = client.Call(ctx, &cdp.Message{
		SessionID: sessionID,
		Method:    "DOM.getOuterHTML",
		Params: cdp.Object{
			"objectId": res.Get("result.objectId").String(),
		},
	})
	kit.E(err)

	assert.Equal(t, "<h4>it works</h4>", res.Get("outerHTML").String())
}
