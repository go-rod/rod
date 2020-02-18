package rod_test

import (
	"time"

	"github.com/ysmood/rod/lib/cdp"
)

func (s *S) TestBrowserPages() {
	page := s.browser.Timeout(time.Minute).Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
}

func (s *S) TestBrowserWaitEvent() {
	wait, cancel := s.browser.WaitEvent("Page.frameNavigated")
	defer cancel()
	s.page.Navigate(s.htmlFile("fixtures/click.html"))
	wait()
}

func (s *S) TestBrowserCall() {
	v := s.browser.Call(&cdp.Request{
		Method: "Browser.getVersion",
	})

	s.Regexp("HeadlessChrome", v.Get("product").String())
}
