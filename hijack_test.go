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
)

func (s *S) TestHijack() {
	url, mux, close := utils.Serve("")
	defer close()

	// to simulate a backend server
	mux.HandleFunc("/", httpHTMLFile("fixtures/fetch.html"))
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			panic("wrong http method")
		}

		s.Equal("header", r.Header.Get("Test"))

		b, err := ioutil.ReadAll(r.Body)
		utils.E(err)
		s.Equal("a", string(b))

		httpString("test")(w, r)
	})
	mux.HandleFunc("/b", httpString("b"))

	router := s.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd(url+"/a", func(ctx *rod.Hijack) {
		r := ctx.Request
		r.Req().URL = r.Req().URL            // override request url
		r.Req().Header.Set("Test", "header") // override request header
		r.SetBody([]byte("test"))            // override request body
		r.SetBody(123)                       // override request body
		r.SetBody(r.Body())                  // override request body

		s.Equal(http.MethodPost, r.Method())
		s.Equal(url+"/a", r.URL().String())

		s.Equal(proto.NetworkResourceTypeXHR, ctx.Request.Type())
		s.Contains(ctx.Request.Header("Origin"), url)
		s.Len(ctx.Request.Headers(), 5)
		s.Equal("", ctx.Request.JSONBody().String())

		// send request load response from real destination as the default value to hijack
		ctx.MustLoadResponse()

		s.EqualValues(200, ctx.Response.Payload().ResponseCode)

		// override status code
		ctx.Response.Payload().ResponseCode = http.StatusCreated

		s.Equal("4", ctx.Response.Headers().Get("Content-Length"))
		s.Equal("text/plain; charset=utf-8", ctx.Response.Headers().Get("Content-Type"))

		// override response header
		ctx.Response.SetHeader("Set-Cookie", "key=val")

		// override response body
		ctx.Response.SetBody([]byte("test"))
		ctx.Response.SetBody("test")
		ctx.Response.SetBody(map[string]string{
			"text": "test",
		})

		s.Equal("{\"text\":\"test\"}", ctx.Response.Body())
	})

	router.MustAdd(url+"/b", func(ctx *rod.Hijack) {
		panic("should not come to here")
	})
	router.MustRemove(url + "/b")

	router.MustAdd(url+"/b", func(ctx *rod.Hijack) {
		// transparent proxy
		ctx.MustLoadResponse()
	})

	go router.Run()

	s.page.MustNavigate(url)

	s.Equal("201 test key=val", s.page.MustElement("#a").MustText())
	s.Equal("b", s.page.MustElement("#b").MustText())
}

func (s *S) TestHijackContinue() {
	url, mux, close := utils.Serve("")
	defer close()

	mux.HandleFunc("/", httpHTML(`<body>ok</body>`))

	router := s.page.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
		wg.Done()
	})

	go router.Run()

	s.page.MustNavigate(url)

	s.Equal("ok", s.page.MustElement("body").MustText())

	func() { // test error log
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.errorAt(1, proto.FetchContinueRequest{})
		go func() {
			_ = s.page.Context(ctx).Navigate(url)
		}()
		wg.Wait()
	}()
}

func (s *S) TestHijackFailRequest() {
	url, mux, close := utils.Serve("")
	defer close()

	// to simulate a backend server
	mux.HandleFunc("/", httpHTML(`<html>
	<body></body>
	<script>
		fetch('/a').catch(async (err) => {
			document.body.innerText = err.message
		})
	</script></html>`))

	router := s.browser.HijackRequests()
	defer router.MustStop()

	err := make(chan error)
	router.MustAdd(url+"/a", func(ctx *rod.Hijack) {
		ctx.OnError = func(e error) { err <- e }
		ctx.Response.Fail(proto.NetworkErrorReasonAborted)
	})

	go router.Run()

	s.page.MustNavigate(url)

	s.Equal("Failed to fetch", s.page.MustElement("body").MustText())

	{ // test error log
		s.errorAt(1, proto.FetchFailRequest{})
		s.page.MustNavigate(url)
		s.Error(<-err)
	}
}

func (s *S) TestHijackLoadResponseErr() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := s.page.Context(ctx)
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd("*", func(ctx *rod.Hijack) {
		s.Error(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{err: errors.New("err")},
		}, true))

		s.Error(ctx.LoadResponse(&http.Client{
			Transport: &MockRoundTripper{res: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(&MockReader{err: errors.New("err")}),
			}},
		}, true))

		wg.Done()
	})

	go router.Run()

	go func() { _ = p.Navigate(srcFile("./fixtures/click.html")) }()

	wg.Wait()
}

func (s *S) TestHijackResponseErr() {
	url, mux, close := utils.Serve("")
	defer close()

	// to simulate a backend server
	mux.HandleFunc("/", httpHTML(`ok`))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := s.page.Context(ctx)
	router := p.HijackRequests()
	defer router.MustStop()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.OnError = func(err error) {
			s.Error(err)
			wg.Done()
		}

		ctx.MustLoadResponse()
		s.errorAt(1, proto.FetchFulfillRequest{})
	})

	go router.Run()

	go func() { _ = p.Navigate(url) }()

	wg.Wait()
}

func (s *S) TestHandleAuth() {
	url, mux, close := utils.Serve("")
	defer close()

	// mock the server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="web"`)
			w.WriteHeader(401)
			return
		}

		s.Equal("a", u)
		s.Equal("b", p)
		httpHTML(`<p>ok</p>`)(w, r)
	})

	s.browser.MustHandleAuth("a", "b")

	page := s.browser.MustPage(url)
	defer page.MustClose()
	page.MustElementMatches("p", "ok")

	wait := s.browser.HandleAuth("a", "b")
	go func() { _, _ = s.browser.Page(url) }()
	utils.Sleep(0.1)
	s.errorAt(1, proto.FetchContinueRequest{})
	s.Error(wait())
}

func (s *S) TestGetDownloadFile() {
	url, mux, close := utils.Serve("")
	defer close()

	content := "test content"

	mux.HandleFunc("/d", func(w http.ResponseWriter, r *http.Request) {
		utils.E(w.Write([]byte(content)))
	})
	mux.HandleFunc("/", httpHTML(fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, url)))

	page := s.page.MustNavigate(url)

	wait := page.MustGetDownloadFile(url + "/d") // the pattern is used to prevent favicon request
	page.MustElement("a").MustClick()
	data := wait()

	s.Equal(content, string(data))

	waitErr := page.GetDownloadFile(url+"/d", "", &http.Client{
		Transport: &MockRoundTripper{err: errors.New("err")},
	})
	page.MustElement("a").MustClick()
	{
		s.errorAt(1, proto.FetchEnable{})
		_, _, err := waitErr()
		s.Error(err)
	}
	_, _, err := waitErr()
	s.Error(err)
}

func (s *S) TestGetDownloadFileFromDataURI() {
	url, mux, close := utils.Serve("")
	defer close()

	mux.HandleFunc("/", httpHTML(
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
	))

	page := s.page.MustNavigate(url)

	wait := page.MustGetDownloadFile("data:*")
	page.MustElement("#a").MustClick()
	data := wait()
	s.Equal("test data", string(data))

	wait = page.MustGetDownloadFile("data:*")
	page.MustElement("#b").MustClick()
	data = wait()
	s.Equal("test blob", string(data))

	s.Panics(func() {
		wait = page.MustGetDownloadFile("data:*")
		page.MustElement("#b").MustClick()
		s.errorAt(1, proto.RuntimeCallFunctionOn{})
		data = wait()
	})
}

func (s *S) TestGetDownloadFileWithHijack() {
	url, mux, close := utils.Serve("")
	defer close()

	content := "test content"

	mux.HandleFunc("/d", func(w http.ResponseWriter, r *http.Request) {
		utils.E(w.Write([]byte(content)))
	})
	mux.HandleFunc("/", httpHTML(fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, url)))

	page := s.page.MustNavigate(url)

	r := page.HijackRequests()
	r.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.OnError = func(error) {}
		ctx.MustLoadResponse()
	})
	go r.Run()
	defer r.MustStop()

	wait := page.MustGetDownloadFile(url + "/d") // the pattern is used to prevent favicon request
	page.MustElement("a").MustClick()
	data := wait()

	s.Equal(content, string(data))
}
