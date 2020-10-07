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

	got.Each(t, C{})
}

type C struct {
	got.G
}

func (c C) Basic() {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	url := launcher.New().MustLaunch()

	client := cdp.New(url).Websocket(nil).
		Logger(utils.Log(func(msg ...interface{}) { fmt.Sprintln(msg...) })).
		Header(http.Header{"test": {}}).MustConnect(ctx)

	defer func() {
		c.E(client.Call(ctx, "", "Browser.close", nil))
	}()

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	c.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]string{
		"url": "file://" + file,
	})
	c.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true, // if it's not set no response will return
	})
	c.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	c.E(err)

	_, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": "abc",
	})
	c.Err(err)

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
	c.Eq(err.Error(), context.Canceled.Error())

	c.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"expression": `document.querySelector('iframe')`,
		})

		return err == nil && gson.New(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.describeNode", map[string]interface{}{
		"objectId": gson.New(res).Get("result.objectId").String(),
	})
	c.E(err)

	frameId := gson.New(res).Get("node.frameId").String()

	timeout, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c.E(utils.Retry(timeout, sleeper(), func() (bool, error) {
		// we might need to recreate the world because world can be
		// destroyed after the frame is reloaded
		res, err = client.Call(ctx, sessionID, "Page.createIsolatedWorld", map[string]interface{}{
			"frameId": frameId,
		})
		c.E(err)

		res, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
			"contextId":  gson.New(res).Get("executionContextId").Int(),
			"expression": `document.querySelector('h4')`,
		})

		return err == nil && gson.New(res).Get("result.subtype").String() != "null", nil
	}))

	res, err = client.Call(ctx, sessionID, "DOM.getOuterHTML", map[string]interface{}{
		"objectId": gson.New(res).Get("result.objectId").String(),
	})
	c.E(err)

	c.Eq("<h4>it works</h4>", gson.New(res).Get("outerHTML").String())
}

func (c C) Error() {
	cdpErr := cdp.Error{10, "err", "data"}
	c.Eq(cdpErr.Error(), "{10 err data}")

	c.Panic(func() {
		cdp.New("").MustConnect(context.Background())
	})
}

func (c C) NewWithLogger() {

	c.Panic(func() {
		cdp.New("").MustConnect(context.Background())
	})
}

func (c C) Crash() {
	ctx := context.Background()
	l := launcher.New()

	client := cdp.New(l.MustLaunch()).Logger(utils.LoggerQuiet).MustConnect(ctx)

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/iframe.html"))
	c.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]interface{}{
		"url": "file://" + file,
	})
	c.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	c.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	_, err = client.Call(ctx, sessionID, "Page.enable", nil)
	c.E(err)

	go func() {
		utils.Sleep(2)
		_, _ = client.Call(ctx, sessionID, "Browser.crash", nil)
	}()

	_, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":   `new Promise(() => {})`,
		"awaitPromise": true,
	})
	c.Regex(`context canceled`, err.Error())
}
