package launcher_test

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/ysmood/got"
)

type T struct {
	got.G
}

func Test(t *testing.T) {
	launcher.NewBrowser().MustGet() // preload browser to local

	got.Each(t, T{})
}

func (t T) DownloadHosts() {
	t.Has(launcher.HostGoogle(launcher.DefaultRevision), "https://storage.googleapis.com/chromium-browser-snapshots")
	t.Has(launcher.HostNPM(launcher.DefaultRevision), "https://registry.npmmirror.com/-/binary/chromium-browser-snapshots")
}

func (t T) Download() {
	s := t.Serve()
	s.Mux.HandleFunc("/fast/", func(rw http.ResponseWriter, r *http.Request) {
		buf := bytes.NewBuffer(nil)
		zw := zip.NewWriter(buf)

		// folder "to"
		h := &zip.FileHeader{Name: "to/"}
		h.SetMode(0755)
		_, err := zw.CreateHeader(h)
		t.E(err)

		// file "file.txt"
		w, err := zw.CreateHeader(&zip.FileHeader{Name: "to/file.txt"})
		t.E(err)
		b := []byte(t.Srand(2 * 1024 * 1024))
		t.E(w.Write(b))

		t.E(zw.Close())

		rw.Header().Add("Content-Length", fmt.Sprintf("%d", buf.Len()))
		_, _ = io.Copy(rw, buf)
	})
	s.Mux.HandleFunc("/slow/", func(rw http.ResponseWriter, r *http.Request) {
		t := time.NewTimer(3 * time.Second)
		select {
		case <-t.C:
		case <-r.Context().Done():
			t.Stop()
		}
	})

	b, cancel := newBrowser()
	b.Logger = ioutil.Discard
	defer cancel()
	b.Hosts = []launcher.Host{launcher.HostTest(s.URL("/slow")), launcher.HostTest(s.URL("/fast"))}
	b.Dir = filepath.Join("tmp", "browser-from-mirror", t.Srand(16))
	t.E(b.Download())
	t.Nil(os.Stat(b.Dir))
}

func (t T) BrowserGet() {
	t.Nil(os.Stat(launcher.NewBrowser().MustGet()))
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

		_, err := launcher.ResolveURL("")
		t.Err(err)
	}

	{
		_, err := launcher.NewManaged("")
		t.Err(err)

		_, err = launcher.NewManaged("1://")
		t.Err(err)

		_, err = launcher.NewManaged("ws://not-exists")
		t.Err(err)
	}
}

func (t T) LaunchUserMode() {
	l := launcher.NewUserMode()
	defer l.Kill()

	l.Kill() // empty kill should do nothing

	has := l.Has("not-exists")
	t.False(has)

	l.Append("test-append", "a")
	f := l.Get("test-append")
	t.Eq("a", f)

	dir := l.Get(flags.UserDataDir)
	port := 58472

	url := l.Context(t.Context()).Delete("test").Bin("").
		Logger(ioutil.Discard).
		Leakless(false).Leakless(true).
		Headless(false).Headless(true).RemoteDebuggingPort(port).
		NoSandbox(true).NoSandbox(false).
		Devtools(true).Devtools(false).
		StartURL("about:blank").
		Proxy("test.com").
		UserDataDir("test").UserDataDir(dir).
		WorkingDir("").
		Env("TZ=Asia/Tokyo").
		MustLaunch()

	t.Eq(url, launcher.NewUserMode().RemoteDebuggingPort(port).MustLaunch())
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
	{
		l := launcher.New().XVFB()
		_, _ = l.Launch()
		l.Kill()
	}
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
	url := launcher.New().Headless(false).
		ProfileDir("").ProfileDir("test-profile-dir")

	if !*testProfileDir {
		t.Skip("It's not CI friendly, so we skip it!")
	}

	url.MustLaunch()

	userDataDir := url.Get(flags.UserDataDir)
	file, err := os.Stat(filepath.Join(userDataDir, "test-profile-dir"))

	t.E(err)
	t.True(file.IsDir())
}
