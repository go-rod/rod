package rod_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

func (c C) Incognito() {
	file := srcFile("fixtures/click.html")
	k := utils.RandString(8)

	b := c.browser.MustIncognito().Sleeper(rod.DefaultSleeper)
	page := b.MustPage(file)
	defer page.MustClose()
	page.MustEval(`k => localStorage[k] = 1`, k)

	c.Nil(c.page.MustNavigate(file).MustEval(`k => localStorage[k]`, k).Value())
	c.Eq(1, page.MustEval(`k => localStorage[k]`, k).Int())
}

func (c C) PageErr() {
	c.Panic(func() {
		c.mc.stubErr(1, proto.TargetAttachToTarget{})
		c.browser.MustPage("")
	})
}

func (c C) PageFromTarget() {
	c.Panic(func() {
		res, err := proto.TargetCreateTarget{URL: "about:blank"}.Call(c.browser)
		c.E(err)
		defer func() {
			c.browser.MustPageFromTargetID(res.TargetID).MustClose()
		}()

		c.mc.stubErr(1, proto.EmulationSetDeviceMetricsOverride{})
		c.browser.MustPageFromTargetID(res.TargetID)
	})
}

func (c C) BrowserPages() {
	page := c.browser.MustPage(srcFile("fixtures/click.html")).MustWaitLoad()
	defer page.MustClose()

	pages := c.browser.MustPages()

	c.Len(pages, 2)

	{
		c.mc.stub(1, proto.TargetGetTargets{}, func(send StubSend) (proto.JSON, error) {
			d, _ := send()
			return d.Set("targetInfos.0.type", "iframe")
		})
		pages := c.browser.MustPages()
		c.Len(pages, 1)
	}

	c.Panic(func() {
		c.mc.stubErr(1, proto.TargetCreateTarget{})
		c.browser.MustPage("")
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.TargetGetTargets{})
		c.browser.MustPages()
	})
	c.Panic(func() {
		res, err := proto.TargetCreateTarget{URL: "about:blank"}.Call(c.browser)
		c.E(err)
		defer func() {
			c.browser.MustPageFromTargetID(res.TargetID).MustClose()
		}()
		c.mc.stubErr(1, proto.TargetAttachToTarget{})
		c.browser.MustPages()
	})
}

func (c C) BrowserClearStates() {
	c.E(proto.EmulationClearGeolocationOverride{}.Call(c.page))

	defer c.browser.EnableDomain("", &proto.TargetSetDiscoverTargets{Discover: true})()
	c.browser.DisableDomain("", &proto.TargetSetDiscoverTargets{Discover: false})()
}

func (c C) BrowserWaitEvent() {
	c.NotNil(c.browser.Event())

	wait := c.page.WaitEvent(&proto.PageFrameNavigated{})
	c.page.MustNavigate(srcFile("fixtures/click.html"))
	wait()
}

func (c C) BrowserCrash() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	browser := rod.New().Context(ctx).MustConnect()
	page := browser.MustPage("")

	_ = proto.BrowserCrash{}.Call(browser)

	c.Panic(func() {
		page.MustEval(`new Promise(() => {})`)
	})
}

func (c C) BrowserCall() {
	v, err := proto.BrowserGetVersion{}.Call(c.browser)
	c.E(err)

	c.Regex("1.3", v.ProtocolVersion)
}

func (c C) Monitor() {
	b := rod.New().Timeout(1 * time.Minute).MustConnect()
	defer b.MustClose()
	p := b.MustPage(srcFile("fixtures/click.html")).MustWaitLoad()

	b, cancel := b.WithCancel()
	defer cancel()
	host := b.ServeMonitor("")

	page := c.page.MustNavigate(host)
	c.Has(page.MustElement("#targets a").MustParent().MustHTML(), string(p.TargetID))

	page.MustNavigate(host + "/page/" + string(p.TargetID))
	page.MustWait(`(id) => document.title.includes(id)`, p.TargetID)

	res, err := http.Get(host + "/screenshot")
	c.E(err)
	c.Gt(len(utils.MustReadBytes(res.Body)), 10)

	res, err = http.Get(host + "/api/page/test")
	c.E(err)
	c.Eq(400, res.StatusCode)
	c.Eq(-32602, utils.MustReadJSON(res.Body).Get("code").Int())
}

func (c C) MonitorErr() {
	defaults.Monitor = "abc"
	defer defaults.ResetWithEnv("")

	l := launcher.New()
	u := l.MustLaunch()
	defer l.Kill()

	c.Panic(func() {
		rod.New().ControlURL(u).MustConnect()
	})
}

func (c C) Trace() {
	var msg *rod.TraceMsg
	c.browser.TraceLog(func(m *rod.TraceMsg) { msg = m })
	c.browser.Trace(true).Slowmotion(time.Microsecond)
	defer func() {
		c.browser.TraceLog(nil)
		c.browser.Trace(defaults.Trace).Slowmotion(defaults.Slow)
	}()

	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	c.Eq(rod.TraceTypeInput, msg.Type)
	c.Eq("left click", msg.Details)
	c.Eq(`[input] "left click"`, msg.String())

	c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	_ = p.Mouse.Move(10, 10, 1)
}

func (c C) TraceLogs() {
	c.browser.Trace(true)
	defer func() {
		c.browser.Trace(defaults.Trace)
	}()

	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustElement("button")
	el.MustClick()

	c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	p.Overlay(0, 0, 100, 30, "")
}

func (c C) ConcurrentOperations() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	list := []int64{}
	lock := sync.Mutex{}
	add := func(item int64) {
		lock.Lock()
		defer lock.Unlock()
		list = append(list, item)
	}

	utils.All(func() {
		add(p.MustEval(`new Promise(r => setTimeout(r, 100, 2))`).Int())
	}, func() {
		add(p.MustEval(`1`).Int())
	})()

	c.Eq([]int64{1, 2}, list)
}

func (c C) PromiseLeak() {
	/*
		Perform a slow action then navigate the page to another url,
		we can see the slow operation will still be executed.

		The unexpected part is that the promise will resolve to the next page's url.
	*/

	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	var out string

	utils.All(func() {
		out = p.MustEval(`new Promise(r => setTimeout(() => r(location.href), 200))`).String()
	}, func() {
		utils.Sleep(0.1)
		p.MustNavigate(srcFile("fixtures/input.html"))
	})()

	c.Has(out, "input.html")
}

func (c C) ObjectLeak() {
	/*
		Seems like it won't leak
	*/

	p := c.page.MustNavigate(srcFile("fixtures/click.html"))

	el := p.MustElement("button")
	p.MustNavigate(srcFile("fixtures/input.html")).MustWaitLoad()
	c.Panic(func() {
		el.MustDescribe()
	})
}

func (c C) BlockingNavigation() {
	/*
		Navigate can take forever if a page doesn't response.
		If one page is blocked, other pages should still work.
	*/

	url, mux, close := utils.Serve("")
	defer close()
	pause, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		<-pause.Done()
	})
	mux.HandleFunc("/b", httpHTML(`<html>ok</html>`))

	blocked := c.browser.MustPage("")
	defer blocked.MustClose()

	go func() {
		c.Panic(func() {
			blocked.MustNavigate(url + "/a")
		})
	}()

	utils.Sleep(0.3)

	p := c.browser.MustPage(url + "/b")
	defer p.MustClose()
}

func (c C) ResolveBlocking() {
	url, mux, close := utils.Serve("")
	defer close()

	pause, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		<-pause.Done()
	})

	p := c.browser.MustPage("")
	defer p.MustClose()

	go func() {
		utils.Sleep(0.1)
		p.MustStopLoading()
	}()

	c.Panic(func() {
		p.MustNavigate(url)
	})
}

func (c C) Try() {
	c.Nil(rod.Try(func() {}))

	err := rod.Try(func() { panic(1) })
	var errVal *rod.Error
	ok := errors.As(err, &errVal)
	c.True(ok)
	c.Eq(1, errVal.Details)
}

func (c C) BrowserOthers() {
	c.browser.Timeout(time.Hour).CancelTimeout().MustPages()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c.Panic(func() {
		c.browser.Context(ctx).MustIncognito()
	})
}

func (c C) BinarySize() {
	if runtime.GOOS == "windows" {
		c.Testable.(*testing.T).SkipNow()
	}

	cmd := exec.Command("go", "build",
		"-trimpath",
		"-ldflags", "-w -s",
		"-o", "tmp/translator",
		"./lib/examples/translator")

	cmd.Env = append(os.Environ(), "GOOS=linux")

	out, err := cmd.CombinedOutput()
	if err != nil {
		c.Testable.(*testing.T).Skip(err, string(out))
	}

	stat, err := os.Stat("tmp/translator")
	c.E(err)

	c.Lt(float64(stat.Size())/1024/1024, 8.65) // mb
}

func (c C) BrowserConnectErr() {
	c.Panic(func() {
		c := newMockClient(c.Testable.(*testing.T), nil)
		c.connect = func() error { return errors.New("err") }
		rod.New().Client(c).MustConnect()
	})

	c.Panic(func() {
		ch := make(chan *cdp.Event)
		defer close(ch)

		c := newMockClient(c.Testable.(*testing.T), nil)
		c.connect = func() error { return nil }
		c.event = ch
		c.stubErr(1, proto.BrowserGetBrowserCommandLine{})
		rod.New().Client(c).MustConnect()
	})
}

func (c C) StreamReader() {
	r := rod.NewStreamReader(c.page, "")

	c.mc.stub(1, proto.IORead{}, func(send StubSend) (proto.JSON, error) {
		return proto.NewJSON(proto.IOReadResult{
			Data: "test",
		}), nil
	})
	b := make([]byte, 4)
	_, _ = r.Read(b)
	c.Eq("test", string(b))

	c.mc.stubErr(1, proto.IORead{})
	_, err := r.Read(nil)
	c.Err(err)

	c.mc.stub(1, proto.IORead{}, func(send StubSend) (proto.JSON, error) {
		return proto.NewJSON(proto.IOReadResult{
			Base64Encoded: true,
			Data:          "@",
		}), nil
	})
	_, err = r.Read(nil)
	c.Err(err)
}

// It's obvious that, the v8 will take more time to parse long function.
// For BenchmarkCache and BenchmarkNoCache, the difference is nearly 12% which is too much to ignore.
func BenchmarkCacheOff(b *testing.B) {
	p := rod.New().Timeout(1 * time.Minute).MustConnect().MustPage(srcFile("fixtures/click.html"))

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
	p := rod.New().Timeout(1 * time.Minute).MustConnect().MustPage(srcFile("fixtures/click.html"))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.MustEval(`(time) => {
			return time
		}`, time.Now().UnixNano())
	}
}
