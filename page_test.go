package rod_test

import (
	"fmt"

	"github.com/ysmood/kit"
)

func (s *S) TestClosePage() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()
	page.Element("button")
}

func (s *S) TestPageElements() {
	s.page.Navigate(s.htmlFile("fixtures/input.html"))
	list := s.page.Elements("input")
	s.Equal("submit", list[2].Eval("() => this.value").String())
}

func (s *S) TestPages() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
}

func (s *S) TestUntilPage() {
	page := s.page.Navigate(s.htmlFile("fixtures/open-page.html"))

	wait := kit.All(func() {
		page.Element("a").Click()
	})

	newPage := page.WaitPage()

	s.Equal("click me", newPage.Element("button").Text())

	wait()
}

func (s *S) TestAlert() {
	page := s.page.Navigate(s.htmlFile("fixtures/alert.html"))

	wait := kit.All(func() {
		page.Element("button").Click()
	})

	page.HandleDialog(true, "")

	wait()
}

func (s *S) TestDownloadFile() {
	srv := kit.MustServer(":0")
	host := srv.Listener.Addr().String()
	content := "test content"

	srv.Engine.GET("/d", func(ctx kit.GinContext) {
		ctx.Writer.WriteHeader(200)
		kit.E(ctx.Writer.Write([]byte(content)))
	})
	srv.Engine.GET("/", func(ctx kit.GinContext) {
		ctx.Header("Content-Type", "text/html;")
		data := []byte(fmt.Sprintf(`<html><a href="//%s/d" download>click</a></html>`, host))
		kit.E(ctx.Writer.Write(data))
	})

	go srv.MustDo()

	page := s.page.Navigate("http://" + host)

	wait := kit.All(func() {
		page.Element("a").Click()
	})

	_, data := page.GetDownloadFile("")

	s.Equal(content, string(data))
	wait()
}

func (s *S) TestDrag() {
	s.T().Skip("not able to use mouse event to simulate it for now")

	page := s.page.Navigate(s.htmlFile("fixtures/drag.html"))
	mouse := page.Mouse

	mouse.Move(60, 30)
	mouse.Down("left")
	kit.E(mouse.MoveE(60, 80, 5))
	mouse.Up("left")

	page.Element(".dropzone:nth-child(2) #draggable")
}
