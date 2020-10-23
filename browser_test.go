package rod_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
	"github.com/ysmood/gson"
)

func (t T) Incognito() {
	k := t.Srand(16)

	b := t.browser.MustIncognito().Sleeper(rod.DefaultSleeper)
	defer b.MustClose()

	page := b.MustPage(t.blank())
	defer page.MustClose()
	page.MustEval(`k => localStorage[k] = 1`, k)

	t.True(t.page.MustNavigate(t.blank()).MustEval(`k => localStorage[k]`, k).Nil())
	t.Eq(page.MustEval(`k => localStorage[k]`, k).Str(), "1") // localStorage can only store string
}

func (t T) PageErr() {
	t.Panic(func() {
		t.mc.stubErr(1, proto.TargetAttachToTarget{})
		t.browser.MustPage("")
	})
}

func (t T) PageFromTarget() {
	t.Panic(func() {
		res, err := proto.TargetCreateTarget{URL: "about:blank"}.Call(t.browser)
		t.E(err)
		defer func() {
			t.browser.MustPageFromTargetID(res.TargetID).MustClose()
		}()

		t.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		t.browser.MustPageFromTargetID(res.TargetID)
	})
}

func (t T) BrowserPages() {
	t.newPage(t.blank()).MustWaitLoad()

	pages := t.browser.MustPages()

	t.Len(pages, 2)

	{
		t.mc.stub(1, proto.TargetGetTargets{}, func(send StubSend) (gson.JSON, error) {
			d, _ := send()
			return *d.Set("targetInfos.0.type", "iframe"), nil
		})
		pages := t.browser.MustPages()
		t.Len(pages, 1)
	}

	t.Panic(func() {
		t.mc.stubErr(1, proto.TargetCreateTarget{})
		t.browser.MustPage("")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.TargetGetTargets{})
		t.browser.MustPages()
	})
	t.Panic(func() {
		res, err := proto.TargetCreateTarget{URL: "about:blank"}.Call(t.browser)
		t.E(err)
		defer func() {
			t.browser.MustPageFromTargetID(res.TargetID).MustClose()
		}()
		t.mc.stubErr(1, proto.TargetAttachToTarget{})
		t.browser.MustPages()
	})
}

func (t T) BrowserClearStates() {
	t.E(proto.EmulationClearGeolocationOverride{}.Call(t.page))
}

func (t T) BrowserEvent() {
	messages := t.browser.Context(t.Context()).Event()
	p := t.newPage("")
	wait := make(chan struct{})
	for msg := range messages {
		e := proto.TargetAttachedToTarget{}
		if msg.Load(&e) {
			t.Eq(e.TargetInfo.TargetID, p.TargetID)
			close(wait)
			break
		}
	}
	<-wait
}

func (t T) BrowserEventClose() {
	event := make(chan *cdp.Event)
	c := &MockClient{
		connect: func() error { return nil },
		call: func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
			return nil, errors.New("err")
		},
		event: event,
	}
	b := rod.New().Client(c)
	_ = b.Connect()
	b.Event()
	close(event)
}

func (t T) BrowserWaitEvent() {
	t.NotNil(t.browser.Context(t.Context()).Event())

	wait := t.page.WaitEvent(proto.PageFrameNavigated{})
	t.page.MustNavigate(t.blank())
	wait()

	wait = t.browser.EachEvent(func(e *proto.PageFrameNavigated, id proto.TargetSessionID) bool {
		return true
	})
	t.page.MustNavigate(t.blank())
	wait()
}

func (t T) BrowserCrash() {
	browser := rod.New().Context(t.Context()).MustConnect()
	page := browser.MustPage("")
	js := `new Promise(r => setTimeout(r, 10000))`

	go t.Panic(func() {
		page.MustEval(js)
	})

	utils.Sleep(0.2)

	_ = proto.BrowserCrash{}.Call(browser)

	utils.Sleep(0.3)

	t.Panic(func() {
		page.MustEval(js)
	})
}

func (t T) BrowserCall() {
	v, err := proto.BrowserGetVersion{}.Call(t.browser)
	t.E(err)

	t.Regex("1.3", v.ProtocolVersion)
}

func (t T) Monitor() {
	b := rod.New().MustConnect()
	defer b.MustClose()
	p := b.MustPage(t.blank()).MustWaitLoad()

	b, cancel := b.WithCancel()
	defer cancel()
	host := b.Context(t.Context()).ServeMonitor("")

	page := t.page.MustNavigate(host)
	t.Has(page.MustElement("#targets a").MustParent().MustHTML(), string(p.TargetID))

	page.MustNavigate(host + "/page/" + string(p.TargetID))
	page.MustWait(`(id) => document.title.includes(id)`, p.TargetID)

	img := t.Req("", host+"/screenshot").Bytes()
	t.Gt(len(img), 10)

	res := t.Req("", host+"/api/page/test")
	t.Eq(400, res.StatusCode)
	t.Eq(-32602, gson.New(res.Body).Get("code").Int())
}

func (t T) MonitorErr() {
	l := launcher.New()
	u := l.MustLaunch()
	defer l.Kill()

	t.Panic(func() {
		rod.New().Monitor("abc").ControlURL(u).MustConnect()
	})
}

func (t T) Trace() {
	var msg *rod.TraceMsg
	t.browser.Logger(utils.Log(func(list ...interface{}) { msg = list[0].(*rod.TraceMsg) }))
	t.browser.Trace(true).Slowmotion(time.Microsecond)
	defer func() {
		t.browser.Logger(rod.DefaultLogger)
		t.browser.Trace(defaults.Trace).Slowmotion(defaults.Slow)
	}()

	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	t.Eq(rod.TraceTypeInput, msg.Type)
	t.Eq("left click", msg.Details)
	t.Eq(`[input] "left click"`, msg.String())

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	_ = p.Mouse.Move(10, 10, 1)
}

func (t T) TraceLogs() {
	t.browser.Logger(utils.LoggerQuiet)
	t.browser.Trace(true)
	defer func() {
		t.browser.Logger(rod.DefaultLogger)
		t.browser.Trace(defaults.Trace)
	}()

	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	p.Overlay(0, 0, 100, 30, "")
}

func (t T) BlockingNavigation() {
	/*
		Navigate can take forever if a page doesn't response.
		If one page is blocked, other pages should still work.
	*/

	s := t.Serve()
	pause := t.Context()

	s.Mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		<-pause.Done()
	})
	s.Route("/b", ".html", `<html>ok</html>`)

	blocked := t.newPage("")

	go func() {
		t.Panic(func() {
			blocked.MustNavigate(s.URL("/a"))
		})
	}()

	utils.Sleep(0.3)

	t.newPage(s.URL("/b"))
}

func (t T) ResolveBlocking() {
	s := t.Serve()

	pause := t.Context()

	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		<-pause.Done()
	})

	p := t.newPage("")

	go func() {
		utils.Sleep(0.1)
		p.MustStopLoading()
	}()

	t.Panic(func() {
		p.MustNavigate(s.URL())
	})
}

func (t T) TestTry() {
	t.Nil(rod.Try(func() {}))

	err := rod.Try(func() { panic(1) })
	var errVal *rod.ErrTry
	t.True(errors.As(err, &errVal))
	t.Is(err, &rod.ErrTry{})
	t.Eq(1, errVal.Value)
	t.Eq(errVal.Error(), "error value: 1")
}

func (t T) BrowserOthers() {
	t.browser.Timeout(time.Hour).CancelTimeout().MustPages()

	t.Panic(func() {
		t.browser.Context(t.Timeout(0)).MustIncognito()
	})
}

func (t T) BinarySize() {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	cmd := exec.Command("go", "build",
		"-trimpath",
		"-ldflags", "-w -s",
		"-o", "tmp/translator",
		"./lib/examples/translator")

	cmd.Env = append(os.Environ(), "GOOS=linux")

	t.Nil(cmd.Run())

	stat, err := os.Stat("tmp/translator")
	t.E(err)

	t.Lte(float64(stat.Size())/1024/1024, 8.3) // mb
}

func (t T) BrowserCookies() {
	b := t.browser.MustIncognito()
	defer b.MustClose()

	b.MustSetCookies([]*proto.NetworkCookie{{
		Name:   "a",
		Value:  "val",
		Domain: "test.com",
	}})

	cookies := b.MustGetCookies()

	t.Len(cookies, 1)
	t.Eq(cookies[0].Name, "a")
	t.Eq(cookies[0].Value, "val")

	t.mc.stubErr(1, proto.StorageGetCookies{})
	t.Err(b.GetCookies())
}

func (t T) BrowserConnectErr() {
	t.Panic(func() {
		c := &MockClient{connect: func() error { return errors.New("err") }}
		rod.New().Client(c).MustConnect()
	})
	t.Panic(func() {
		ch := make(chan *cdp.Event)
		defer close(ch)

		c := &MockClient{connect: func() error { return nil }, event: ch}
		c.stubErr(1, proto.TargetSetDiscoverTargets{})
		rod.New().Client(c).MustConnect()
	})
	t.Panic(func() {
		ch := make(chan *cdp.Event)
		defer close(ch)

		c := &MockClient{connect: func() error { return nil }, event: ch}
		c.stub(1, proto.TargetSetDiscoverTargets{}, func(send StubSend) (gson.JSON, error) {
			c.stubErr(1, proto.BrowserGetBrowserCommandLine{})
			return gson.JSON{}, nil
		})
		rod.New().Client(c).MustConnect()
	})
}

func (t T) StreamReader() {
	r := rod.NewStreamReader(t.page, "")

	t.mc.stub(1, proto.IORead{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(proto.IOReadResult{
			Data: "test",
		}), nil
	})
	b := make([]byte, 4)
	_, _ = r.Read(b)
	t.Eq("test", string(b))

	t.mc.stubErr(1, proto.IORead{})
	_, err := r.Read(nil)
	t.Err(err)

	t.mc.stub(1, proto.IORead{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(proto.IOReadResult{
			Base64Encoded: true,
			Data:          "@",
		}), nil
	})
	_, err = r.Read(nil)
	t.Err(err)
}

// It's obvious that, the v8 will take more time to parse long function.
// For BenchmarkCache and BenchmarkNoCache, the difference is nearly 12% which is too much to ignore.
func BenchmarkCacheOff(b *testing.B) {
	c := T{G: got.New(b)}

	p := rod.New().Timeout(1 * time.Minute).MustConnect().MustPage(c.srcFile("fixtures/click.html"))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.MustEval(`(time) => {
			// won't call this function, it's used to make the declaration longer
			function foo (id, left, top, width, height, msg) {
				var div = document.createElement('div')
				var msgDiv = document.createElement('div')
				div.id = id
				div.style = 'position: fixed; z-index:2147483647; border: 2px dashed red;'
					+ 'border-radius: 3px; box-shadow: #5f3232 0 0 3px; pointer-events: none;'
					+ 'box-sizing: border-box;'
					+ 'left:' + left + 'px;'
					+ 'top:' + top + 'px;'
					+ 'height:' + height + 'px;'
					+ 'width:' + width + 'px;'
		
				if (height === 0) {
					div.style.border = 'none'
				}
			
				msgDiv.style = 'position: absolute; color: #cc26d6; font-size: 12px; background: #ffffffeb;'
					+ 'box-shadow: #333 0 0 3px; padding: 2px 5px; border-radius: 3px; white-space: nowrap;'
					+ 'top:' + height + 'px; '
			
				msgDiv.innerHTML = msg
			
				div.appendChild(msgDiv)
				document.body.appendChild(div)
			}
			return time
		}`, time.Now().UnixNano())
	}
}

func BenchmarkCache(b *testing.B) {
	c := T{G: got.New(b)}

	p := rod.New().Timeout(1 * time.Minute).MustConnect().MustPage(c.srcFile("fixtures/click.html"))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.MustEval(`(time) => {
			return time
		}`, time.Now().UnixNano())
	}
}

func TestLab(t *testing.T) {
	b := &rod.Browser{}
	_ = b.PageFromSession("")

	t.SkipNow()

	b = rod.New().MustConnect()

	target, _ := proto.TargetCreateTarget{URL: "http://www.example.com"}.Call(b)

	session, _ := proto.TargetAttachToTarget{TargetID: target.TargetID, Flatten: true}.Call(b)

	p := b.PageFromSession(session.SessionID)

	_, _ = proto.RuntimeEvaluate{Expression: `window`}.Call(p)

	utils.Pause()
}
