package launcher

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

func HostTest(host string) Host {
	return func(revision int) string {
		return fmt.Sprintf(
			"%s/chromium-browser-snapshots/%s/%d/%s",
			host,
			hostConf.urlPrefix,
			revision,
			hostConf.zipName,
		)
	}
}

var setup = got.Setup(nil)

func TestMain(m *testing.M) {
	NewBrowser().MustGet() // preload browser to local

	os.Exit(m.Run())
}

func TestToHTTP(t *testing.T) {
	g := setup(t)

	u, _ := url.Parse("wss://a.com")
	g.Eq("https", toHTTP(*u).Scheme)

	u, _ = url.Parse("ws://a.com")
	g.Eq("http", toHTTP(*u).Scheme)
}

func TestToWS(t *testing.T) {
	g := setup(t)

	u, _ := url.Parse("https://a.com")
	g.Eq("wss", toWS(*u).Scheme)

	u, _ = url.Parse("http://a.com")
	g.Eq("ws", toWS(*u).Scheme)
}

func TestUnzip(t *testing.T) {
	g := setup(t)

	g.Err(unzip(utils.LoggerQuiet, "", ""))
}

func TestLaunchOptions(t *testing.T) {
	g := setup(t)

	defaults.Show = true
	defaults.Devtools = true
	inContainer = true

	// restore
	defer func() {
		defaults.ResetWith("")
		inContainer = utils.InContainer
	}()

	l := New()

	g.False(l.Has(flags.Headless))

	g.True(l.Has(flags.NoSandbox))

	g.True(l.Has("auto-open-devtools-for-tabs"))
}

func TestGetURLErr(t *testing.T) {
	g := setup(t)

	l := New()

	l.ctxCancel()
	_, err := l.getURL()
	g.Err(err)

	l = New()
	l.parser.lock.Lock()
	l.parser.Buffer = "err"
	l.parser.lock.Unlock()
	close(l.exit)
	_, err = l.getURL()
	g.Eq("[launcher] Failed to get the debug url: err", err.Error())
}

func TestManaged(t *testing.T) {
	g := setup(t)

	ctx := g.Timeout(5 * time.Second)

	s := got.New(g).Serve()
	rl := NewManager()
	s.Mux.Handle("/", rl)

	l := MustNewManaged(s.URL()).KeepUserDataDir().Delete(flags.KeepUserDataDir)
	c := l.MustClient()
	g.E(c.Call(ctx, "", "Browser.getVersion", nil))
	utils.Sleep(1)
	_, _ = c.Call(ctx, "", "Browser.crash", nil)
	dir := l.Get(flags.UserDataDir)

	for ctx.Err() == nil {
		utils.Sleep(0.1)
		_, err := os.Stat(dir)
		if err != nil {
			break
		}
	}
	g.Err(os.Stat(dir))

	u, h := MustNewManaged(s.URL()).Bin("go").ClientHeader()
	_, err := cdp.StartWithURL(ctx, u, h)
	g.Eq(err.(*cdp.ErrBadHandshake).Body, "not allowed rod-bin path: go")
}

func TestLaunchErrs(t *testing.T) {
	g := setup(t)

	l := New().Bin("echo")
	_, err := l.Launch()
	g.Err(err)

	s := g.Serve()
	s.Route("/", "", nil)
	l = New().Bin("")
	l.browser.Logger = utils.LoggerQuiet
	l.browser.Dir = filepath.Join("tmp", "browser-from-mirror", g.RandStr(16))
	l.browser.Hosts = []Host{HostTest(s.URL())}
	_, err = l.Launch()
	g.Err(err)
}

func TestProgresser(t *testing.T) {
	g := setup(t)

	p := progresser{size: 100, logger: utils.LoggerQuiet}

	g.E(p.Write(make([]byte, 100)))
	g.E(p.Write(make([]byte, 100)))
	g.E(p.Write(make([]byte, 100)))
}

func TestURLParserErr(t *testing.T) {
	g := setup(t)

	u := &URLParser{
		Buffer: "error",
		lock:   &sync.Mutex{},
	}

	g.Eq(u.Err().Error(), "[launcher] Failed to get the debug url: error")

	u.Buffer = "/tmp/rod/chromium-818858/chrome-linux/chrome: error while loading shared libraries: libgobject-2.0.so.0: cannot open shared object file: No such file or directory"
	g.Eq(u.Err().Error(), "[launcher] Failed to launch the browser, the doc might help https://go-rod.github.io/#/compatibility?id=os: /tmp/rod/chromium-818858/chrome-linux/chrome: error while loading shared libraries: libgobject-2.0.so.0: cannot open shared object file: No such file or directory")
}

func TestBrowserDownloadErr(t *testing.T) {
	g := setup(t)

	b := NewBrowser()
	b.Logger = utils.LoggerQuiet
	malURL := "https://npm.taobao.org/mirrors/chromium-browser-snapshots//869685/"
	g.Has(b.download(g.Context(), malURL).Error(), "failed to download the browser: 200")
}

func TestTestOpen(t *testing.T) {
	openExec = func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command("not-exists")
		cmd.Process = &os.Process{}
		return cmd
	}
	defer func() { openExec = exec.Command }()

	Open("about:blank")
}
