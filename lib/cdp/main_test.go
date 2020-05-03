package cdp_test

import (
	"context"
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

	url := launcher.New().Launch()

	client := cdp.New().URL(url).Context(ctx).Websocket(nil).Connect()

	defer func() {
		kit.E(client.Call(ctx, "", "Browser.close", nil))
	}()

	go func() {
		for msg := range client.Event() {
			ob.Publish(msg)
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	kit.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]string{
		"url": "file://" + file,
	})
	kit.E(err)

	targetID := kit.JSON(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true, // if it's not set no response will return
	})
	kit.E(err)

	sessionID := kit.JSON(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	kit.E(err)

	_, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": "abc",
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

	// cancel call
	tmpCtx, tmpCancel := context.WithCancel(ctx)
	tmpCancel()
	_, err = client.Call(tmpCtx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `10`,
	})
	assert.EqualError(t, err, context.Canceled.Error())

	kit.E(kit.Retry(timeout, sleeper(), func() (bool, error) {
		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"expression": `document.querySelector('iframe')`,
		})

		return err == nil && kit.JSON(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"objectId": kit.JSON(res).Get("result.objectId").String(),
	})
	kit.E(err)

	frameId := kit.JSON(res).Get("node.frameId").String()

	timeout, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	kit.E(kit.Retry(timeout, sleeper(), func() (bool, error) {
		// we might need to recreate the world because world can be
		// destroyed after the frame is reloaded
		res, err = client.Call(ctx, sessionID, "Page.createIsolatedWorld", map[string]interface{}{
			"frameId": frameId,
		})
		kit.E(err)

		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"contextId":  kit.JSON(res).Get("executionContextId").Int(),
			"expression": `document.querySelector('h4')`,
		})

		return err == nil && kit.JSON(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.getOuterHTML", map[string]interface{}{
		"objectId": kit.JSON(res).Get("result.objectId").String(),
	})
	kit.E(err)

	assert.Equal(t, "<h4>it works</h4>", kit.JSON(res).Get("outerHTML").String())
}

func TestError(t *testing.T) {
	cdpErr := cdp.Error{10, "err", "data"}
	assert.Equal(t, "{\"code\":10,\"message\":\"err\",\"data\":\"data\"}", cdpErr.Error())

	assert.Panics(t, func() {
		cdp.New().Connect()
	})
}

func TestCrash(t *testing.T) {
	ctx := context.Background()
	l := launcher.New()

	client := cdp.New().URL(l.Launch()).Debug(true).Connect()

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	kit.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]interface{}{
		"url": "file://" + file,
	})
	kit.E(err)

	targetID := kit.JSON(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	kit.E(err)

	sessionID := kit.JSON(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	kit.E(err)

	go func() {
		kit.Sleep(2)
		_, _ = client.Call(ctx, sessionID, "Browser.crash", nil)
	}()

	_, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":   `new Promise(() => {})`,
		"awaitPromise": true,
	})
	assert.Regexp(t, `websocket: close 1006 \(abnormal closure\)|forcibly closed by the remote host`, err.Error())
}
