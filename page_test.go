package rod_test

func (s *S) TestClosePage() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()
	page.Element("button")
}

func (s *S) TestPageElements() {
	s.page.Navigate(s.htmlFile("fixtures/input.html"))
	list := s.page.Elements("input")
	s.Equal("submit", list[2].Eval("() => this.value").String())
}

func (s *S) TestPages() {
	page := s.browser.Page(s.htmlFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
	s.Equal("click me", pages[0].Element("button").Text())
}

func (s *S) TestUntilPage() {
	page := s.page.Navigate(s.htmlFile("fixtures/open-page.html"))

	go page.Element("a").Click()

	newPage := page.WaitPage()

	s.Equal("click me", newPage.Element("button").Text())
}

func (s *S) TestAlert() {
	page := s.page.Navigate(s.htmlFile("fixtures/alert.html"))

	go page.Element("button").Click()

	page.WaitEvent("Page.javascriptDialogOpening")
	page.HandleDialog(true, "")
}
