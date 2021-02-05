package launcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/defaults"
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

type T struct {
	got.G
}

func TestPrivate(t *testing.T) {
	NewBrowser().MustGet() // preload browser to local

	got.Each(t, T{})
}

func (t T) ToHTTP() {
	u, _ := url.Parse("wss://a.com")
	t.Eq("https", toHTTP(*u).Scheme)

	u, _ = url.Parse("ws://a.com")
	t.Eq("http", toHTTP(*u).Scheme)
}

func (t T) ToWS() {
	u, _ := url.Parse("https://a.com")
	t.Eq("wss", toWS(*u).Scheme)

	u, _ = url.Parse("http://a.com")
	t.Eq("ws", toWS(*u).Scheme)
}

func (t T) Unzip() {
	t.Err(unzip(ioutil.Discard, "", ""))
}

func (t T) LaunchOptions() {
	defaults.Show = true
	defaults.Devtools = true
	inContainer = true

	// restore
	defer func() {
		defaults.ResetWithEnv("")
		inContainer = utils.InContainer
	}()

	l := New()

	_, has := l.Get("headless")
	t.False(has)

	_, has = l.Get("no-sandbox")
	t.True(has)

	_, has = l.Get("auto-open-devtools-for-tabs")
	t.True(has)
}

func (t T) GetURLErr() {
	l := New()

	l.ctxCancel()
	_, err := l.getURL()
	t.Err(err)

	l = New()
	l.parser.Lock()
	l.parser.Buffer = "err"
	l.parser.Unlock()
	close(l.exit)
	_, err = l.getURL()
	t.Eq("[launcher] Failed to get the debug url: err", err.Error())
}

func (t T) RemoteLaunch() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s := got.New(t).Serve()
	rl := NewRemoteLauncher()
	s.Mux.Handle("/", rl)

	l := MustNewRemote(s.URL()).KeepUserDataDir().Delete(flagKeepUserDataDir)
	client := l.Client()
	b := client.MustConnect(ctx)
	t.E(b.Call(ctx, "", "Browser.getVersion", nil))
	utils.Sleep(1)
	_, _ = b.Call(ctx, "", "Browser.crash", nil)
	dir, _ := l.Get("user-data-dir")

	for ctx.Err() == nil {
		utils.Sleep(0.1)
		_, err := os.Stat(dir)
		if err != nil {
			break
		}
	}
	t.Err(os.Stat(dir))
}

func (t T) LaunchErrs() {
	l := New().Bin("echo")
	_, err := l.Launch()
	t.Err(err)

	s := t.Serve()
	s.Route("/", "", nil)
	l = New().Bin("")
	l.browser.Logger = ioutil.Discard
	l.browser.Dir = filepath.Join("tmp", "browser-from-mirror", t.Srand(16))
	l.browser.Hosts = []Host{HostTest(s.URL())}
	_, err = l.Launch()
	t.Err(err)
}

func (t T) Progresser() {
	p := progresser{size: 100, logger: ioutil.Discard}

	t.E(p.Write(make([]byte, 100)))
	t.E(p.Write(make([]byte, 100)))
	t.E(p.Write(make([]byte, 100)))
}

func (t T) URLParserErr() {
	u := &URLParser{
		Buffer: "error",
	}

	t.Eq(u.Err().Error(), "[launcher] Failed to get the debug url: error")

	u.Buffer = "/tmp/rod/chromium-818858/chrome-linux/chrome: error while loading shared libraries: libgobject-2.0.so.0: cannot open shared object file: No such file or directory"
	t.Eq(u.Err().Error(), "[launcher] Failed to launch the browser, the doc might help https://go-rod.github.io/#/compatibility?id=os: /tmp/rod/chromium-818858/chrome-linux/chrome: error while loading shared libraries: libgobject-2.0.so.0: cannot open shared object file: No such file or directory")
}
