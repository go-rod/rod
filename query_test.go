package rod_test

import (
	"errors"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/defaults"
)

func (s *S) TestPageElements() {
	s.page.MustNavigate(srcFile("fixtures/input.html"))
	s.page.MustElement("input")
	list := s.page.MustElements("input")
	s.Equal("input", list.First().MustDescribe().LocalName)
	s.Equal("submit", list.Last().MustText())
}

func (s *S) TestPages() {
	s.page.MustNavigate(srcFile("fixtures/click.html"))

	s.True(s.browser.MustPages().Find("button").MustHas("button"))
	s.True(s.browser.MustPages().MustFindByURL("click.html").MustHas("button"))

	s.Nil(s.browser.MustPages().Find("____"))
	s.Nil(s.browser.MustPages().MustFindByURL("____"))
}

func (s *S) TestPageHas() {
	s.page.MustNavigate(srcFile("fixtures/selector.html"))
	s.page.MustElement("body")
	s.True(s.page.MustHas("span"))
	s.False(s.page.MustHas("a"))
	s.True(s.page.MustHasX("//span"))
	s.False(s.page.MustHasX("//a"))
	s.True(s.page.MustHasMatches("button", "03"))
	s.False(s.page.MustHasMatches("button", "11"))
}

func (s *S) TestPageRaceElement() {
	s.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := s.page.MustElement("p", "button")
	s.Equal("01", el.MustText())
}

func (s *S) TestElementHas() {
	s.page.MustNavigate(srcFile("fixtures/selector.html"))
	b := s.page.MustElement("body")
	s.True(b.MustHas("span"))
	s.False(b.MustHas("a"))
	s.True(b.MustHasX("//span"))
	s.False(b.MustHasX("//a"))
	s.True(b.MustHasMatches("button", "03"))
	s.False(b.MustHasMatches("button", "11"))
}

func (s *S) TestSearch() {
	p := s.page.MustNavigate(srcFile("fixtures/click.html"))
	el := p.MustSearch("click me")
	s.Equal("click me", el.MustText())
	s.True(el.MustClick().MustMatches("[a=ok]"))

	_, err := p.Sleeper(nil).Search([]string{"not-exists"}, 0, 1)
	s.True(errors.Is(err, rod.ErrElementNotFound))
}

func (s *S) TestSearchIframes() {
	p := s.page.MustNavigate(srcFile("fixtures/click-iframes.html"))
	el := p.MustSearch("button[onclick]")
	s.Equal("click me", el.MustText())
	s.True(el.MustClick().MustMatches("[a=ok]"))
}

func (s *S) TestSearchIframesAfterReload() {
	p := s.page.MustNavigate(srcFile("fixtures/click-iframes.html"))
	frame := p.MustElement("iframe").Frame().MustElement("iframe").Frame()
	frame.MustEval(`location.reload()`)
	el := p.MustSearch("button[onclick]")
	s.Equal("click me", el.MustText())
	s.True(el.MustClick().MustMatches("[a=ok]"))
}

func (s *S) TestPageElementX() {
	s.page.MustNavigate(srcFile("fixtures/click.html"))
	s.page.MustElement("body")
	name := s.page.MustElementX("//*[contains(text(), 'click')]").MustDescribe().LocalName
	s.Equal("button", name)
}

func (s *S) TestPageElementsX() {
	s.page.MustNavigate(srcFile("fixtures/input.html"))
	s.page.MustElement("body")
	list := s.page.MustElementsX("//input")
	s.Len(list, 5)
}

func (s *S) TestElementMatches() {
	p := s.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := p.MustElementMatches("button", `\d1`)
	s.Equal("01", el.MustText())

	el = p.MustElement("div").MustElementMatches("button", `03`)
	s.Equal("03", el.MustText())

	p = s.page.MustNavigate(srcFile("fixtures/input.html"))
	el = p.MustElementMatches("input", `submit`)
	s.Equal("submit", el.MustText())
}

func (s *S) TestElementFromElement() {
	p := s.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElement("button")
	s.Equal("02", el.MustText())
}

func (s *S) TestElementsFromElement() {
	p := s.page.MustNavigate(srcFile("fixtures/input.html"))
	p.MustElement("form")
	list := p.MustElement("form").MustElements("option")

	s.Len(list, 4)
	s.Equal("B", list[1].MustText())
}

func (s *S) TestElementParent() {
	p := s.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("input").MustParent()
	s.Equal("FORM", el.MustEval(`this.tagName`).String())
}

func (s *S) TestElementParents() {
	p := s.page.MustNavigate(srcFile("fixtures/input.html"))
	s.Len(p.MustElement("option").MustParents("*"), 4)
	s.Len(p.MustElement("option").MustParents("form"), 1)
}

func (s *S) TestElementSiblings() {
	p := s.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("hr")
	a := el.MustPrevious()
	b := el.MustNext()

	s.Equal("INPUT", a.MustEval(`this.tagName`).String())
	s.Equal("SELECT", b.MustEval(`this.tagName`).String())
}

func (s *S) TestElementFromElementX() {
	p := s.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElementX("./button")
	s.Equal("02", el.MustText())
}

func (s *S) TestElementsFromElementsX() {
	p := s.page.MustNavigate(srcFile("fixtures/selector.html"))
	list := p.MustElement("div").MustElementsX("./button")
	s.Len(list, 2)
}

func (s *S) TestElementTracing() {
	s.browser.Trace(true).Quiet(true)
	defer func() {
		s.browser.Trace(defaults.Trace).Quiet(defaults.Quiet)
	}()

	p := s.page.MustNavigate(srcFile("fixtures/click.html"))
	s.Equal(`rod.element("code")`, p.MustElement("code").MustText())
}

func (s *S) TestPageElementByJS_Err() {
	p := s.page.MustNavigate(srcFile("fixtures/click.html"))
	_, err := p.ElementByJS("", `1`, nil)
	s.EqualError(err, `{"type":"number","value":1,"description":"1"}: expect js to return an element`)
}

func (s *S) TestPageElementsByJS_Err() {
	p := s.page.MustNavigate(srcFile("fixtures/click.html"))
	_, err := p.ElementsByJS("", `[1]`, nil)
	s.EqualError(err, `{"type":"number","value":1,"description":"1"}: expect js to return an array of elements`)
	_, err = p.ElementsByJS("", `1`, nil)
	s.EqualError(err, `{"type":"number","value":1,"description":"1"}: expect js to return an array of elements`)
	_, err = p.ElementsByJS("", `foo()`, nil)
	s.Error(err)
}

func (s *S) TestElementsOthers() {
	list := &rod.Elements{}
	s.Nil(list.First())
	s.Nil(list.Last())
}
