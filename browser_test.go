package rod_test

func (s *S) TestBrowserPages() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
}
