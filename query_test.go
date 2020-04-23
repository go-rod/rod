package rod_test

import (
	"github.com/ysmood/rod"
)

func (s *S) TestPageElements() {
	s.page.Navigate(srcFile("fixtures/input.html"))
	s.page.Element("input")
	list := s.page.Elements("input")
	s.Equal("input", list.First().Describe().Get("localName").String())
	s.Equal("submit", list.Last().Text())
}

func (s *S) TestPages() {
	s.page.Navigate(srcFile("fixtures/click.html"))

	s.True(s.browser.Pages().Find("button").Has("button"))
	s.True(s.browser.Pages().FindByURL("click.html").Has("button"))

	s.Nil(s.browser.Pages().Find("____"))
	s.Nil(s.browser.Pages().FindByURL("____"))
}

func (s *S) TestPageHas() {
	s.page.Navigate(srcFile("fixtures/selector.html"))
	s.page.Element("body")
	s.True(s.page.Has("span"))
	s.False(s.page.Has("a"))
	s.True(s.page.HasX("//span"))
	s.False(s.page.HasX("//a"))
	s.True(s.page.HasMatches("button", "03"))
	s.False(s.page.HasMatches("button", "11"))
}

func (s *S) TestElementHas() {
	s.page.Navigate(srcFile("fixtures/selector.html"))
	b := s.page.Element("body")
	s.True(b.Has("span"))
	s.False(b.Has("a"))
	s.True(b.HasX("//span"))
	s.False(b.HasX("//a"))
	s.True(b.HasMatches("button", "03"))
	s.False(b.HasMatches("button", "11"))
}

func (s *S) TestPageElementX() {
	s.page.Navigate(srcFile("fixtures/click.html"))
	s.page.Element("body")
	name := s.page.ElementX("//*[contains(text(), 'click')]").Describe().Get("localName").String()
	s.Equal("button", name)
}

func (s *S) TestPageElementsX() {
	s.page.Navigate(srcFile("fixtures/input.html"))
	s.page.Element("body")
	list := s.page.ElementsX("//input")
	s.Len(list, 3)
}

func (s *S) TestElementMatches() {
	p := s.page.Navigate(srcFile("fixtures/selector.html"))
	el := p.ElementMatches("button", `\d1`)
	s.Equal("01", el.Text())

	el = p.Element("div").ElementMatches("button", `03`)
	s.Equal("03", el.Text())

	p = s.page.Navigate(srcFile("fixtures/input.html"))
	el = p.ElementMatches("input", `submit`)
	s.Equal("submit", el.Text())
}

func (s *S) TestElementFromElement() {
	p := s.page.Navigate(srcFile("fixtures/selector.html"))
	el := p.Element("div").Element("button")
	s.Equal("02", el.Text())
}

func (s *S) TestElementsFromElement() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	p.Element("form")
	list := p.Element("form").Elements("option")

	s.Len(list, 4)
	s.Equal("B", list[1].Text())
}

func (s *S) TestElementParent() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("input").Parent()
	s.Equal("FORM", el.Eval(`() => this.tagName`).String())
}

func (s *S) TestElementParents() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	s.Len(p.Element("option").Parents("*"), 4)
	s.Len(p.Element("option").Parents("form"), 1)
}

func (s *S) TestElementSiblings() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("hr")
	a := el.Previous()
	b := el.Next()

	s.Equal("INPUT", a.Eval(`() => this.tagName`).String())
	s.Equal("SELECT", b.Eval(`() => this.tagName`).String())
}

func (s *S) TestElementFromElementX() {
	p := s.page.Navigate(srcFile("fixtures/selector.html"))
	el := p.Element("div").ElementX("./button")
	s.Equal("02", el.Text())
}

func (s *S) TestElementsFromElementsX() {
	p := s.page.Navigate(srcFile("fixtures/selector.html"))
	list := p.Element("div").ElementsX("./button")
	s.Len(list, 2)
}

func (s *S) TestElementTracing() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	s.Equal(`rod.element("code")`, p.Element("code").Text())
}

func (s *S) TestPageElementByJS_Err() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	_, err := p.ElementByJSE(p.Sleeper(), "", `() => 1`, nil)
	s.EqualError(err, "[rod] expect js to return an element\n{\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
}

func (s *S) TestPageElementsByJS_Err() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	_, err := p.ElementsByJSE("", `() => [1]`, nil)
	s.EqualError(err, "[rod] expect js to return an array of elements\n{\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJSE("", `() => 1`, nil)
	s.EqualError(err, "[rod] expect js to return an array of elements\n{\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJSE("", `() => foo()`, nil)
	s.Error(err)
}

func (s *S) TestElementsOthers() {
	list := &rod.Elements{}
	s.Nil(list.First())
	s.Nil(list.Last())
}
