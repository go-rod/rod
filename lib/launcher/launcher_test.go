package launcher_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	skipDownload(t)

	c := launcher.NewBrowser()
	utils.E(c.Download())
	assert.FileExists(t, c.ExecPath())
}

func TestDownloadWithMirror(t *testing.T) {
	skipDownload(t)

	c := launcher.NewBrowser()
	c.Hosts = []string{"https://github.com", launcher.HostTaobao}
	c.Dir = filepath.Join("tmp", "browser-from-mirror", utils.RandString(8))
	utils.E(c.Download())
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
	defer func() { kill(l.PID()) }()

	url := l.MustLaunch()

	assert.Regexp(t, `\Aws://.+\z`, url)
}

func TestLaunchUserMode(t *testing.T) {
	l := launcher.NewUserMode()
	defer func() { kill(l.PID()) }()

	_, has := l.Get("not-exists")
	assert.False(t, has)

	l.Append("test-append", "a")
	f, has := l.Get("test-append")
	assert.True(t, has)
	assert.Equal(t, "a", f)

	dir, _ := l.Get("user-data-dir")
	port := 58472

	url := l.Context(context.Background()).Delete("test").Bin("").
		Logger(ioutil.Discard).
		Leakless(false).Leakless(true).
		Headless(false).Headless(true).RemoteDebuggingPort(port).
		Devtools(true).Devtools(false).Reap(true).
		Proxy("test.com").
		UserDataDir("test").UserDataDir(dir).
		WorkingDir("").
		Env("TZ=Asia/Tokyo").
		MustLaunch()

	assert.Equal(t,
		url,
		launcher.NewUserMode().RemoteDebuggingPort(port).MustLaunch(),
	)
}

func TestOpen(t *testing.T) {
	launcher.NewBrowser().Open("about:blank")
}

func TestUserModeErr(t *testing.T) {
	_, err := launcher.NewUserMode().RemoteDebuggingPort(48277).Bin("not-exists").Launch()
	assert.Error(t, err)

	_, err = launcher.NewUserMode().RemoteDebuggingPort(58217).Bin("echo").Launch()
	assert.Error(t, err)
}

func TestGetWebSocketDebuggerURLErr(t *testing.T) {
	_, err := launcher.GetWebSocketDebuggerURL("1://")
	assert.Error(t, err)
}

func TestLaunchErr(t *testing.T) {
	assert.Panics(t, func() {
		launcher.New().Bin("not-exists").MustLaunch()
	})
	assert.Panics(t, func() {
		launcher.New().Headless(false).Bin("not-exists").MustLaunch()
	})
	assert.Panics(t, func() {
		launcher.New().Client()
	})
}

func skipDownload(t *testing.T) {
	_, skipDownload := os.LookupEnv("skip_download")

	if skipDownload {
		t.SkipNow()
	}
}

func kill(pid int) {
	ps, err := os.FindProcess(pid)
	utils.E(err)
	utils.E(ps.Kill())
}
