package rod_test

import (
	"fmt"
	"io/ioutil"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
)

func (s *S) TestHijack() {
	url, engine, close := serve()
	defer close()

	// to simulate a backend server
	engine.GET("/", ginHTMLFile("fixtures/fetch.html"))
	engine.POST("/a", func(ctx kit.GinContext) {
		s.Equal("header", ctx.GetHeader("Test"))

		b, err := ioutil.ReadAll(ctx.Request.Body)
		kit.E(err)
		s.Equal("a", string(b))

		ginString("test")(ctx)
	})
	engine.GET("/b", ginString("b"))

	router := s.page.HijackRequests()
	defer router.Stop()

	router.Add(url+"/a", func(ctx *rod.Hijack) {
		// override request method
		ctx.Request.SetMethod(ctx.Request.Method())

		// override request url
		ctx.Request.SetURL(ctx.Request.URL().String())

		// override request header
		ctx.Request.SetHeader("Test", "header")

		// override request body
		ctx.Request.SetBody(ctx.Request.Body())

		// send request load response from real destination as the default value to hijack
		ctx.LoadResponse()

		s.Equal(200, ctx.Response.StatusCode())

		// override status code
		ctx.Response.SetStatusCode(201)

		s.Equal("4", ctx.Response.Header("Content-Length"))
		s.Equal("text/plain; charset=utf-8", ctx.Response.Headers().Get("Content-Type"))

		// override response header
		ctx.Response.SetHeader("Set-Cookie", "key=val")

		// override response body
		ctx.Response.SetBody(map[string]string{
			"text": ctx.Response.StringBody(),
		})
	})

	router.Add(url+"/b", func(ctx *rod.Hijack) {
		panic("should not come to here")
	})
	router.Remove(url + "/b")

	router.Add(url+"/b", func(ctx *rod.Hijack) {
		// transparent proxy
		ctx.LoadResponse()
	})

	go router.Run()

	s.page.Navigate(url)

	s.Equal("201 test key=val", s.page.Element("#a").Text())
	s.Equal("b", s.page.Element("#b").Text())
}

func (s *S) TestBrowserHandleAuth() {
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

	s.browser.HandleAuth("a", "b")

	page := s.browser.Page(url)
	defer page.Close()
	page.ElementMatches("p", "ok")
}

func (s *S) TestDownloadFile() {
	url, engine, close := serve()
	defer close()

	content := "test content"

	engine.GET("/d", func(ctx kit.GinContext) {
		kit.E(ctx.Writer.Write([]byte(content)))
	})
	engine.GET("/", ginHTML(fmt.Sprintf(`<html><a href="%s/d" download>click</a></html>`, url)))

	page := s.page.Navigate(url)

	wait := page.GetDownloadFile(url + "/d") // the pattern is used to prevent favicon request
	page.Element("a").Click()
	data := wait()

	s.Equal(content, string(data))
}
