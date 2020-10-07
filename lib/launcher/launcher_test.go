package launcher_test

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ysmood/got"
)

func TestDownload(t *testing.T) {
	as := got.New(t)

	b, cancel := newBrowser()
	defer cancel()
	as.E(b.Download())
	as.Nil(os.Stat(b.ExecPath()))
}

func TestDownloadWithMirror(t *testing.T) {
	as := got.New(t)

	b, cancel := newBrowser()
	defer cancel()
	b.Hosts = []string{"https://github.com", launcher.HostTaobao}
	b.Dir = filepath.Join("tmp", "browser-from-mirror", as.Srand(16))
	as.E(b.Download())
	as.Nil(os.Stat(b.ExecPath()))

	b.Hosts = []string{}
	as.Err(b.Download())

	b.Hosts = []string{"not-exists"}
	as.Err(b.Download())

	b.Dir = ""
	b.ExecSearchMap = map[string][]string{runtime.GOOS: {}}
	_, err := b.Get()
	as.Err(err)
}

func TestLaunch(t *testing.T) {
	as := got.New(t)

	defaults.Proxy = "test.com"
	defer func() { defaults.ResetWithEnv("") }()

	l := launcher.New()
	defer l.Kill()

	u := l.MustLaunch()
	as.Regex(`\Aws://.+\z`, u)

	parsed, _ := url.Parse(u)

	{ // test GetWebSocketDebuggerURL
		for _, prefix := range []string{"", ":", "127.0.0.1:", "ws://127.0.0.1:"} {
			u2 := launcher.MustResolveURL(prefix + parsed.Port())
			as.Regex(u, u2)
		}
	}

	{
		_, err := launcher.NewRemote("1://")
		as.Err(err)

		_, err = launcher.NewRemote("ws://not-exists")
		as.Err(err)
	}
}

func TestLaunchUserMode(t *testing.T) {
	as := got.New(t)

	l := launcher.NewUserMode()
	defer l.Kill()

	_, has := l.Get("not-exists")
	as.False(has)

	l.Append("test-append", "a")
	f, has := l.Get("test-append")
	as.True(has)
	as.Eq("a", f)

	dir, _ := l.Get("user-data-dir")
	port := 58472

	url := l.Context(context.Background()).Delete("test").Bin("").
		Logger(ioutil.Discard).
		Leakless(false).Leakless(true).
		Headless(false).Headless(true).RemoteDebuggingPort(port).
		Devtools(true).Devtools(false).
		Proxy("test.com").
		UserDataDir("test").UserDataDir(dir).
		WorkingDir("").
		Env("TZ=Asia/Tokyo").
		MustLaunch()

	as.Eq(url, launcher.NewUserMode().RemoteDebuggingPort(port).MustLaunch())
}

func TestOpen(t *testing.T) {
	launcher.NewBrowser().Open("about:blank")
}

func TestUserModeErr(t *testing.T) {
	as := got.New(t)

	_, err := launcher.NewUserMode().RemoteDebuggingPort(48277).Bin("not-exists").Launch()
	as.Err(err)

	_, err = launcher.NewUserMode().RemoteDebuggingPort(58217).Bin("echo").Launch()
	as.Err(err)
}

func TestGetWebSocketDebuggerURLErr(t *testing.T) {
	as := got.New(t)

	_, err := launcher.ResolveURL("1://")
	as.Err(err)
}

func TestLaunchErr(t *testing.T) {
	as := got.New(t)

	as.Panic(func() {
		launcher.New().Bin("not-exists").MustLaunch()
	})
	as.Panic(func() {
		launcher.New().Headless(false).Bin("not-exists").MustLaunch()
	})
	as.Panic(func() {
		launcher.New().Client()
	})
}

func newBrowser() (*launcher.Browser, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	b := launcher.NewBrowser()
	b.Context = ctx
	return b, cancel
}
