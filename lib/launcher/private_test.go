package launcher

import (
	"context"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
	"github.com/ysmood/kit/pkg/utils"
)

func TestToHTTP(t *testing.T) {
	u, _ := url.Parse("wss://a.com")
	toHTTP(u)
	assert.Equal(t, "https", u.Scheme)

	u, _ = url.Parse("ws://a.com")
	toHTTP(u)
	assert.Equal(t, "http", u.Scheme)
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
	srv := kit.MustServer("127.0.0.1:0")
	defer func() { _ = srv.Listener.Close() }()
	proxy := NewProxy()
	proxy.isWindows = true
	srv.Engine.NoRoute(gin.WrapH(proxy))
	go func() { _ = srv.Do() }()

	u := "ws://" + srv.Listener.Addr().String()
	client := NewRemote(u).KeepUserDataDir().Delete("keep-user-data-dir").Client()
	b := client.Context(ctx, cancel).Connect()
	kit.E(b.Call(ctx, "", "Browser.getVersion", nil))
	_, _ = b.Call(ctx, "", "Browser.close", nil)
}
func TestLaunchErr(t *testing.T) {
	l := New().Bin("echo")
	go func() {
		l.exit <- utils.Nil{}
	}()
	_, err := l.LaunchE()
	assert.Error(t, err)
}
