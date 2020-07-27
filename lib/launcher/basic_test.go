package launcher_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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

	c.Hosts = []string{}
	assert.Error(t, c.Download())

	c.Hosts = []string{"not-exists"}
	assert.Error(t, c.Download())

	c.Dir = ""
	c.ExecSearchMap = map[string][]string{runtime.GOOS: {}}
	_, err := c.Get()
	assert.Error(t, err)
}

func TestLaunch(t *testing.T) {
	l := launcher.New()
	defer func() {
		_ = kit.KillTree(l.PID())
	}()

	url := l.Launch()

	assert.NotEmpty(t, url)
}

func TestLaunchUserMode(t *testing.T) {
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
	port := 58472

	url := l.Context(context.Background()).Delete("test").Bin("").
		Log(func(s string) { kit.E(os.Stdout.WriteString(s)) }).
		Headless(false).Headless(true).RemoteDebuggingPort(port).
		Devtools(true).Devtools(false).Reap(true).
		UserDataDir("test").UserDataDir(dir).
		Launch()

	assert.Equal(t,
		url,
		launcher.NewUserMode().RemoteDebuggingPort(port).Launch(),
	)
}

func TestOpen(t *testing.T) {
	launcher.NewBrowser().Open("about:blank")
}

func TestUserModeErr(t *testing.T) {
	_, err := launcher.NewUserMode().RemoteDebuggingPort(48277).Bin("not-exists").LaunchE()
	assert.Error(t, err)

	_, err = launcher.NewUserMode().RemoteDebuggingPort(58217).Bin("echo").LaunchE()
	assert.Error(t, err)
}

func TestGetWebSocketDebuggerURLErr(t *testing.T) {
	_, err := launcher.GetWebSocketDebuggerURL(context.Background(), "1://")
	assert.Error(t, err)
}

func TestLaunchErr(t *testing.T) {
	assert.Panics(t, func() {
		launcher.New().Bin("not-exists").Launch()
	})
	assert.Panics(t, func() {
		launcher.New().Headless(false).Bin("not-exists").Launch()
	})
}
