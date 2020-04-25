package launcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
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

func TestLaunchErr(t *testing.T) {
	assert.Panics(t, func() {
		launcher.New().Bin("not-exists").Launch()
	})
}
