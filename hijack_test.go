package rod_test

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/kit"
)

func (s *S) TestHijack() {
	url, engine, close := serve()
	defer close()

	// to simulate a backend server
	engine.GET("/", ginHTMLFile("fixtures/fetch.html"))
	engine.POST("/a", func(ctx kit.GinContext) {
		s.Equal("header", ctx.GetHeader("Test"))

		b, err := ioutil.ReadAll(ctx.Request.Body)
		utils.E(err)
		s.Equal("a", string(b))

		ginString("test")(ctx)
	})
	engine.GET("/b", ginString("b"))

	router := s.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd(url+"/a", func(ctx *rod.Hijack) {
		ctx.Request.
			SetClient(&http.Client{
				Transport: &http.Transport{
					DisableKeepAlives: true,
				},
			}).                                 // customize http client
			SetMethod(ctx.Request.Method()).    // override request method
			SetURL(ctx.Request.URL().String()). // override request url
			SetQuery("a", "b").
			SetHeader("Test", "header"). // override request header
			SetBody(0).
			SetBody([]byte("")).
			SetBody(ctx.Request.Body()) // override request body

		s.Equal(proto.NetworkResourceTypeXHR, ctx.Request.Type())
		s.Contains(ctx.Request.Header("Origin"), url)
		s.Len(ctx.Request.Headers(), 5)
		s.Equal("", ctx.Request.JSONBody().String())

		// send request load response from real destination as the default value to hijack
		ctx.MustLoadResponse()

		s.Equal(200, ctx.Response.MustStatusCode())

		// override status code
		ctx.Response.SetStatusCode(201)

		s.Equal("4", ctx.Response.MustHeader("Content-Length"))
		s.Equal("text/plain; charset=utf-8", ctx.Response.MustHeaders().Get("Content-Type"))

		// override response header
		ctx.Response.SetHeader("Set-Cookie", "key=val")

		// override response body
		ctx.Response.SetBody("").SetBody(map[string]string{
			"text": ctx.Response.StringBody(),
		})

		s.NotNil(ctx.Response.MustBodyStream())
		s.Equal("true", ctx.Response.JSONBody().String())
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
	url, engine, close := serve()
	defer close()

	// to simulate a backend server
	engine.GET("/", ginHTML(`<html>
	<body></body>
	<script>
		fetch('/a').then(async (res) => {
			document.body.innerText = await res.text()
		})
	</script></html>`))
	engine.GET("/a", ginString(`ok`))

	router := s.page.HijackRequests()
	defer router.MustStop()

	router.MustAdd(url+"/a", func(ctx *rod.Hijack) {
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()

	s.page.MustNavigate(url)

	s.Equal("ok", s.page.MustElement("body").MustText())
}

func (s *S) TestHijackFailRequest() {
	url, engine, close := serve()
	defer close()

	// to simulate a backend server
	engine.GET("/", ginHTML(`<html>
	<body></body>
	<script>
		fetch('/a').catch(async (err) => {
			document.body.innerText = err.message
		})
	</script></html>`))

	router := s.browser.HijackRequests()
	defer router.MustStop()

	router.MustAdd(url+"/a", func(ctx *rod.Hijack) {
		ctx.Response.Fail(proto.NetworkErrorReasonAborted)
	})

	go router.Run()

	s.page.MustNavigate(url)

	s.Equal("Failed to fetch", s.page.MustElement("body").MustText())
}

func (s *S) TestHandleAuth() {
	url, engine, close := serve()
	defer close()

	// mock the server
	engine.NoRoute(func(ctx kit.GinContext) {
		u, p, ok := ctx.Request.BasicAuth()
		if !ok {
			ctx.Header("WWW-Authenticate", `Basic realm="web"`)
			ctx.Writer.WriteHeader(401)
			return
		}

		s.Equal("a", u)
		s.Equal("b", p)
		ginHTML(`<p>ok</p>`)(ctx)
	})

	s.browser.MustHandleAuth("a", "b")

	page := s.browser.MustPage(url)
	defer page.MustClose()
	page.MustElementMatches("p", "ok")
}

func (s *S) TestGetDownloadFile() {
	url, engine, close := serve()
	defer close()

	content := "test content"

	engine.GET("/d", func(ctx kit.GinContext) {
		utils.E(ctx.Writer.Write([]byte(content)))
	})
	engine.GET("/", ginHTML(fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, url)))

	page := s.page.MustNavigate(url)

	wait := page.MustGetDownloadFile(url + "/d") // the pattern is used to prevent favicon request
	page.MustElement("a").MustClick()
	data := wait()

	s.Equal(content, string(data))
}

func (s *S) TestGetDownloadFileFromDataURI() {
	url, engine, close := serve()
	defer close()

	engine.GET("/", ginHTML(
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
}

func (s *S) TestGetDownloadFileWithHijack() {
	url, engine, close := serve()
	defer close()

	content := "test content"

	engine.GET("/d", func(ctx kit.GinContext) {
		utils.E(ctx.Writer.Write([]byte(content)))
	})
	engine.GET("/", ginHTML(fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, url)))

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
