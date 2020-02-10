package rod_test

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/ysmood/rod/lib/input"
)

func (s *S) TestClick() {
	p := s.page.Navigate(s.htmlFile("fixtures/click.html"))
	p.Element("button").Click()

	s.True(p.Has("[a=ok]"))
}

func (s *S) TestClickInIframes() {
	p := s.page.Navigate(s.htmlFile("fixtures/click-iframes.html"))
	frame := p.Element("iframe").Frame().Element("iframe").Frame()
	frame.Element("button").Click()
	s.True(frame.Has("[a=ok]"))
}

func (s *S) TestPress() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("[type=text]")
	el.Press('A')
	el.Press(' ')
	el.Press('b')

	s.Equal("A b", el.Eval(`() => this.value`).String())
}

func (s *S) TestKeyDown() {
	p := s.page.Navigate(s.htmlFile("fixtures/keys.html"))
	p.Element("body")
	p.Keyboard.Down('j')

	s.True(p.Has("body[event=key-down-j]"))
}

func (s *S) TestKeyUp() {
	p := s.page.Navigate(s.htmlFile("fixtures/keys.html"))
	p.Element("body")
	p.Keyboard.Up('x')

	s.True(p.Has("body[event=key-up-x]"))
}

func (s *S) TestText() {
	text := "雲の上は\nいつも晴れ"

	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("textarea")
	el.Input(text)

	s.Equal(text, el.Eval(`() => this.value`).String())
	s.True(p.Has("[event=textarea-change]"))
}

func (s *S) TestSelect() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("select")
	el.Select("C")

	s.EqualValues(2, el.Eval("() => this.selectedIndex").Int())
}

func (s *S) TestSetFiles() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element(`[type=file]`)
	el.SetFiles(
		filepath.FromSlash("fixtures/click.html"),
		filepath.FromSlash("fixtures/alert.html"),
	)

	list := el.Eval("() => Array.from(this.files).map(f => f.name)").Array()
	s.Len(list, 2)
	s.Equal("alert.html", list[1].String())
}

func (s *S) TestSelectQuery() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("select")
	el.Select("[value=c]")

	s.EqualValues(2, el.Eval("() => this.selectedIndex").Int())
}

func (s *S) TestSelectQueryNum() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("select")
	el.Select("123")

	s.EqualValues(0, el.Eval("() => this.selectedIndex").Int())
}

func (s *S) TestEnter() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("[type=submit]")
	el.Press(input.Enter)

	s.True(p.Has("[event=submit]"))
}

func (s *S) TestWaitInvisible() {
	p := s.page.Navigate(s.htmlFile("fixtures/click.html"))
	h4 := p.Element("h4")
	btn := p.Element("button")
	timeout := 3 * time.Second

	h4t := h4.Timeout(timeout)
	h4t.WaitVisible()
	h4t.CancelTimeout()

	go func() {
		time.Sleep(30 * time.Millisecond)
		h4.Eval(`() => this.remove()`)
		time.Sleep(30 * time.Millisecond)
		btn.Eval(`() => this.style.visibility = 'hidden'`)
	}()

	h4.Timeout(timeout).WaitInvisible()
	btn.Timeout(timeout).WaitInvisible()

	s.False(p.Has("h4"))
}

func (s *S) TestFnErr() {
	p := s.page.Navigate(s.htmlFile("fixtures/click.html"))
	el := p.Element("button")

	_, err := el.EvalE(true, "foo()")
	s.Error(err)
	s.Contains(err.Error(), "[rod] ReferenceError: foo is not defined")
	s.Nil(errors.Unwrap(err))

	_, err = el.ElementByJSE("foo()")
	s.Error(err)
	s.Contains(err.Error(), "[rod] ReferenceError: foo is not defined")
	s.Nil(errors.Unwrap(err))
}

func (s *S) TestElementOthers() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("form")
	el.Focus()
	el.ScrollIntoViewIfNeeded()
	s.EqualValues(784, el.Box().Get("width").Int())
	s.Equal("<input type=\"submit\" value=\"submit\">", el.Element("[type=submit]").HTML())
	el.Wait(`() => true`)
	s.Equal("form", el.ElementByJS(`() => this`).Describe().Get("localName").String())
	s.Len(el.ElementsByJS(`() => []`), 0)
}
