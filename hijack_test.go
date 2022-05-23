package rod_test

import (
	"context"
	"errors"
	"io/ioutil"
	"mime"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func TestHijack(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	// to simulate a backend server
	s.Route("/", slash("fixtures/fetch.html"))
	s.Mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			panic("wrong http method")
		}

		g.Eq("header", r.Header.Get("Test"))

		b, err := ioutil.ReadAll(r.Body)
		g.E(err)
		g.Eq("a", string(b))

		g.HandleHTTP(".html", "test")(w, r)
	})
	s.Route("/b", "", "b")

	router := g.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		r := ctx.Request.SetContext(g.Context())
		r.Req().Header.Set("Test", "header") // override request header
		r.SetBody([]byte("test"))            // override request body
		r.SetBody(123)                       // override request body
		r.SetBody(r.Body())                  // override request body

		type MyState struct {
			Val int
		}

		ctx.CustomState = &MyState{10}

		g.Eq(http.MethodPost, r.Method())
		g.Eq(s.URL("/a"), r.URL().String())

		g.Eq(proto.NetworkResourceTypeXHR, ctx.Request.Type())
		g.Is(ctx.Request.IsNavigation(), false)
		g.Has(ctx.Request.Header("Origin"), s.URL())
		g.Len(ctx.Request.Headers(), 6)
		g.True(ctx.Request.JSONBody().Nil())

		// send request load response from real destination as the default value to hijack
		ctx.MustLoadResponse()

		g.Eq(200, ctx.Response.Payload().ResponseCode)

		// override status code
		ctx.Response.Payload().ResponseCode = http.StatusCreated

		g.Eq("4", ctx.Response.Headers().Get("Content-Length"))
		g.Has(ctx.Response.Headers().Get("Content-Type"), "text/html; charset=utf-8")

		// override response header
		ctx.Response.SetHeader("Set-Cookie", "key=val")

		// override response body
		ctx.Response.SetBody([]byte("test"))
		ctx.Response.SetBody("test")
		ctx.Response.SetBody(map[string]string{
			"text": "test",
		})

		g.Eq("{\"text\":\"test\"}", ctx.Response.Body())
	})

	router.MustAdd(s.URL("/b"), func(ctx *rod.Hijack) {
		panic("should not come to here")
	})
	router.MustRemove(s.URL("/b"))

	router.MustAdd(s.URL("/b"), func(ctx *rod.Hijack) {
		// transparent proxy
		ctx.MustLoadResponse()
	})

	go router.Run()

	g.page.MustNavigate(s.URL())

	g.Eq("201 test key=val", g.page.MustElement("#a").MustText())
	g.Eq("b", g.page.MustElement("#b").MustText())
}

func TestHijackContinue(t *testing.T) {
	g := setup(t)

	s := g.Serve().Route("/", ".html", `<body>ok</body>`)

	router := g.page.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
		wg.Done()
	})

	go router.Run()

	g.page.MustNavigate(s.URL("/a"))

	g.Eq("ok", g.page.MustElement("body").MustText())
	wg.Wait()
}

func TestHijackMockWholeResponseEmptyBody(t *testing.T) {
	g := setup(t)

	router := g.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.Response.SetBody("")
	})

	go router.Run()

	// needs to timeout or will hang when "omitempty" does not get removed from body in fulfillRequest
	timed := g.page.Timeout(time.Second)
	timed.MustNavigate(g.Serve().Route("/", ".txt", "OK").URL())

	g.Eq("", g.page.MustElement("body").MustText())
}

func TestHijackMockWholeResponseNoBody(t *testing.T) {
	// TODO: remove the skip
	t.Skip("Because of flaky test result")

	g := setup(t)

	router := g.page.HijackRequests()
	defer router.MustStop()

	// intercept and reply without setting a body
	router.MustAdd("*", func(ctx *rod.Hijack) {
		// we don't set any body here
	})

	go router.Run()

	// has to timeout as it will lock up the browser reading the reply.
	err := g.page.Timeout(time.Second).Navigate(g.Serve().Route("/", "").URL())
	g.Is(err, context.DeadlineExceeded)
}

func TestHijackMockWholeResponse(t *testing.T) {
	g := setup(t)

	router := g.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.Response.SetHeader("Content-Type", mime.TypeByExtension(".html"))
		ctx.Response.SetBody("<body>ok</body>")
	})

	go router.Run()

	g.page.MustNavigate("http://test.com")

	g.Eq("ok", g.page.MustElement("body").MustText())
}

func TestHijackSkip(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	router := g.page.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		ctx.Skip = true
		wg.Done()
	})
	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
		wg.Done()
	})

	go router.Run()

	g.page.MustNavigate(s.URL("/a"))

	wg.Wait()
}

func TestHijackOnErrorLog(t *testing.T) {
	g := setup(t)

	s := g.Serve().Route("/", ".html", `<body>ok</body>`)

	router := g.page.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	var err error

	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		ctx.OnError = func(e error) {
			err = e
			wg.Done()
		}
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()

	g.mc.stub(1, proto.FetchContinueRequest{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), errors.New("err")
	})

	go func() {
		_ = g.page.Context(g.Context()).Navigate(s.URL("/a"))
	}()
	wg.Wait()

	g.Eq(err.Error(), "err")
}

func TestHijackFailRequest(t *testing.T) {
	g := setup(t)

	s := g.Serve().Route("/page", ".html", `<html>
	<body></body>
	<script>
		fetch('/a').catch(async (err) => {
			document.title = err.message
		})
	</script></html>`)

	router := g.browser.HijackRequests()
	defer router.MustStop()

	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		ctx.Response.Fail(proto.NetworkErrorReasonAborted)
	})

	go router.Run()

	g.page.MustNavigate(s.URL("/page")).MustWaitLoad()

	g.page.MustWait(`() => document.title === 'Failed to fetch'`)

	{ // test error log
		g.mc.stub(1, proto.FetchFailRequest{}, func(send StubSend) (gson.JSON, error) {
			_, _ = send()
			return gson.JSON{}, errors.New("err")
		})
		_ = g.page.Navigate(s.URL("/a"))
	}
}

func TestHijackLoadResponseErr(t *testing.T) {
	g := setup(t)

	p := g.newPage().Context(g.Context())
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd("http://test.com/a", func(ctx *rod.Hijack) {
		g.Err(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{err: errors.New("err")},
		}, true))

		g.Err(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{res: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(&MockReader{err: errors.New("err")}),
			}},
		}, true))

		wg.Done()

		ctx.Response.Fail(proto.NetworkErrorReasonAborted)
	})

	go router.Run()

	_ = p.Navigate("http://test.com/a")

	wg.Wait()
}

func TestHijackResponseErr(t *testing.T) {
	g := setup(t)

	s := g.Serve().Route("/", ".html", `ok`)

	p := g.newPage().Context(g.Context())
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) { // to ignore favicon
		ctx.OnError = func(err error) {
			g.Err(err)
			wg.Done()
		}

		ctx.MustLoadResponse()
		g.mc.stub(1, proto.FetchFulfillRequest{}, func(send StubSend) (gson.JSON, error) {
			res, _ := send()
			return res, errors.New("err")
		})
	})

	go router.Run()

	p.MustNavigate(s.URL("/a"))

	wg.Wait()
}

func TestHandleAuth(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	// mock the server
	s.Mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="web"`)
			w.WriteHeader(401)
			return
		}

		g.Eq("a", u)
		g.Eq("b", p)
		g.HandleHTTP(".html", `<p>ok</p>`)(w, r)
	})
	s.Route("/err", ".html", "err page")

	go g.browser.MustHandleAuth("a", "b")()

	page := g.newPage(s.URL("/a"))
	page.MustElementR("p", "ok")

	wait := g.browser.HandleAuth("a", "b")
	var page2 *rod.Page
	wait2 := utils.All(func() {
		page2, _ = g.browser.Page(proto.TargetCreateTarget{URL: s.URL("/err")})
	})
	g.mc.stubErr(1, proto.FetchContinueRequest{})
	g.Err(wait())
	wait2()
	page2.MustClose()
}
