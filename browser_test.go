package rod_test

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/tidwall/sjson"
	"github.com/ysmood/kit"
)

func (s *S) TestIncognito() {
	file := srcFile("fixtures/click.html")
	k := kit.RandString(8)

	b := s.browser.Incognito()
	page := b.Page(file)
	page.Eval(`k => localStorage[k] = 1`, k)

	s.Nil(s.page.Navigate(file).Eval(`k => localStorage[k]`, k).Value())
	s.EqualValues(1, page.Eval(`k => localStorage[k]`, k).Int())
}

func (s *S) TestBrowserPages() {
	page := s.browser.Page(srcFile("fixtures/click.html")).WaitLoad()
	defer page.Close()

	pages := s.browser.Pages()

	// TODO: I don't know why sometimes windows can miss one
	if runtime.GOOS == "windows" {
		s.GreaterOrEqual(len(pages), 2)
	} else {
		s.Len(pages, 3)

		func() {
			defer s.at(1, func(d []byte, err error) ([]byte, error) {
				return sjson.SetBytes(d, "targetInfos.0.type", "iframe")
			})()
			pages := s.browser.Pages()
			s.Len(pages, 2)
		}()
	}
	s.Panics(func() {
		defer s.errorAt(1, nil)()
		s.browser.Page("")
	})
	s.Panics(func() {
		defer s.errorAt(1, nil)()
		s.browser.Pages()
	})
	s.Panics(func() {
		res, err := proto.TargetCreateTarget{URL: "about:blank"}.Call(s.browser)
		kit.E(err)
		defer func() {
			s.browser.PageFromTargetID(res.TargetID).Close()
		}()
		defer s.errorAt(2, nil)()
		s.browser.Pages()
	})
}

func (s *S) TestBrowserClearStates() {
	kit.E(proto.EmulationClearGeolocationOverride{}.Call(s.page))

	defer s.browser.EnableDomain(s.browser.GetContext(), "", &proto.TargetSetDiscoverTargets{Discover: true})()
	s.browser.DisableDomain(s.browser.GetContext(), "", &proto.TargetSetDiscoverTargets{Discover: false})()
}

func (s *S) TestBrowserWaitEvent() {
	s.NotNil(s.browser.Event())

	wait := s.page.WaitEvent(&proto.PageFrameNavigated{})
	s.page.Navigate(srcFile("fixtures/click.html"))
	wait()
}

func (s *S) TestBrowserCrash() {
	browser := rod.New().Timeout(1 * time.Minute).Connect()
	page := browser.Page("")

	wait := browser.WaitEvent(&proto.PageFrameNavigated{})
	go func() {
		kit.Sleep(0.3)
		_ = proto.BrowserCrash{}.Call(browser)
	}()

	s.Panics(func() {
		page.Eval(`new Promise(() => {})`)
	})

	wait()
}

func (s *S) TestBrowserCall() {
	v, err := proto.BrowserGetVersion{}.Call(s.browser)
	kit.E(err)

	s.Regexp("1.3", v.ProtocolVersion)
}

func (s *S) TestMonitor() {
	b := rod.New().Timeout(1 * time.Minute).Connect()
	defer b.Close()
	p := b.Page(srcFile("fixtures/click.html")).WaitLoad()
	host := b.ServeMonitor("127.0.0.1:0", true).Listener.Addr().String()

	page := s.page.Navigate("http://" + host)
	s.Contains(page.Element("#targets a").Parent().HTML(), string(p.TargetID))

	page.Navigate("http://" + host + "/page/" + string(p.TargetID))
	s.Contains(page.Eval(`document.title`).Str, p.TargetID)

	s.Equal(400, kit.Req("http://"+host+"/api/page/test").MustResponse().StatusCode)
}

func (s *S) TestRemoteLaunch() {
	url, engine, close := serve()
	defer close()

	proxy := launcher.NewProxy()
	engine.NoRoute(gin.WrapH(proxy))

	ctx, cancel := context.WithCancel(context.Background())
	l := launcher.NewRemote(strings.ReplaceAll(url, "http", "ws"))
	c := l.Client().Context(ctx, cancel)
	b := rod.New().Context(ctx, cancel).Client(c).Connect()
	defer b.Close()

	p := b.Page(srcFile("fixtures/click.html"))
	p.Element("button").Click()
	s.True(p.Has("[a=ok]"))
}

func (s *S) TestTrace() {
	msg := ""
	var errs []error
	s.browser.TraceLog(
		func(s string) {
			msg = s
		},
		func(string, rod.Array) {},
		func(e error) {
			errs = append(errs, e)
		},
	)
	s.browser.Trace(true).Slowmotion(time.Microsecond)
	defer func() {
		s.browser.TraceLog(nil, nil, nil)
		s.browser.Trace(false).Slowmotion(0)
	}()

	p := s.page.Navigate(srcFile("fixtures/click.html"))
	el := p.Element("button")
	el.Click()
	s.Equal("left click", msg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p.Context(ctx, cancel).Overlay(0, 0, 100, 100, "msg")
	s.Error(errs[0])

	el.Context(ctx, cancel).Trace("ok")
	s.Error(errs[1])
}

func (s *S) TestConcurrentOperations() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	list := []int64{}
	lock := sync.Mutex{}
	add := func(item int64) {
		lock.Lock()
		defer lock.Unlock()
		list = append(list, item)
	}

	kit.All(func() {
		add(p.Eval(`new Promise(r => setTimeout(r, 100, 2))`).Int())
	}, func() {
		add(p.Eval(`1`).Int())
	})()

	s.Equal([]int64{1, 2}, list)
}

func (s *S) TestPromiseLeak() {
	/*
		Perform a slow action then navigate the page to another url,
		we can see the slow operation will still be executed.

		The unexpected part is that the promise will resolve to the next page's url.
	*/

	p := s.page.Navigate(srcFile("fixtures/click.html"))
	var out string

	kit.All(func() {
		out = p.Eval(`new Promise(r => setTimeout(() => r(location.href), 200))`).String()
	}, func() {
		kit.Sleep(0.1)
		p.Navigate(srcFile("fixtures/input.html"))
	})()

	s.Contains(out, "input.html")
}

func (s *S) TestObjectLeak() {
	/*
		Seems like it won't leak
	*/

	p := s.page.Navigate(srcFile("fixtures/click.html"))

	el := p.Element("button")
	p.Navigate(srcFile("fixtures/input.html")).WaitLoad()
	s.Panics(func() {
		el.Describe()
	})
}

func (s *S) TestBlockingNavigation() {
	/*
		Navigate can take forever if a page doesn't response.
		If one page is blocked, other pages should still work.
	*/

	url, engine, close := serve()
	defer close()
	pause, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.GET("/a", func(ctx kit.GinContext) {
		<-pause.Done()
	})
	engine.GET("/b", ginHTML(`<html>ok</html>`))

	blocked := s.browser.Page("")
	defer blocked.Close()

	go func() {
		s.Panics(func() {
			blocked.Navigate(url + "/a")
		})
	}()

	kit.Sleep(0.3)

	p := s.browser.Page(url + "/b")
	defer p.Close()
}

func (s *S) TestResolveBlocking() {
	url, engine, close := serve()
	defer close()

	pause, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.NoRoute(func(ctx kit.GinContext) {
		<-pause.Done()
	})

	p := s.browser.Page("")
	defer p.Close()

	go func() {
		kit.Sleep(0.1)
		p.StopLoading()
	}()

	s.Panics(func() {
		p.Navigate(url)
	})
}

func (s *S) TestTry() {
	s.Nil(rod.Try(func() {}))

	err := rod.Try(func() { panic(1) })
	var errVal *rod.Error
	ok := errors.As(err, &errVal)
	s.True(ok)
	s.Equal(1, errVal.Details)
}

func (s *S) TestBrowserOthers() {
	s.browser.Timeout(time.Minute).CancelTimeout()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s.Panics(func() {
		s.browser.Context(ctx, cancel).Incognito()
	})
}

// It's obvious that, the v8 will take more time to parse long function.
// For BenchmarkCache and BenchmarkNoCache, the difference is nearly 12% which is too much to ignore.
func BenchmarkCacheOff(b *testing.B) {
	p := rod.New().Timeout(1 * time.Minute).Connect().Page(srcFile("fixtures/click.html"))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Eval(`(time) => {
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
	p := rod.New().Timeout(1 * time.Minute).Connect().Page(srcFile("fixtures/click.html"))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Eval(`(time) => {
			return time
		}`, time.Now().UnixNano())
	}
}
