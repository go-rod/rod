package rod_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/color"
	"image/png"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/ysmood/kit"
)

func (s *S) TestClick() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	el := p.Element("button")
	el.Click()

	s.True(p.Has("[a=ok]"))

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.Click()
	})
	s.Panics(func() {
		defer s.errorAt(2, nil)()
		el.Click()
	})
	s.Panics(func() {
		defer s.errorAt(3, nil)()
		el.Click()
	})
	s.Panics(func() {
		defer s.errorAt(4, nil)()
		el.Click()
	})
	s.Panics(func() {
		defer s.errorAt(5, nil)()
		el.Click()
	})
}

func (s *S) TestClickable() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	s.True(p.Element("button").Clickable())
}

func (s *S) TestNotClickable() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	el := p.Element("button")

	// cover the button with a green div
	p.WaitLoad().Eval(`() => {
		let div = document.createElement('div')
		div.style = 'position: absolute; left: 0; top: 0; width: 500px; height: 500px;'
		document.body.append(div)
	}`)
	s.Panics(func() {
		el.Click()
	})

	s.Panics(func() {
		defer s.errorAt(2, nil)()
		el.Clickable()
	})
	s.Panics(func() {
		defer s.errorAt(4, nil)()
		el.Clickable()
	})
	s.Panics(func() {
		defer s.errorAt(8, nil)()
		el.Clickable()
	})
}

func (s *S) TestHover() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	el := p.Element("button")
	el.Eval(`this.onmouseenter = () => this.dataset['a'] = 1`)
	el.Hover()
	s.Equal("1", el.Eval(`this.dataset['a']`).String())
}

func (s *S) TestElementContext() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	el := p.Element("button").Timeout(time.Minute).CancelTimeout()
	s.Error(el.ClickE(proto.InputMouseButtonLeft))
}

func (s *S) TestIframes() {
	p := s.page.Navigate(srcFile("fixtures/click-iframes.html"))
	frame := p.Element("iframe").Frame().Element("iframe").Frame()
	el := frame.Element("button")
	el.Click()
	s.True(frame.Has("[a=ok]"))

	id := el.NodeID()
	s.Panics(func() {
		defer s.errorAt(2, nil)()
		p.ElementFromNode(id)
	})
	s.Panics(func() {
		defer s.at(4, func(d []byte, err error) ([]byte, error) {
			return sjson.SetBytes(d, "result", rod.Array{})
		})()
		p.ElementFromNode(id).Text()
	})
	s.Panics(func() {
		defer s.errorAt(7, nil)()
		p.ElementFromNode(id)
	})
	s.Panics(func() {
		defer s.errorAt(12, nil)()
		p.ElementFromNode(id)
	})
	s.Panics(func() {
		defer s.errorAt(16, nil)()
		p.ElementFromNode(id)
	})
}

func (s *S) TestContains() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	a := p.Element("button")

	b := p.ElementFromNode(a.NodeID())
	s.True(a.ContainsElement(b))

	box := a.Box()
	c := p.ElementFromPoint(int(box.X)+3, int(box.Y)+3)
	s.True(a.ContainsElement(c))

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		a.ContainsElement(b)
	})
}

func (s *S) TestShadowDOM() {
	p := s.page.Navigate(srcFile("fixtures/shadow-dom.html")).WaitLoad()
	el := p.Element("#container")
	s.Equal("inside", el.ShadowRoot().Element("p").Text())

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.ShadowRoot()
	})
	s.Panics(func() {
		defer s.errorAt(2, nil)()
		el.ShadowRoot()
	})
}

func (s *S) TestPress() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("[type=text]")
	el.Press('A')
	el.Press(' ')
	el.Press('b')

	s.Equal("A b", el.Text())

	s.Panics(func() {
		defer s.errorAt(2, nil)()
		el.Press(' ')
	})
	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.SelectAllText()
	})
}

func (s *S) TestKeyDown() {
	p := s.page.Navigate(srcFile("fixtures/keys.html"))
	p.Element("body")
	p.Keyboard.Down('j')

	s.True(p.Has("body[event=key-down-j]"))
}

func (s *S) TestKeyUp() {
	p := s.page.Navigate(srcFile("fixtures/keys.html"))
	p.Element("body")
	p.Keyboard.Up('x')

	s.True(p.Has("body[event=key-up-x]"))
}

func (s *S) TestText() {
	text := "雲の上は\nいつも晴れ"

	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("textarea")
	el.Input(text)

	s.Equal(text, el.Text())
	s.True(p.Has("[event=textarea-change]"))

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.Text()
	})
}

func (s *S) TestCheckbox() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("[type=checkbox]")
	s.True(el.Click().Property("checked").Bool())
}

func (s *S) TestSelectText() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("textarea")
	el.Input("test")
	el.SelectAllText()
	el.Input("test")
	s.Equal("test", el.Text())

	el.SelectText(`es`)
	el.Input("__")

	s.Equal("t__t", el.Text())

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.SelectText("")
	})
	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.SelectAllText()
	})

	s.Panics(func() {
		defer s.errorAt(2, nil)()
		el.Input("")
	})
	s.Panics(func() {
		defer s.errorAt(4, nil)()
		el.Input("")
	})
}

func (s *S) TestBlur() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("#blur").Input("test").Blur()

	s.Equal("ok", *el.Attribute("a"))
}

func (s *S) TestSelectOptions() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("select")
	el.Select("B", "C")

	s.Equal("B,C", el.Text())
	s.EqualValues(1, el.Property("selectedIndex").Int())
}

func (s *S) TestMatches() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("textarea")
	s.True(el.Matches(`[cols="30"]`))

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.Matches("")
	})
}

func (s *S) TestAttribute() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("textarea")
	cols := el.Attribute("cols")
	rows := el.Attribute("rows")

	s.Equal("30", *cols)
	s.Equal("10", *rows)

	p = s.page.Navigate(srcFile("fixtures/click.html"))
	el = p.Element("button").Click()

	s.Equal("ok", *el.Attribute("a"))
	s.Nil(el.Attribute("b"))

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.Attribute("")
	})
}

func (s *S) TestProperty() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("textarea")
	cols := el.Property("cols")
	rows := el.Property("rows")

	s.Equal(float64(30), cols.Num)
	s.Equal(float64(10), rows.Num)

	p = s.page.Navigate(srcFile("fixtures/open-page.html"))
	el = p.Element("a")

	s.Equal("link", el.Property("id").Str)
	s.Equal("_blank", el.Property("target").Str)
	s.Equal(gjson.Null, el.Property("test").Type)

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.Property("")
	})
}

func (s *S) TestSetFiles() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element(`[type=file]`)
	el.SetFiles(
		slash("fixtures/click.html"),
		slash("fixtures/alert.html"),
	)

	list := el.Eval("Array.from(this.files).map(f => f.name)").Array()
	s.Len(list, 2)
	s.Equal("alert.html", list[1].String())
}

func (s *S) TestSelectQuery() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("select")
	el.Select("[value=c]")

	s.EqualValues(2, el.Eval("this.selectedIndex").Int())
}

func (s *S) TestSelectQueryNum() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("select")
	el.Select("123")

	s.EqualValues(-1, el.Eval("this.selectedIndex").Int())
}

func (s *S) TestEnter() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("[type=submit]")
	el.Press(input.Enter)

	s.True(p.Has("[event=submit]"))
}

func (s *S) TestWaitInvisible() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	h4 := p.Element("h4")
	btn := p.Element("button")
	timeout := 3 * time.Second

	s.True(h4.Visible())

	h4t := h4.Timeout(timeout)
	h4t.WaitVisible()
	h4t.CancelTimeout()

	go func() {
		kit.Sleep(0.03)
		h4.Eval(`this.remove()`)
		kit.Sleep(0.03)
		btn.Eval(`this.style.visibility = 'hidden'`)
	}()

	h4.Timeout(timeout).WaitInvisible()
	btn.Timeout(timeout).WaitInvisible()

	s.False(p.Has("h4"))
}

func (s *S) TestWaitStable() {
	p := s.page.Navigate(srcFile("fixtures/wait-stable.html"))
	el := p.Element("button")
	el.WaitStable()
	el.Click()
	p.Has("[event=click]")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		kit.Sleep(0.2)
		cancel()
	}()
	s.Error(el.Context(ctx, cancel).WaitStableE(time.Minute))
}

func (s *S) TestCanvasToImage() {
	p := s.page.Navigate(srcFile("fixtures/canvas.html"))
	src, err := png.Decode(bytes.NewBuffer(p.Element("#canvas").CanvasToImage("", 1.0)))
	kit.E(err)
	s.Equal(src.At(50, 50), color.NRGBA{0xFF, 0x00, 0x00, 0xFF})
}

func (s *S) TestResource() {
	p := s.page.Navigate(srcFile("fixtures/resource.html"))
	el := p.Element("img").WaitLoad()
	s.Equal(15456, len(el.Resource()))

	func() {
		defer s.at(3, func(res []byte, err error) ([]byte, error) {
			return kit.MustToJSONBytes(proto.PageGetResourceContentResult{
				Content:       "ok",
				Base64Encoded: false,
			}), nil
		})()
		s.Equal([]byte("ok"), el.Resource())
	}()

	s.Panics(func() {
		defer s.errorAt(2, nil)()
		el.Resource()
	})
	s.Panics(func() {
		defer s.errorAt(3, nil)()
		el.Resource()
	})
}

func (s *S) TestElementScreenshot() {
	f := filepath.Join("tmp", kit.RandString(8)+".png")
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	el := p.Element("h4")

	data := el.Screenshot(f)
	img, err := png.Decode(bytes.NewBuffer(data))
	kit.E(err)
	s.EqualValues(200, img.Bounds().Dx())
	s.EqualValues(30, img.Bounds().Dy())
	s.FileExists(f)

	s.Panics(func() {
		defer s.errorAt(1, nil)()
		el.Screenshot()
	})
	s.Panics(func() {
		s.countCall()
		defer s.errorAt(2, nil)()
		el.Screenshot()
	})
	s.Panics(func() {
		defer s.errorAt(3, nil)()
		el.Screenshot()
	})
}

func (s *S) TestUseReleasedElement() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	btn := p.Element("button")
	btn.Release()
	s.EqualError(btn.ClickE("left"), "context canceled")

	btn = p.Element("button")
	kit.E(proto.RuntimeReleaseObject{ObjectID: btn.ObjectID}.Call(p))
	s.EqualError(btn.ClickE("left"), "{\"code\":-32000,\"message\":\"Could not find object with given id\",\"data\":\"\"}")
}

func (s *S) TestElementMultipleTimes() {
	// To see whether chrome will reuse the remote object ID or not.
	// Seems like it will not.

	page := s.page.Navigate(srcFile("fixtures/click.html"))

	btn01 := page.Element("button")
	btn02 := page.Element("button")

	s.Equal(btn01.Text(), btn02.Text())
	s.NotEqual(btn01.ObjectID, btn02.ObjectID)
}

func (s *S) TestFnErr() {
	p := s.page.Navigate(srcFile("fixtures/click.html"))
	el := p.Element("button")

	_, err := el.EvalE(true, "foo()", nil)
	s.Error(err)
	s.Contains(err.Error(), "ReferenceError: foo is not defined")
	s.True(errors.Is(err, rod.ErrEval))

	_, err = el.ElementByJSE("foo()", nil)
	s.Error(err)
	s.Contains(err.Error(), "ReferenceError: foo is not defined")
	s.True(errors.Is(err, rod.ErrEval))
}

func (s *S) TestElementEWithDepth() {
	checkStr := `green tea`
	p := s.page.Navigate(srcFile("fixtures/describe.html"))

	ulDOMNode, err := p.Element(`ul`).DescribeE(-1, true)
	s.Nil(errors.Unwrap(err))

	data, err := json.Marshal(ulDOMNode)
	s.Nil(errors.Unwrap(err))
	// The depth is -1, should contain checkStr
	s.Contains(string(data), checkStr)
}

func (s *S) TestElementOthers() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("form")
	s.IsType(p.GetContext(), el.GetContext())
	el.Focus()
	el.ScrollIntoView()
	s.EqualValues(784, el.Box().Width)
	s.Equal("submit", el.Element("[type=submit]").Text())
	s.Equal("<input type=\"submit\" value=\"submit\">", el.Element("[type=submit]").HTML())
	el.Wait(`true`)
	s.Equal("form", el.ElementByJS(`this`).Describe().LocalName)
	s.Len(el.ElementsByJS(`[]`), 0)
}

func (s *S) TestElementErrors() {
	p := s.page.Navigate(srcFile("fixtures/input.html"))
	el := p.Element("form")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := el.Context(ctx, cancel).DescribeE(-1, true)
	s.Error(err)

	err = el.Context(ctx, cancel).FocusE()
	s.Error(err)

	err = el.Context(ctx, cancel).PressE('a')
	s.Error(err)

	err = el.Context(ctx, cancel).InputE("a")
	s.Error(err)

	err = el.Context(ctx, cancel).SelectE([]string{"a"})
	s.Error(err)

	err = el.Context(ctx, cancel).WaitStableE(0)
	s.Error(err)

	_, err = el.Context(ctx, cancel).BoxE()
	s.Error(err)

	_, err = el.Context(ctx, cancel).ResourceE()
	s.Error(err)

	err = el.Context(ctx, cancel).InputE("a")
	s.Error(err)

	err = el.Context(ctx, cancel).InputE("a")
	s.Error(err)

	_, err = el.Context(ctx, cancel).HTMLE()
	s.Error(err)

	_, err = el.Context(ctx, cancel).VisibleE()
	s.Error(err)

	_, err = el.Context(ctx, cancel).CanvasToImageE("", 0)
	s.Error(err)

	err = el.Context(ctx, cancel).ReleaseE()
	s.Error(err)

	s.Panics(func() {
		s.countCall()
		defer s.errorAt(2, nil)()
		el.NodeID()
	})
}
