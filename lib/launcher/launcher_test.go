package launcher_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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

type T struct {
	got.G
}

func Test(t *testing.T) {
	got.Each(t, T{})
}

func (t T) Download() {
	s := t.Serve()
	s.Mux.HandleFunc("/bin/", func(rw http.ResponseWriter, r *http.Request) {
		buf := bytes.NewBuffer(nil)
		zw := zip.NewWriter(buf)
		h := &zip.FileHeader{Name: "to/"}
		h.SetMode(0755)
		_, err := zw.CreateHeader(h)
		t.E(err)
		w, err := zw.Create("to/file.txt")
		t.E(err)
		b := make([]byte, 10*1024)
		t.E(rand.Read(b))
		t.E(w.Write(b))
		t.E(zw.Close())

		rw.Header().Add("Content-Length", fmt.Sprintf("%d", buf.Len()))
		t.E(io.Copy(rw, buf))
	})

	b, cancel := newBrowser()
	defer cancel()
	b.Hosts = []string{"https://github.com", s.URL("/bin")}
	b.Dir = filepath.Join("tmp", "browser-from-mirror", t.Srand(16))
	t.E(b.Download())
	t.Nil(os.Stat(b.Dir))
}

func (t T) DownloadErr() {
	b, cancel := newBrowser()
	defer cancel()

	b.Hosts = []string{}
	t.Err(b.Download())

	b.Hosts = []string{"not-exists"}
	t.Err(b.Download())

	b.Dir = ""
	b.ExecSearchMap = map[string][]string{runtime.GOOS: {}}
	_, err := b.Get()
	t.Err(err)
}

func (t T) Launch() {
	defaults.Proxy = "test.com"
	defer func() { defaults.ResetWithEnv("") }()

	l := launcher.New()
	defer l.Kill()

	u := l.MustLaunch()
	t.Regex(`\Aws://.+\z`, u)

	parsed, _ := url.Parse(u)

	{ // test GetWebSocketDebuggerURL
		for _, prefix := range []string{"", ":", "127.0.0.1:", "ws://127.0.0.1:"} {
			u2 := launcher.MustResolveURL(prefix + parsed.Port())
			t.Regex(u, u2)
		}
	}

	{
		_, err := launcher.NewRemote("1://")
		t.Err(err)

		_, err = launcher.NewRemote("ws://not-exists")
		t.Err(err)
	}
}

func (t T) LaunchUserMode() {
	l := launcher.NewUserMode()
	defer l.Kill()

	_, has := l.Get("not-exists")
	t.False(has)

	l.Append("test-append", "a")
	f, has := l.Get("test-append")
	t.True(has)
	t.Eq("a", f)

	dir, _ := l.Get("user-data-dir")
	port := 58472

	url := l.Context(t.Context()).Delete("test").Bin("").
		Logger(ioutil.Discard).
		Leakless(false).Leakless(true).
		Headless(false).Headless(true).RemoteDebuggingPort(port).
		Devtools(true).Devtools(false).
		Proxy("test.com").
		UserDataDir("test").UserDataDir(dir).
		WorkingDir("").
		Env("TZ=Asia/Tokyo").
		MustLaunch()

	t.Eq(url, launcher.NewUserMode().RemoteDebuggingPort(port).MustLaunch())
}

func (t T) TestOpen() {
	launcher.NewBrowser().Open("about:blank")
}

func (t T) UserModeErr() {
	_, err := launcher.NewUserMode().RemoteDebuggingPort(48277).Bin("not-exists").Launch()
	t.Err(err)

	_, err = launcher.NewUserMode().RemoteDebuggingPort(58217).Bin("echo").Launch()
	t.Err(err)
}

func (t T) GetWebSocketDebuggerURLErr() {
	_, err := launcher.ResolveURL("1://")
	t.Err(err)
}

func (t T) LaunchErr() {
	t.Panic(func() {
		launcher.New().Bin("not-exists").MustLaunch()
	})
	t.Panic(func() {
		launcher.New().Headless(false).Bin("not-exists").MustLaunch()
	})
	t.Panic(func() {
		launcher.New().Client()
	})
}

func newBrowser() (*launcher.Browser, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	b := launcher.NewBrowser()
	if !testing.Verbose() {
		b.Logger = ioutil.Discard
	}
	b.Context = ctx
	return b, cancel
}

var testProfileDir = flag.Bool("test-profile-dir", false, "set it to test profile dir")

func (t T) ProfileDir() {
	if !*testProfileDir {
		t.Skip("It's not CI friendly, so we skip it!")
	}

	url := launcher.New().Headless(false).
		ProfileDir("test-profile-dir")
	url.MustLaunch()

	userDataDir, _ := url.Get("user-data-dir")
	file, err := os.Stat(filepath.Join(userDataDir, "test-profile-dir"))

	t.E(err)
	t.True(file.IsDir())
}
