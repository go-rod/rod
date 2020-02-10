package rod_test

func (s *S) TestPageElements() {
	s.page.Navigate(s.htmlFile("fixtures/input.html"))
	s.page.Element("input")
	list := s.page.Elements("input")
	s.Equal("submit", list[2].Eval("() => this.value").String())
}

func (s *S) TestPageElementX() {
	s.page.Navigate(s.htmlFile("fixtures/click.html"))
	s.page.Element("body")
	name := s.page.ElementX("//*[contains(text(), 'click')]").Describe().Get("localName").String()
	s.Equal("button", name)
}

func (s *S) TestPageElementsX() {
	s.page.Navigate(s.htmlFile("fixtures/input.html"))
	s.page.Element("body")
	list := s.page.ElementsX("//input")
	s.Len(list, 3)
}

func (s *S) TestElementMatches() {
	p := s.page.Navigate(s.htmlFile("fixtures/selector.html"))
	el := p.ElementMatches("button", `\d1`)
	s.Equal("01", el.Text())

	el = p.Element("div").ElementMatches("button", `03`)
	s.Equal("03", el.Text())
}

func (s *S) TestElementFromElement() {
	p := s.page.Navigate(s.htmlFile("fixtures/selector.html"))
	el := p.Element("div").Element("button")
	s.Equal("02", el.Text())
}

func (s *S) TestElementsFromElement() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	p.Element("form")
	list := p.Element("form").Elements("option")

	s.Len(list, 3)
	s.Equal("B", list[1].Text())
}

func (s *S) TestElementParent() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("input").Parent()
	s.Equal("FORM", el.Eval(`() => this.tagName`).String())
}

func (s *S) TestElementSiblings() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("hr")
	a := el.Previous()
	b := el.Next()

	s.Equal("INPUT", a.Eval(`() => this.tagName`).String())
	s.Equal("SELECT", b.Eval(`() => this.tagName`).String())
}

func (s *S) TestElementFromElementX() {
	p := s.page.Navigate(s.htmlFile("fixtures/selector.html"))
	el := p.Element("div").ElementX("./button")
	s.Equal("02", el.Text())
}

func (s *S) TestElementsFromElementsX() {
	p := s.page.Navigate(s.htmlFile("fixtures/selector.html"))
	list := p.Element("div").ElementsX("./button")
	s.Len(list, 2)
}
