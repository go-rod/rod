package rod_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func TestIncognito(t *testing.T) {
	g := setup(t)

	k := g.RandStr(16)

	b := g.browser.MustIncognito().Sleeper(rod.DefaultSleeper)
	defer b.MustClose()

	page := b.MustPage(g.blank())
	defer page.MustClose()
	page.MustEval(`k => localStorage[k] = 1`, k)

	g.True(g.page.MustNavigate(g.blank()).MustEval(`k => localStorage[k]`, k).Nil())
	g.Eq(page.MustEval(`k => localStorage[k]`, k).Str(), "1") // localStorage can only store string

	g.Panic(func() {
		g.mc.stubErr(1, proto.TargetCreateBrowserContext{})
		g.browser.MustIncognito()
	})
}

func TestBrowserResetControlURL(_ *testing.T) {
	rod.New().ControlURL("test").ControlURL("")
}

func TestDefaultDevice(t *testing.T) {
	g := setup(t)

	ua := ""

	s := g.Serve()
	s.Mux.HandleFunc("/t", func(rw http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
	})

	// TODO: https://github.com/golang/go/issues/51459
	b := *g.browser
	b.DefaultDevice(devices.IPhoneX)

	b.MustPage(s.URL("/t")).MustClose()
	g.Eq(ua, devices.IPhoneX.UserAgentEmulation().UserAgent)

	b.NoDefaultDevice()
	b.MustPage(s.URL("/t")).MustClose()
	g.Neq(ua, devices.IPhoneX.UserAgentEmulation().UserAgent)
}

func TestPageErr(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		g.mc.stubErr(1, proto.TargetAttachToTarget{})
		g.browser.MustPage()
	})
}

func TestPageFromTarget(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		res, err := proto.TargetCreateTarget{URL: "about:blank"}.Call(g.browser)
		g.E(err)
		defer func() {
			g.browser.MustPageFromTargetID(res.TargetID).MustClose()
		}()

		g.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		g.browser.MustPageFromTargetID(res.TargetID)
	})
}

func TestBrowserPages(t *testing.T) {
	g := setup(t)

	b := g.browser
	pages := b.MustPages()
	g.Gte(len(pages), 1)

	{
		g.mc.stub(1, proto.TargetGetTargets{}, func(send StubSend) (gson.JSON, error) {
			d, _ := send()
			return *d.Set("targetInfos.0.type", "iframe"), nil
		})
		b.MustPages()
	}

	g.Panic(func() {
		g.mc.stubErr(1, proto.TargetCreateTarget{})
		b.MustPage()
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.TargetGetTargets{})
		b.MustPages()
	})
	g.Panic(func() {
		_, err := proto.TargetCreateTarget{URL: "about:blank"}.Call(b)
		g.E(err)
		g.mc.stubErr(1, proto.TargetAttachToTarget{})
		b.MustPages()
	})
}

func TestBrowserClearStates(t *testing.T) {
	g := setup(t)

	g.E(proto.EmulationClearGeolocationOverride{}.Call(g.page))
}

func TestBrowserEvent(t *testing.T) {
	g := setup(t)

	messages := g.browser.Context(g.Context()).Event()
	p := g.newPage()
	wait := make(chan struct{})
	for msg := range messages {
		e := proto.TargetAttachedToTarget{}
		if msg.Load(&e) {
			g.Eq(e.TargetInfo.TargetID, p.TargetID)
			close(wait)
			break
		}
	}
	<-wait
}

func TestBrowserWaitEvent(t *testing.T) {
	g := setup(t)

	g.NotNil(g.browser.Context(g.Context()).Event())

	wait := g.page.WaitEvent(proto.PageFrameNavigated{})
	g.page.MustNavigate(g.blank())
	wait()

	wait = g.browser.EachEvent(func(e *proto.PageFrameNavigated, id proto.TargetSessionID) bool {
		return true
	})
	g.page.MustNavigate(g.blank())
	wait()
}

func TestBrowserCrash(t *testing.T) {
	g := setup(t)

	browser := rod.New().Context(g.Context()).MustConnect()
	page := browser.MustPage()
	js := `() => new Promise(r => setTimeout(r, 10000))`

	go g.Panic(func() {
		page.MustEval(js)
	})

	utils.Sleep(0.2)

	_ = proto.BrowserCrash{}.Call(browser)

	utils.Sleep(0.3)

	_, err := page.Eval(js)
	g.Has(err.Error(), "use of closed network connection")
}

func TestBrowserCall(t *testing.T) {
	g := setup(t)

	v, err := proto.BrowserGetVersion{}.Call(g.browser)
	g.E(err)

	g.Regex("1.3", v.ProtocolVersion)
}

func TestBlockingNavigation(t *testing.T) {
	g := setup(t)

	/*
		Navigate can take forever if a page doesn't response.
		If one page is blocked, other pages should still work.
	*/

	s := g.Serve()
	pause := g.Context()

	s.Mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		<-pause.Done()
	})
	s.Route("/b", ".html", `<html>ok</html>`)

	blocked := g.newPage()

	go func() {
		g.Panic(func() {
			blocked.MustNavigate(s.URL("/a"))
		})
	}()

	utils.Sleep(0.3)

	g.newPage(s.URL("/b"))
}

func TestResolveBlocking(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	pause := g.Context()

	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		<-pause.Done()
	})

	p := g.newPage()

	go func() {
		utils.Sleep(0.1)
		p.MustStopLoading()
	}()

	g.Panic(func() {
		p.MustNavigate(s.URL())
	})
}

func TestTestTry(t *testing.T) {
	g := setup(t)

	g.Nil(rod.Try(func() {}))

	err := rod.Try(func() { panic(1) })
	var errVal *rod.ErrTry
	g.True(errors.As(err, &errVal))
	g.Is(err, &rod.ErrTry{})
	g.Eq(errVal.Unwrap().Error(), "1")
	g.Eq(1, errVal.Value)
	g.Has(errVal.Error(), "error value: 1\ngoroutine")

	errVal = rod.Try(func() { panic(errors.New("t")) }).(*rod.ErrTry)
	g.Eq(errVal.Unwrap().Error(), "t")
}

func TestBrowserOthers(t *testing.T) {
	g := setup(t)

	g.browser.Timeout(time.Second).CancelTimeout().MustGetCookies()
}

func TestBinarySize(t *testing.T) {
	g := setup(t)

	if runtime.GOOS == "windows" || utils.InContainer {
		g.SkipNow()
	}

	cmd := exec.Command("go", "build",
		"-trimpath",
		"-ldflags", "-w -s",
		"-o", "tmp/translator",
		"./lib/examples/translator")

	cmd.Env = append(os.Environ(), "GOOS=linux")

	g.Nil(cmd.Run())

	stat, err := os.Stat("tmp/translator")
	g.E(err)

	g.Lte(float64(stat.Size())/1024/1024, 11) // mb
}

func TestBrowserCookies(t *testing.T) {
	g := setup(t)

	b := g.browser.MustIncognito()
	defer b.MustClose()

	b.MustSetCookies(&proto.NetworkCookie{
		Name:   "a",
		Value:  "val",
		Domain: "test.com",
	})

	cookies := b.MustGetCookies()

	g.Len(cookies, 1)
	g.Eq(cookies[0].Name, "a")
	g.Eq(cookies[0].Value, "val")

	{
		b.MustSetCookies()
		cookies := b.MustGetCookies()
		g.Len(cookies, 0)
	}

	g.mc.stubErr(1, proto.StorageGetCookies{})
	g.Err(b.GetCookies())
}

func TestWaitDownload(t *testing.T) {
	g := setup(t)

	s := g.Serve()
	content := "test content"

	s.Route("/d", ".bin", []byte(content))
	s.Route("/page", ".html", fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, s.URL()))

	page := g.page.MustNavigate(s.URL("/page"))

	wait := g.browser.MustWaitDownload()
	page.MustElement("a").MustClick()
	data := wait()

	g.Eq(content, string(data))
}

func TestWaitDownloadDataURI(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	s.Route("/", ".html",
		`<html>
			<a id="a" href="data:text/plain;,test%20data" download>click</a>
			<a id="b" download>click</a>
			<script>
				const b = document.getElementById('b')
				b.href = URL.createObjectURL(new Blob(['test blob'], {
					type: "text/plain; charset=utf-8"
				}))
			</script>
		</html>`,
	)

	page := g.page.MustNavigate(s.URL())

	wait1 := g.browser.MustWaitDownload()
	page.MustElement("#a").MustClick()
	data := wait1()
	g.Eq("test data", string(data))

	wait2 := g.browser.MustWaitDownload()
	page.MustElement("#b").MustClick()
	data = wait2()
	g.Eq("test blob", string(data))
}

func TestWaitDownloadCancel(t *testing.T) {
	g := setup(t)

	wait := g.browser.Context(g.Timeout(0)).WaitDownload(os.TempDir())
	g.Eq(wait(), (*proto.PageDownloadWillBegin)(nil))
}

func TestWaitDownloadFromNewPage(t *testing.T) {
	g := setup(t)

	s := g.Serve()
	content := "test content"

	s.Route("/d", ".bin", content)
	s.Route("/page", ".html", fmt.Sprintf(
		`<html><a href="%s/d" download target="_blank">click</a></html>`,
		s.URL()),
	)

	page := g.page.MustNavigate(s.URL("/page"))
	wait := g.browser.MustWaitDownload()
	page.MustElement("a").MustClick()
	data := wait()

	g.Eq(content, string(data))
}

func TestBrowserConnectErr(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		rod.New().ControlURL(g.RandStr(16)).MustConnect()
	})
}

func TestStreamReader(t *testing.T) {
	g := setup(t)

	r := rod.NewStreamReader(g.page, "")

	g.mc.stub(1, proto.IORead{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(proto.IOReadResult{
			Data: "test",
		}), nil
	})
	b := make([]byte, 4)
	_, _ = r.Read(b)
	g.Eq("test", string(b))

	g.mc.stubErr(1, proto.IORead{})
	_, err := r.Read(nil)
	g.Err(err)

	g.mc.stub(1, proto.IORead{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(proto.IOReadResult{
			Base64Encoded: true,
			Data:          "@",
		}), nil
	})
	_, err = r.Read(nil)
	g.Err(err)
}

func TestBrowserConnectFailure(t *testing.T) {
	g := setup(t)

	c := g.Context()
	c.Cancel()
	err := rod.New().Context(c).Connect()
	if err == nil {
		g.Fatal("expected an error on connect failure")
	}
}

func TestBrowserPool(_ *testing.T) {
	pool := rod.NewBrowserPool(3)
	create := func() *rod.Browser { return rod.New().MustConnect() }
	b := pool.Get(create)
	pool.Put(b)
	pool.Cleanup(func(p *rod.Browser) {
		p.MustClose()
	})
}

func TestOldBrowser(t *testing.T) {
	t.Skip()

	g := setup(t)
	u := launcher.New().Revision(686378).MustLaunch()
	b := rod.New().ControlURL(u).MustConnect()
	g.Cleanup(b.MustClose)
	res, err := proto.BrowserGetVersion{}.Call(b)
	g.E(err)
	g.Eq(res.Revision, "@19d4547535ab5aba70b4730443f84e8153052174")
}

func TestBrowserLostConnection(t *testing.T) {
	g := setup(t)

	l := launcher.New()
	p := rod.New().ControlURL(l.MustLaunch()).MustConnect().MustPage(g.blank())

	go func() {
		utils.Sleep(1)
		l.Kill()
	}()

	_, err := p.Eval(`() => new Promise(r => {})`)
	g.Err(err)
}

func TestBrowserConnectConflict(t *testing.T) {
	g := setup(t)
	g.Panic(func() {
		rod.New().Client(&cdp.Client{}).ControlURL("test").MustConnect()
	})
}
