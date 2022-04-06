package cdp_test

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

func TestMain(m *testing.M) {
	if !*loud {
		log.SetOutput(ioutil.Discard)
	}

	os.Exit(m.Run())
}

var setup = got.Setup(nil)

func TestBasic(t *testing.T) {
	g := setup(t)

	ctx := g.Context()

	url := launcher.New().MustLaunch()

	client := cdp.New(url).Websocket(nil).
		Logger(utils.Log(func(msg ...interface{}) { fmt.Sprintln(msg...) })).
		Header(http.Header{"test": {}}).MustConnect(ctx)

	defer func() {
		_, _ = client.Call(ctx, "", "Browser.close", nil)
	}()

	go func() {
		for range client.Event() {
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

func TestTestError(t *testing.T) {
	g := setup(t)

	cdpErr := cdp.Error{10, "err", "data"}
	g.Eq(cdpErr.Error(), "{10 err data}")

	g.Panic(func() {
		cdp.New("").MustConnect(g.Context())
	})
}

func TestNewWithLogger(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		cdp.New("").MustConnect(g.Context())
	})
}

func TestCrash(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
	l := launcher.New()

	client := cdp.New(l.MustLaunch()).Logger(utils.LoggerQuiet).MustConnect(ctx)

	go func() {
		for range client.Event() {
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
		utils.Sleep(2)
		_, _ = client.Call(ctx, sessionID, "Browser.crash", nil)
	}()

	_, err = client.Call(ctx, sessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":   `new Promise(() => {})`,
		"awaitPromise": true,
	})
	g.Is(err, cdp.ErrConnClosed)
	g.Eq(err.Error(), "cdp connection closed: EOF")
}
