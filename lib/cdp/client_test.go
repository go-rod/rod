package cdp_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestBasic(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	url := launcher.New().MustLaunch()

	client := cdp.New(url).Context(ctx, done).Websocket(nil).Header(http.Header{"test": {}}).MustConnect()

	defer func() {
		utils.E(client.Call(ctx, "", "Browser.close", nil))
	}()

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	utils.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]string{
		"url": "file://" + file,
	})
	utils.E(err)

	targetID := gjson.ParseBytes(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true, // if it's not set no response will return
	})
	utils.E(err)

	sessionID := gjson.ParseBytes(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	utils.E(err)

	_, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": "abc",
	})
	assert.Error(t, err)

	timeout, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	sleeper := func() utils.Sleeper {
		return utils.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	}

	// cancel call
	tmpCtx, tmpCancel := context.WithCancel(ctx)
	tmpCancel()
	_, err = client.Call(tmpCtx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `10`,
	})
	assert.EqualError(t, err, context.Canceled.Error())

	utils.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"expression": `document.querySelector('iframe')`,
		})

		return err == nil && gjson.ParseBytes(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"objectId": gjson.ParseBytes(res).Get("result.objectId").String(),
	})
	utils.E(err)

	frameId := gjson.ParseBytes(res).Get("node.frameId").String()

	timeout, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	utils.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		// we might need to recreate the world because world can be
		// destroyed after the frame is reloaded
		res, err = client.Call(ctx, sessionID, "Page.createIsolatedWorld", map[string]interface{}{
			"frameId": frameId,
		})
		utils.E(err)

		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"contextId":  gjson.ParseBytes(res).Get("executionContextId").Int(),
			"expression": `document.querySelector('h4')`,
		})

		return err == nil && gjson.ParseBytes(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.getOuterHTML", map[string]interface{}{
		"objectId": gjson.ParseBytes(res).Get("result.objectId").String(),
	})
	utils.E(err)

	assert.Equal(t, "<h4>it works</h4>", gjson.ParseBytes(res).Get("outerHTML").String())
}

func TestError(t *testing.T) {
	cdpErr := cdp.Error{10, "err", "data"}
	assert.Equal(t, "{\"code\":10,\"message\":\"err\",\"data\":\"data\"}", cdpErr.Error())

	assert.Panics(t, func() {
		cdp.New("").MustConnect()
	})
}

func TestCrash(t *testing.T) {
	ctx := context.Background()
	l := launcher.New()

	client := cdp.New(l.MustLaunch()).Debug(true).MustConnect()

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	utils.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]interface{}{
		"url": "file://" + file,
	})
	utils.E(err)

	targetID := gjson.ParseBytes(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	utils.E(err)

	sessionID := gjson.ParseBytes(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	utils.E(err)

	go func() {
		utils.Sleep(2)
		_, _ = client.Call(ctx, sessionID, "Browser.crash", nil)
	}()

	_, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":   `new Promise(() => {})`,
		"awaitPromise": true,
	})
	assert.Regexp(t, `context canceled`, err.Error())
}
