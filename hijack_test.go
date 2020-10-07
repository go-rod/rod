package rod_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func (c C) Hijack() {
	s := c.Serve()

	// to simulate a backend server
	s.Route("/", slash("fixtures/fetch.html"))
	s.Mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			panic("wrong http method")
		}

		c.Eq("header", r.Header.Get("Test"))

		b, err := ioutil.ReadAll(r.Body)
		c.E(err)
		c.Eq("a", string(b))

		c.HandleHTTP(".html", "test")(w, r)
	})
	s.Route("/b", "", "b")

	router := c.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		r := ctx.Request.SetContext(context.Background())
		r.Req().URL = r.Req().URL            // override request url
		r.Req().Header.Set("Test", "header") // override request header
		r.SetBody([]byte("test"))            // override request body
		r.SetBody(123)                       // override request body
		r.SetBody(r.Body())                  // override request body

		c.Eq(http.MethodPost, r.Method())
		c.Eq(s.URL("/a"), r.URL().String())

		c.Eq(proto.NetworkResourceTypeXHR, ctx.Request.Type())
		c.Has(ctx.Request.Header("Origin"), s.URL())
		c.Len(ctx.Request.Headers(), 5).Msg("%s", utils.Dump(ctx.Request.Headers()))
		c.True(ctx.Request.JSONBody().Nil())

		// send request load response from real destination as the default value to hijack
		ctx.MustLoadResponse()

		c.Eq(200, ctx.Response.Payload().ResponseCode)

		// override status code
		ctx.Response.Payload().ResponseCode = http.StatusCreated

		c.Eq("4", ctx.Response.Headers().Get("Content-Length"))
		c.Has(ctx.Response.Headers().Get("Content-Type"), "text/html; charset=utf-8")

		// override response header
		ctx.Response.SetHeader("Set-Cookie", "key=val")

		// override response body
		ctx.Response.SetBody([]byte("test"))
		ctx.Response.SetBody("test")
		ctx.Response.SetBody(map[string]string{
			"text": "test",
		})

		c.Eq("{\"text\":\"test\"}", ctx.Response.Body())
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

	c.page.MustNavigate(s.URL())

	c.Eq("201 test key=val", c.page.MustElement("#a").MustText())
	c.Eq("b", c.page.MustElement("#b").MustText())
}

func (c C) HijackContinue() {
	s := c.Serve().Route("/", ".html", `<body>ok</body>`)

	router := c.page.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
		wg.Done()
	})

	go router.Run()

	c.page.MustNavigate(s.URL())

	c.Eq("ok", c.page.MustElement("body").MustText())
	wg.Wait()
}

func (c C) HijackOnErrorLog() {
	s := c.Serve().Route("/", ".html", `<body>ok</body>`)

	router := c.page.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	var err error

	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.OnError = func(e error) {
			err = e
			wg.Done()
		}
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()

	c.mc.stub(1, proto.FetchContinueRequest{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), errors.New("err")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = c.page.Context(ctx).Navigate(s.URL())
	}()
	wg.Wait()

	c.Eq(err.Error(), "err")
}

func (c C) HijackFailRequest() {
	s := c.Serve().Route("/", ".html", `<html>
	<body></body>
	<script>
		fetch('/a').catch(async (err) => {
			document.title = err.message
		})
	</script></html>`)

	router := c.browser.HijackRequests()
	defer router.MustStop()

	err := make(chan error)
	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		ctx.OnError = func(e error) { err <- e }
		ctx.Response.Fail(proto.NetworkErrorReasonAborted)
	})

	go router.Run()

	c.page.MustNavigate(s.URL()).MustWaitLoad()

	c.page.MustWait(`document.title == 'Failed to fetch'`)

	{ // test error log
		c.mc.stubErr(1, proto.FetchFailRequest{})
		c.page.MustNavigate(s.URL())
		c.Err(<-err)
	}
}

func (c C) HijackLoadResponseErr() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := c.page.Context(ctx)
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd("*", func(ctx *rod.Hijack) {
		c.Err(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{err: errors.New("err")},
		}, true))

		c.Err(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{res: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(&MockReader{err: errors.New("err")}),
			}},
		}, true))

		wg.Done()
	})

	go router.Run()

	go func() { _ = p.Navigate(c.srcFile("./fixtures/click.html")) }()

	wg.Wait()
}

func (c C) HijackResponseErr() {
	s := c.Serve().Route("/", ".html", `ok`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := c.page.Context(ctx)
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.OnError = func(err error) {
			c.Err(err)
			wg.Done()
		}

		ctx.MustLoadResponse()
		c.mc.stubErr(1, proto.FetchFulfillRequest{})
	})

	go router.Run()

	go func() { _ = p.Navigate(s.URL()) }()

	wg.Wait()
}

func (c C) HandleAuth() {
	s := c.Serve()

	// mock the server
	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="web"`)
			w.WriteHeader(401)
			return
		}

		c.Eq("a", u)
		c.Eq("b", p)
		c.HandleHTTP(".html", `<p>ok</p>`)(w, r)
	})
	s.Route("/err", ".html", "err page")

	c.browser.MustHandleAuth("a", "b")

	page := c.browser.MustPage(s.URL())
	defer page.MustClose()
	page.MustElementR("p", "ok")

	wait := c.browser.HandleAuth("a", "b")
	var page2 *rod.Page
	wait2 := utils.All(func() {
		page2, _ = c.browser.Page(s.URL("/err"))
	})
	c.mc.stubErr(1, proto.FetchContinueRequest{})
	c.Err(wait())
	wait2()
	page2.MustClose()
}

func (c C) GetDownloadFile() {
	s := c.Serve()
	content := "test content"

	s.Route("/d", ".bin", []byte(content))
	s.Route("/", ".html", fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, s.URL()))

	page := c.page.MustNavigate(s.URL())

	wait := page.MustGetDownloadFile(s.URL("/d")) // the pattern is used to prevent favicon request
	page.MustElement("a").MustClick()
	data := wait()

	c.Eq(content, string(data))

	waitErr := page.GetDownloadFile(s.URL("/d"), "", &http.Client{
		Transport: &MockRoundTripper{err: errors.New("err")},
	})
	page.MustElement("a").MustClick()
	{
		c.mc.stubErr(1, proto.FetchEnable{})
		_, _, err := waitErr()
		c.Err(err)
	}
	_, _, err := waitErr()
	c.Err(err)
}

func (c C) GetDownloadFileFromDataURI() {
	s := c.Serve()

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

	page := c.page.MustNavigate(s.URL())

	wait := page.MustGetDownloadFile("data:*")
	page.MustElement("#a").MustClick()
	data := wait()
	c.Eq("test data", string(data))

	wait = page.MustGetDownloadFile("data:*")
	page.MustElement("#b").MustClick()
	data = wait()
	c.Eq("test blob", string(data))

	c.Panic(func() {
		wait = page.MustGetDownloadFile("data:*")
		page.MustElement("#b").MustClick()
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		data = wait()
	})
}

func (c C) GetDownloadFileWithHijack() {
	s := c.Serve()
	content := "test content"

	s.Route("/d", ".bin", []byte(content))
	s.Route("/", ".html", fmt.Sprintf(`<html><a href="%s" download>click</a></html>`, s.URL("/d")))

	page := c.page.MustNavigate(s.URL())

	r := page.HijackRequests()
	r.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.OnError = func(error) {}
		ctx.MustLoadResponse()
	})
	go r.Run()
	defer r.MustStop()

	wait := page.MustGetDownloadFile(s.URL("/d")) // the pattern is used to prevent favicon request
	page.MustElement("a").MustClick()
	data := wait()

	c.Eq(content, string(data))
}
