package rod_test

func (s *S) TestClick() {
	p := s.page.Navigate(s.htmlFile("click.html"))
	p.Element("button").Click()

	s.True(p.Has("[a=ok]"))
}

func (s *S) TestClickInIframes() {
	p := s.page.Navigate(s.htmlFile("click-iframes.html"))
	p.Element("iframe").Frame().Element("iframe").Frame().Element("button").Click()

	s.True(p.Has("[a=ok]"))
}
