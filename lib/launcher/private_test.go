package launcher

import (
	"context"
	"net/url"
	"testing"

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
	assert.Error(t, unzip("", ""))
}

func TestLaunchOptions(t *testing.T) {
	oldShow := defaults.Show
	oldIsInDocker := isInDocker
	defer func() {
		defaults.Show = oldShow
		isInDocker = oldIsInDocker
	}()

	defaults.Show = true
	isInDocker = true

	l := New()

	_, has := l.Get("headless")
	assert.False(t, has)

	_, has = l.Get("no-sandbox")
	assert.True(t, has)
}

func TestGetURLErr(t *testing.T) {
	l := New()

	go func() {
		l.output <- "Opening in existing browser session"
	}()
	_, err := l.getURL()
	assert.Error(t, err)

	l.ctxCancel()
	_, err = l.getURL()
	assert.Error(t, err)

	l = New()
	close(l.exit)
	_, err = l.getURL()
	assert.Error(t, err)
}

func TestRemoteLaunch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	u, mux, close := utils.Serve("")
	defer close()

	proxy := NewProxy()
	proxy.isWindows = true

	mux.Handle("/", proxy)

	l := NewRemote(u).KeepUserDataDir()
	client := l.Delete("keep-user-data-dir").Client()
	b := client.Context(ctx, cancel).MustConnect()
	utils.E(b.Call(ctx, "", "Browser.getVersion", nil))
	_, _ = b.Call(ctx, "", "Browser.close", nil)
	dir, _ := l.Get("user-data-dir")

	utils.Sleep(1)
	assert.NoDirExists(t, dir)
}

func TestLaunchErr(t *testing.T) {
	l := New().Bin("echo")
	go func() {
		l.exit <- struct{}{}
	}()
	_, err := l.Launch()
	assert.Error(t, err)
}
