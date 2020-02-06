package rod_test

import "time"

func (s *S) TestBrowserPages() {
	page := s.browser.Timeout(time.Minute).Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
}
