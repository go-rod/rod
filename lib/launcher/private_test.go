package launcher

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

func TestToHTTP(t *testing.T) {
	as := got.New(t)
	u, _ := url.Parse("wss://a.com")
	as.Eq("https", toHTTP(*u).Scheme)

	u, _ = url.Parse("ws://a.com")
	as.Eq("http", toHTTP(*u).Scheme)
}

func TestToWS(t *testing.T) {
	as := got.New(t)
	u, _ := url.Parse("https://a.com")
	as.Eq("wss", toWS(*u).Scheme)

	u, _ = url.Parse("http://a.com")
	as.Eq("ws", toWS(*u).Scheme)
}

func TestUnzip(t *testing.T) {
	as := got.New(t)
	as.Err(unzip(ioutil.Discard, "", ""))
}

func TestLaunchOptions(t *testing.T) {
	as := got.New(t)
	defaults.Show = true
	defaults.Devtools = true
	isInDocker = true

	// recover
	defer func() {
		defaults.ResetWithEnv("")
		isInDocker = utils.FileExists("/.dockerenv")
	}()

	l := New()

	_, has := l.Get("headless")
	as.False(has)

	_, has = l.Get("no-sandbox")
	as.True(has)

	_, has = l.Get("auto-open-devtools-for-tabs")
	as.True(has)
}

func TestGetURLErr(t *testing.T) {
	as := got.New(t)
	l := New()

	l.ctxCancel()
	_, err := l.getURL()
	as.Err(err)

	l = New()
	l.parser.Buffer = "err"
	close(l.exit)
	_, err = l.getURL()
	as.Eq("[launcher] Failed to get the debug url err", err.Error())
}

func TestRemoteLaunch(t *testing.T) {
	as := got.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s := got.New(t).Serve()
	s.Mux.Handle("/", NewRemoteLauncher())

	l := MustNewRemote(s.URL()).KeepUserDataDir().Delete(flagKeepUserDataDir)
	client := l.Client()
	b := client.MustConnect(ctx)
	as.E(b.Call(ctx, "", "Browser.getVersion", nil))
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
	as.Err(os.Stat(dir))
}

func TestLaunchErrs(t *testing.T) {
	as := got.New(t)
	l := New().Bin("echo")
	go func() {
		l.exit <- struct{}{}
	}()
	_, err := l.Launch()
	as.Err(err)

	l = New()
	l.browser.Dir = as.Srand(16)
	l.browser.ExecSearchMap = nil
	l.browser.Hosts = []string{}
	_, err = l.Launch()
	as.Err(err)
}
