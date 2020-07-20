package launcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
)

func TestDownload(t *testing.T) {
	c := launcher.NewBrowser()
	kit.E(c.Download())
	assert.FileExists(t, c.ExecPath())
}

func TestDownloadWithMirror(t *testing.T) {
	c := launcher.NewBrowser()
	c.Hosts = []string{"https://github.com", launcher.HostTaobao}
	c.Dir = filepath.Join("tmp", "browser-from-mirror", kit.RandString(8))
	kit.E(c.Download())
	assert.FileExists(t, c.ExecPath())
}

func TestLaunch(t *testing.T) {
	l := launcher.New()
	defer func() {
		_ = kit.KillTree(l.PID())
	}()

	url := l.Launch()

	assert.NotEmpty(t, url)
}

func TestLaunchOptions(t *testing.T) {
	l := launcher.NewUserMode()
	defer func() {
		_ = kit.KillTree(l.PID())
	}()

	_, has := l.Get("not-exists")
	assert.False(t, has)

	l.Append("test-append", "a")
	f, has := l.Get("test-append")
	assert.True(t, has)
	assert.Equal(t, "a", f)

	dir, _ := l.Get("user-data-dir")

	url := l.Context(context.Background()).Delete("test").Bin("").
		Log(func(s string) { kit.E(os.Stdout.WriteString(s)) }).
		Headless(false).Headless(true).RemoteDebuggingPort(0).
		Devtools(true).Devtools(false).Reap(true).
		UserDataDir("test").UserDataDir(dir).
		Launch()

	assert.NotEmpty(t, url)
}

func TestRemoteLaunch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := kit.MustServer("127.0.0.1:0")
	defer func() { _ = srv.Listener.Close() }()
	proxy := &launcher.Proxy{Log: func(s string) {}}
	srv.Engine.NoRoute(gin.WrapH(proxy))
	go func() { _ = srv.Do() }()

	u := "ws://" + srv.Listener.Addr().String()
	client := launcher.NewRemote(u).Client()
	b := client.Context(ctx, cancel).Connect()
	kit.E(b.Call(ctx, "", "Browser.getVersion", nil))
	_, _ = b.Call(ctx, "", "Browser.close", nil)
}

func TestLaunchErr(t *testing.T) {
	assert.Panics(t, func() {
		launcher.New().Bin("not-exists").Launch()
	})
}
