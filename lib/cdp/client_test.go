package cdp_test

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
	"github.com/ysmood/gson"
)

var loud = flag.Bool("loud", false, "log everything")

func Test(t *testing.T) {
	if !*loud {
		log.SetOutput(ioutil.Discard)
	}

	got.Each(t, T{})
}

type T struct {
	got.G
}

func (t T) Basic() {
	ctx := t.Context()

	url := launcher.New().MustLaunch()

	client := cdp.New(url).Websocket(nil).
		Logger(utils.Log(func(msg ...interface{}) { fmt.Sprintln(msg...) })).
		Header(http.Header{"test": {}}).MustConnect(ctx)

	defer func() {
		t.E(client.Call(ctx, "", "Browser.close", nil))
	}()

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	t.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]string{
		"url": "file://" + file,
	})
	t.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true, // if it's not set no response will return
	})
	t.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	t.E(err)

	_, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": "abc",
	})
	t.Err(err)

	timeout := t.Context()

	sleeper := func() utils.Sleeper {
		return utils.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	}

	// cancel call
	tmpCtx, tmpCancel := context.WithCancel(ctx)
	tmpCancel()
	_, err = client.Call(tmpCtx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression": `10`,
	})
	t.Eq(err.Error(), context.Canceled.Error())

	t.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"expression": `document.querySelector('iframe')`,
		})

		return err == nil && gson.New(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"objectId": gson.New(res).Get("result.objectId").String(),
	})
	t.E(err)

	frameId := gson.New(res).Get("node.frameId").String()

	timeout = t.Context()

	t.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		// we might need to recreate the world because world can be
		// destroyed after the frame is reloaded
		res, err = client.Call(ctx, sessionID, "Page.createIsolatedWorld", map[string]interface{}{
			"frameId": frameId,
		})
		t.E(err)

		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"contextId":  gson.New(res).Get("executionContextId").Int(),
			"expression": `document.querySelector('h4')`,
		})

		return err == nil && gson.New(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.getOuterHTML", map[string]interface{}{
		"objectId": gson.New(res).Get("result.objectId").String(),
	})
	t.E(err)

	t.Eq("<h4>it works</h4>", gson.New(res).Get("outerHTML").String())
}

func (t T) Error() {
	cdpErr := cdp.Error{10, "err", "data"}
	t.Eq(cdpErr.Error(), "{10 err data}")

	t.Panic(func() {
		cdp.New("").MustConnect(t.Context())
	})
}

func (t T) NewWithLogger() {

	t.Panic(func() {
		cdp.New("").MustConnect(t.Context())
	})
}

func (t T) Crash() {
	ctx := t.Context()
	l := launcher.New()

	client := cdp.New(l.MustLaunch()).Logger(utils.LoggerQuiet).MustConnect(ctx)

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	t.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]interface{}{
		"url": "file://" + file,
	})
	t.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	t.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	t.E(err)

	go func() {
		utils.Sleep(2)
		_, _ = client.Call(ctx, sessionID, "Browser.crash", nil)
	}()

	_, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":   `new Promise(() => {})`,
		"awaitPromise": true,
	})
	t.Regex(`context canceled`, err.Error())
}
