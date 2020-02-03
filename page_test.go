package rod_test

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

func (s *S) TestClosePage() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()
	page.Element("button")
}

func (s *S) TestPageElements() {
	s.page.Navigate(s.htmlFile("fixtures/input.html"))

	s.page.Element("input")

	list := s.page.Elements("input")
	s.Equal("submit", list[1].Eval("() => this.value").String())
}

func (s *S) TestPages() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
	s.Equal("click me", pages[0].Element("button").Text())
}

func (s *S) TestUntilPage() {
	page := s.page.Navigate(s.htmlFile("fixtures/open-page.html"))

	go page.Element("a").Click()

	newPage := s.browser.UntilPage(page)

	s.Equal("click me", newPage.Element("button").Text())
}

func (s *S) TestAlert() {
	page := s.page.Navigate(s.htmlFile("fixtures/alert.html"))

	go func() {
		_, err := s.browser.Event().Until(context.Background(), func(e kit.Event) bool {
			msg := e.(*cdp.Message)
			return msg.Method == "Page.javascriptDialogOpening"
		})
		kit.E(err)
		page.HandleDialog(true, "")
	}()

	page.Element("button").Click()
}
