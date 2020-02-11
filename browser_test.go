package rod_test

import (
	"time"

	"github.com/ysmood/kit"
)

func (s *S) TestBrowserPages() {
	page := s.browser.Timeout(time.Minute).Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
}

func (s *S) TestBrowserWaitEvent() {
	wait := kit.All(func() { s.browser.WaitEvent("Page.frameNavigated") })
	kit.Sleep(0.01)
	s.page.Navigate(s.htmlFile("fixtures/click.html"))
	wait()
}
