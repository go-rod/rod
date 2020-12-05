package rod_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
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

func (t T) DefaultDevice() {
	ua := ""

	s := t.Serve()
	s.Mux.HandleFunc("/t", func(rw http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
	})

	t.browser.DefaultDevice(devices.IPhoneX)
	defer t.browser.DefaultDevice(devices.Test)

	t.newPage(s.URL("/t"))
	t.Eq(ua, devices.IPhoneX.UserAgentEmulation().UserAgent)

	t.browser.NoDefaultDevice()
	t.newPage(s.URL("/t"))
	t.Neq(ua, devices.IPhoneX.UserAgentEmulation().UserAgent)
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
		b, cancel := t.browser.WithCancel()
		cancel()
		b.MustIncognito()
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

func (t T) WaitDownload() {
	s := t.Serve()
	content := "test content"

	s.Route("/d", ".bin", []byte(content))
	s.Route("/page", ".html", fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, s.URL()))

	page := t.page.MustNavigate(s.URL("/page"))

	wait := t.browser.MustWaitDownload()
	page.MustElement("a").MustClick()
	data := wait()

	t.Eq(content, string(data))
}

func (t T) WaitDownloadDataURI() {
	s := t.Serve()

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

	page := t.page.MustNavigate(s.URL())

	wait1 := t.browser.MustWaitDownload()
	page.MustElement("#a").MustClick()
	data := wait1()
	t.Eq("test data", string(data))

	wait2 := t.browser.MustWaitDownload()
	page.MustElement("#b").MustClick()
	data = wait2()
	t.Eq("test blob", string(data))
}

func (t T) WaitDownloadFromNewPage() {
	s := t.Serve()
	content := "test content"

	s.Route("/d", ".bin", content)
	s.Route("/page", ".html", fmt.Sprintf(
		`<html><a href="%s/d" download target="_blank">click</a></html>`,
		s.URL()),
	)

	page := t.page.MustNavigate(s.URL("/page"))
	wait := t.browser.MustWaitDownload()
	page.MustElement("a").MustClick()
	data := wait()

	t.Eq(content, string(data))
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
			c.stubErr(1, proto.BrowserGetVersion{})
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
