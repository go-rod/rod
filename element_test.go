package rod_test

import (
	"errors"
	"time"
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
	el.Press("a")

	s.Equal("a", el.Eval(`() => this.value`).String())
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
	el.Select("[value=c]")

	s.EqualValues(2, el.Eval("() => this.selectedIndex").Int())
}

func (s *S) TestEnter() {
	p := s.page.Navigate(s.htmlFile("fixtures/input.html"))
	el := p.Element("[type=submit]")
	el.Press("Enter")

	s.True(p.Has("[event=submit]"))
}

func (s *S) TestWaitInvisible() {
	p := s.page.Navigate(s.htmlFile("fixtures/click.html"))
	h4 := p.Element("h4")
	btn := p.Element("button")
	timeout := 3 * time.Second

	h4.Timeout(timeout).WaitVisible()

	go func() {
		time.Sleep(100 * time.Millisecond)
		h4.Eval(`() => this.remove()`)
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
}
