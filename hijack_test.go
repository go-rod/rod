package rod_test

import (
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

func (t T) Hijack() {
	s := t.Serve()

	// to simulate a backend server
	s.Route("/", slash("fixtures/fetch.html"))
	s.Mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			panic("wrong http method")
		}

		t.Eq("header", r.Header.Get("Test"))

		b, err := ioutil.ReadAll(r.Body)
		t.E(err)
		t.Eq("a", string(b))

		t.HandleHTTP(".html", "test")(w, r)
	})
	s.Route("/b", "", "b")

	router := t.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		r := ctx.Request.SetContext(t.Context())
		r.Req().URL = r.Req().URL            // override request url
		r.Req().Header.Set("Test", "header") // override request header
		r.SetBody([]byte("test"))            // override request body
		r.SetBody(123)                       // override request body
		r.SetBody(r.Body())                  // override request body

		t.Eq(http.MethodPost, r.Method())
		t.Eq(s.URL("/a"), r.URL().String())

		t.Eq(proto.NetworkResourceTypeXHR, ctx.Request.Type())
		t.Has(ctx.Request.Header("Origin"), s.URL())
		t.Len(ctx.Request.Headers(), 5).Msg("%s", utils.Dump(ctx.Request.Headers()))
		t.True(ctx.Request.JSONBody().Nil())

		// send request load response from real destination as the default value to hijack
		ctx.MustLoadResponse()

		t.Eq(200, ctx.Response.Payload().ResponseCode)

		// override status code
		ctx.Response.Payload().ResponseCode = http.StatusCreated

		t.Eq("4", ctx.Response.Headers().Get("Content-Length"))
		t.Has(ctx.Response.Headers().Get("Content-Type"), "text/html; charset=utf-8")

		// override response header
		ctx.Response.SetHeader("Set-Cookie", "key=val")

		// override response body
		ctx.Response.SetBody([]byte("test"))
		ctx.Response.SetBody("test")
		ctx.Response.SetBody(map[string]string{
			"text": "test",
		})

		t.Eq("{\"text\":\"test\"}", ctx.Response.Body())
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

	t.page.MustNavigate(s.URL())

	t.Eq("201 test key=val", t.page.MustElement("#a").MustText())
	t.Eq("b", t.page.MustElement("#b").MustText())
}

func (t T) HijackContinue() {
	s := t.Serve().Route("/", ".html", `<body>ok</body>`)

	router := t.page.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
		wg.Done()
	})

	go router.Run()

	t.page.MustNavigate(s.URL())

	t.Eq("ok", t.page.MustElement("body").MustText())
	wg.Wait()
}

func (t T) HijackOnErrorLog() {
	s := t.Serve().Route("/", ".html", `<body>ok</body>`)

	router := t.page.HijackRequests()
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

	t.mc.stub(1, proto.FetchContinueRequest{}, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), errors.New("err")
	})

	go func() {
		_ = t.page.Context(t.Context()).Navigate(s.URL())
	}()
	wg.Wait()

	t.Eq(err.Error(), "err")
}

func (t T) HijackFailRequest() {
	s := t.Serve().Route("/", ".html", `<html>
	<body></body>
	<script>
		fetch('/a').catch(async (err) => {
			document.title = err.message
		})
	</script></html>`)

	router := t.browser.HijackRequests()
	defer router.MustStop()

	err := make(chan error)
	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) {
		ctx.OnError = func(e error) { err <- e }
		ctx.Response.Fail(proto.NetworkErrorReasonAborted)
	})

	go router.Run()

	t.page.MustNavigate(s.URL()).MustWaitLoad()

	t.page.MustWait(`document.title == 'Failed to fetch'`)

	{ // test error log
		t.mc.stubErr(1, proto.FetchFailRequest{})
		t.page.MustNavigate(s.URL())
		t.Err(<-err)
	}
}

func (t T) HijackLoadResponseErr() {
	p := t.page.Context(t.Context())
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd("*", func(ctx *rod.Hijack) {
		t.Err(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{err: errors.New("err")},
		}, true))

		t.Err(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{res: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(&MockReader{err: errors.New("err")}),
			}},
		}, true))

		wg.Done()
	})

	go router.Run()

	go func() { _ = p.Navigate(t.srcFile("./fixtures/click.html")) }()

	wg.Wait()
}

func (t T) HijackResponseErr() {
	s := t.Serve().Route("/", ".html", `ok`)

	p := t.page.Context(t.Context())
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd(s.URL("/a"), func(ctx *rod.Hijack) { // to ignore favicon
		ctx.OnError = func(err error) {
			t.Err(err)
			wg.Done()
		}

		ctx.MustLoadResponse()
		t.mc.stubErr(1, proto.FetchFulfillRequest{})
	})

	go router.Run()

	go func() { _ = p.Navigate(s.URL("/a")) }()

	wg.Wait()
}

func (t T) HandleAuth() {
	s := t.Serve()

	// mock the server
	s.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="web"`)
			w.WriteHeader(401)
			return
		}

		t.Eq("a", u)
		t.Eq("b", p)
		t.HandleHTTP(".html", `<p>ok</p>`)(w, r)
	})
	s.Route("/err", ".html", "err page")

	t.browser.MustHandleAuth("a", "b")

	page := t.newPage(s.URL())
	page.MustElementR("p", "ok")

	wait := t.browser.HandleAuth("a", "b")
	var page2 *rod.Page
	wait2 := utils.All(func() {
		page2, _ = t.browser.Page(s.URL("/err"))
	})
	t.mc.stubErr(1, proto.FetchContinueRequest{})
	t.Err(wait())
	wait2()
	page2.MustClose()
}

func (t T) GetDownloadFile() {
	s := t.Serve()
	content := "test content"

	s.Route("/d", ".bin", []byte(content))
	s.Route("/", ".html", fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, s.URL()))

	page := t.page.MustNavigate(s.URL())

	wait := page.MustGetDownloadFile(s.URL("/d")) // the pattern is used to prevent favicon request
	page.MustElement("a").MustClick()
	data := wait()

	t.Eq(content, string(data))

	waitErr := page.GetDownloadFile(s.URL("/d"), "", &http.Client{
		Transport: &MockRoundTripper{err: errors.New("err")},
	})
	page.MustElement("a").MustClick()
	{
		t.mc.stubErr(1, proto.FetchEnable{})
		_, _, err := waitErr()
		t.Err(err)
	}
	_, _, err := waitErr()
	t.Err(err)
}

func (t T) GetDownloadFileFromDataURI() {
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

	wait := page.MustGetDownloadFile("data:*")
	page.MustElement("#a").MustClick()
	data := wait()
	t.Eq("test data", string(data))

	wait = page.MustGetDownloadFile("data:*")
	page.MustElement("#b").MustClick()
	data = wait()
	t.Eq("test blob", string(data))

	t.Panic(func() {
		wait = page.MustGetDownloadFile("data:*")
		page.MustElement("#b").MustClick()
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		data = wait()
	})
}

func (t T) GetDownloadFileWithHijack() {
	s := t.Serve()
	content := "test content"

	s.Route("/d", ".bin", []byte(content))
	s.Route("/", ".html", fmt.Sprintf(`<html><a href="%s" download>click</a></html>`, s.URL("/d")))

	page := t.page.MustNavigate(s.URL())

	r := page.HijackRequests()
	r.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.OnError = func(error) {}
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})
	go r.Run()
	defer r.MustStop()

	wait := page.MustGetDownloadFile(s.URL("/d")) // the pattern is used to prevent favicon request
	page.MustElement("a").MustClick()
	data := wait()

	t.Eq(content, string(data))
}
