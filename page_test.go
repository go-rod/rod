package rod_test

func (s *S) TestClosePage() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	page.Element("button")
	page.Close()
}

func (s *S) TestPage() {
	s.page.Navigate(s.htmlFile("fixtures/click.html"))

	s.Equal("<h4>Title</h4>", s.page.Element("h4").HTML())
	s.Equal("Title", s.page.Element("h4").Text())
}
