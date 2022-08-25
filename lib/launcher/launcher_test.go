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
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

var setup = got.Setup(nil)

func TestDownloadHosts(t *testing.T) {
	g := setup(t)

	g.Has(launcher.HostGoogle(launcher.RevisionDefault), "https://storage.googleapis.com/chromium-browser-snapshots")
	g.Has(launcher.HostNPM(launcher.RevisionDefault), "https://registry.npmmirror.com/-/binary/chromium-browser-snapshots")
	g.Has(launcher.HostPlaywright(launcher.RevisionDefault), "https://playwright.azureedge.net/")
}

func TestDownload(t *testing.T) {
	g := setup(t)

	s := g.Serve()
	s.Mux.HandleFunc("/fast/", func(rw http.ResponseWriter, r *http.Request) {
		buf := bytes.NewBuffer(nil)
		zw := zip.NewWriter(buf)

		// folder "to"
		h := &zip.FileHeader{Name: "to/"}
		h.SetMode(0755)
		_, err := zw.CreateHeader(h)
		g.E(err)

		// file "file.txt"
		w, err := zw.CreateHeader(&zip.FileHeader{Name: "to/file.txt"})
		g.E(err)
		b := []byte(g.RandStr(2 * 1024 * 1024))
		g.E(w.Write(b))

		g.E(zw.Close())

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
	b.Logger = utils.LoggerQuiet
	defer cancel()
	b.Hosts = []launcher.Host{launcher.HostTest(s.URL("/slow")), launcher.HostTest(s.URL("/fast"))}
	b.Dir = filepath.Join("tmp", "browser-from-mirror", g.RandStr(16))
	g.E(b.Download())
	g.Nil(os.Stat(b.Dir))
}

func TestBrowserGet(t *testing.T) {
	g := setup(t)

	g.Nil(os.Stat(launcher.NewBrowser().MustGet()))

	b := launcher.NewBrowser()
	b.Revision = 0
	b.Logger = utils.LoggerQuiet
	_, err := b.Get()
	g.Eq(err.Error(), "Can't find a browser binary for your OS, the doc might help https://go-rod.github.io/#/compatibility?id=os")
}

func TestLaunch(t *testing.T) {
	g := setup(t)

	defaults.Proxy = "test.com"
	defer func() { defaults.ResetWith("") }()

	l := launcher.New()
	defer l.Kill()

	u := l.MustLaunch()
	g.Regex(`\Aws://.+\z`, u)

	parsed, _ := url.Parse(u)

	{ // test GetWebSocketDebuggerURL
		for _, prefix := range []string{"", ":", "127.0.0.1:", "ws://127.0.0.1:"} {
			u2 := launcher.MustResolveURL(prefix + parsed.Port())
			g.Regex(u, u2)
		}

		_, err := launcher.ResolveURL("")
		g.Err(err)
	}

	{
		_, err := launcher.NewManaged("")
		g.Err(err)

		_, err = launcher.NewManaged("1://")
		g.Err(err)

		_, err = launcher.NewManaged("ws://not-exists")
		g.Err(err)
	}

	{
		g.Panic(func() { launcher.New().Set("a=b") })
	}
}

func TestLaunchUserMode(t *testing.T) {
	g := setup(t)

	l := launcher.NewUserMode()
	defer l.Kill()

	l.Kill() // empty kill should do nothing

	has := l.Has("not-exists")
	g.False(has)

	l.Append("test-append", "a")
	f := l.Get("test-append")
	g.Eq("a", f)

	dir := l.Get(flags.UserDataDir)
	port := 58472

	l = l.Context(g.Context()).Delete("test").Bin("").
		Revision(launcher.RevisionDefault).
		Logger(ioutil.Discard).
		Leakless(false).Leakless(true).
		Headless(false).Headless(true).RemoteDebuggingPort(port).
		NoSandbox(true).NoSandbox(false).
		Devtools(true).Devtools(false).
		StartURL("about:blank").
		Proxy("test.com").
		UserDataDir("test").UserDataDir(dir).
		WorkingDir("").
		Env(append(os.Environ(), "TZ=Asia/Tokyo")...)

	g.Eq(l.FormatArgs(), []string /* len=6 cap=8 */ {
		"--headless",
		`--no-startup-window`,           /* len=19 */
		`--proxy-server=test.com`,       /* len=23 */
		`--remote-debugging-port=58472`, /* len=29 */
		"--test-append=a",
		"about:blank",
	})

	url := l.MustLaunch()

	g.Eq(url, launcher.NewUserMode().RemoteDebuggingPort(port).MustLaunch())
}

func TestUserModeErr(t *testing.T) {
	g := setup(t)

	_, err := launcher.NewUserMode().RemoteDebuggingPort(48277).Bin("not-exists").Launch()
	g.Err(err)

	_, err = launcher.NewUserMode().RemoteDebuggingPort(58217).Bin("echo").Launch()
	g.Err(err)
}

func TestAppMode(t *testing.T) {
	g := setup(t)

	l := launcher.NewAppMode("http://example.com")

	g.Eq(l.Get(flags.App), "http://example.com")
}

func TestGetWebSocketDebuggerURLErr(t *testing.T) {
	g := setup(t)

	_, err := launcher.ResolveURL("1://")
	g.Err(err)
}

func TestLaunchErr(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		launcher.New().Bin("not-exists").MustLaunch()
	})
	g.Panic(func() {
		launcher.New().Headless(false).Bin("not-exists").MustLaunch()
	})
	g.Panic(func() {
		launcher.New().ClientHeader()
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
		b.Logger = utils.LoggerQuiet
	}
	b.Context = ctx
	return b, cancel
}

var testProfileDir = flag.Bool("test-profile-dir", false, "set it to test profile dir")

func TestProfileDir(t *testing.T) {
	g := setup(t)

	url := launcher.New().Headless(false).
		ProfileDir("").ProfileDir("test-profile-dir")

	if !*testProfileDir {
		g.Skip("It's not CI friendly, so we skip it!")
	}

	url.MustLaunch()

	userDataDir := url.Get(flags.UserDataDir)
	file, err := os.Stat(filepath.Join(userDataDir, "test-profile-dir"))

	g.E(err)
	g.True(file.IsDir())
}

func TestBrowserExists(t *testing.T) {
	g := setup(t)

	b := launcher.NewBrowser()
	b.Revision = 0
	g.False(b.Exists())

	// fake a broken executable
	g.E(utils.Mkdir(b.Destination()))
	g.Cleanup(func() { _ = os.RemoveAll(b.Destination()) })
	g.False(b.Exists())
}
