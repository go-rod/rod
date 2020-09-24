package launcher

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
	"github.com/stretchr/testify/assert"
)

func TestToHTTP(t *testing.T) {
	u, _ := url.Parse("wss://a.com")
	assert.Equal(t, "https", toHTTP(*u).Scheme)

	u, _ = url.Parse("ws://a.com")
	assert.Equal(t, "http", toHTTP(*u).Scheme)
}

func TestToWS(t *testing.T) {
	u, _ := url.Parse("https://a.com")
	assert.Equal(t, "wss", toWS(*u).Scheme)

	u, _ = url.Parse("http://a.com")
	assert.Equal(t, "ws", toWS(*u).Scheme)
}

func TestUnzip(t *testing.T) {
	assert.Error(t, unzip(ioutil.Discard, "", ""))
}

func TestLaunchOptions(t *testing.T) {
	defaults.Show = true
	isInDocker = true

	// recover
	defer func() {
		defaults.ResetWithEnv()
		isInDocker = utils.FileExists("/.dockerenv")
	}()

	l := New()

	_, has := l.Get("headless")
	assert.False(t, has)

	_, has = l.Get("no-sandbox")
	assert.True(t, has)
}

func TestGetURLErr(t *testing.T) {
	l := New()

	l.ctxCancel()
	_, err := l.getURL()
	assert.Error(t, err)

	l = New()
	l.parser.Buffer = "err"
	close(l.exit)
	_, err = l.getURL()
	assert.Equal(t, "[launcher] Failed to get the debug url err", err.Error())
}

func TestRemoteLaunch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u, mux, close := utils.Serve("")
	defer close()

	mux.Handle("/", NewRemoteLauncher())

	l := MustNewRemote(u).KeepUserDataDir().Delete(flagKeepUserDataDir)
	client := l.Client()
	b := client.MustConnect(ctx)
	utils.E(b.Call(ctx, "", "Browser.getVersion", nil))
	utils.Sleep(1)
	_, _ = b.Call(ctx, "", "Browser.crash", nil)
	dir, _ := l.Get("user-data-dir")

	for ctx.Err() == nil {
		utils.Sleep(0.1)
		_, err := os.Stat(dir)
		if err != nil {
			break
		}
	}
	assert.NoDirExists(t, dir)
}

func TestLaunchErrs(t *testing.T) {
	l := New().Bin("echo")
	go func() {
		l.exit <- struct{}{}
	}()
	_, err := l.Launch()
	assert.Error(t, err)

	l = New()
	l.browser.Dir = utils.RandString(8)
	l.browser.ExecSearchMap = nil
	l.browser.Hosts = []string{}
	_, err = l.Launch()
	assert.Error(t, err)
}
