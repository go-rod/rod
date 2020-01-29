package rod_test

func (s *S) TestClick() {
	p := s.page.Navigate(s.htmlFile("fixtures/click.html"))
	p.Element("button").Click()

	s.True(p.Has("[a=ok]"))
}

func (s *S) TestClickInIframes() {
	p := s.page.Navigate(s.htmlFile("fixtures/click-iframes.html"))
	p.Element("iframe").Frame().Element("iframe").Frame().Element("button").Click()

	s.True(p.Has("[a=ok]"))
}

func (s *S) TestPress() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("[type=text]")
	el.Press("a")

	s.Equal("a", el.Func(`function() { return this.value }`).String())
	s.True(p.Has("[event=input-change]"))
}

func (s *S) TestText() {
	text := "雲の上は\nいつも晴れ"

	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("textarea")
	el.Text(text)

	s.Equal(text, el.Func(`function() { return this.value }`).String())
	s.True(p.Has("[event=textarea-change]"))
}

func (s *S) TestSelect() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("select")
	el.Select("[value=c]")

	s.EqualValues(2, el.Func("function() { return this.selectedIndex }").Int())
}

func (s *S) TestEnter() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("[type=submit]")
	el.Press("Enter")

	s.True(p.Has("[event=submit]"))
}
