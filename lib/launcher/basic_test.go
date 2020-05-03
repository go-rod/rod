package launcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/launcher"
)

func TestDownload(t *testing.T) {
	c := launcher.NewChrome()
	kit.E(c.Download())
	assert.FileExists(t, c.ExecPath())
}

func TestDownloadFromChina(t *testing.T) {
	c := launcher.NewChrome()
	c.Hosts = []string{"https://github.com", launcher.HostChina}
	c.Dir = filepath.Join("tmp", "chrome-from-china", kit.RandString(8))
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

	dir, _ := l.Get("user-data-dir")

	url := l.Context(context.Background()).Delete("test").Bin("").
		Log(func(s string) { kit.E(os.Stdout.WriteString(s)) }).
		Headless(false).Headless(true).RemoteDebuggingPort(0).
		Devtools(true).Devtools(false).
		UserDataDir("test").UserDataDir(dir).
		Launch()

	assert.NotEmpty(t, url)
}

func TestRemoteLaunch(t *testing.T) {
	ctx := context.Background()
	srv := kit.MustServer("127.0.0.1:0")
	defer func() { _ = srv.Listener.Close() }()
	proxy := &launcher.Proxy{Log: func(s string) {}}
	srv.Engine.NoRoute(gin.WrapH(proxy))
	go func() { _ = srv.Do() }()

	host := "ws://" + srv.Listener.Addr().String()
	header := launcher.NewRemote(host).Header()
	ws := cdp.NewDefaultWsClient(ctx, host, header)
	kit.E(cdp.New().Websocket(ws).Connect().Call(ctx, "", "Browser.close", nil))
}

func TestLaunchErr(t *testing.T) {
	assert.Panics(t, func() {
		launcher.New().Bin("not-exists").Launch()
	})
}
