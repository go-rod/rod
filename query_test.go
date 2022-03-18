package rod_test

import (
	"context"
	"errors"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func (t T) PageElements() {
	t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	t.page.MustElement("input")
	list := t.page.MustElements("input")
	t.Eq("input", list.First().MustDescribe().LocalName)
	t.Eq("submit", list.Last().MustText())
}

func (t T) Pages() {
	t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustWaitLoad()
	pages := t.browser.MustPages()

	t.True(pages.MustFind("button").MustHas("button"))
	t.Panic(func() { rod.Pages{}.MustFind("____") })
	t.True(pages.MustFindByURL("click.html").MustHas("button"))
	t.Panic(func() { rod.Pages{}.MustFindByURL("____") })

	_, err := pages.Find("____")
	t.Err(err)
	t.Eq(err.Error(), "cannot find page")
	t.Panic(func() {
		pages.MustFindByURL("____")
	})

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		pages.MustFind("button")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		pages.MustFindByURL("____")
	})
}

func (t T) PageHas() {
	t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	t.page.MustElement("body")
	t.True(t.page.MustHas("span"))
	t.False(t.page.MustHas("a"))
	t.True(t.page.MustHasX("//span"))
	t.False(t.page.MustHasX("//a"))
	t.True(t.page.MustHasR("button", "03"))
	t.False(t.page.MustHasR("button", "11"))

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	t.Err(t.page.HasX("//a"))

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	t.Err(t.page.HasR("button", "03"))
}

func (t T) ElementHas() {
	t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	b := t.page.MustElement("body")
	t.True(b.MustHas("span"))
	t.False(b.MustHas("a"))
	t.True(b.MustHasX("//span"))
	t.False(b.MustHasX("//a"))
	t.True(b.MustHasR("button", "03"))
	t.False(b.MustHasR("button", "11"))
}

func (t T) Search() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))

	el := p.MustSearch("click me")
	t.Eq("click me", el.MustText())
	t.True(el.MustClick().MustMatches("[a=ok]"))

	_, err := p.Sleeper(rod.NotFoundSleeper).Search("not-exists")
	t.True(errors.Is(err, &rod.ErrElementNotFound{}))
	t.Eq(err.Error(), "cannot find element")

	// when search result is not ready
	{
		t.mc.stub(1, proto.DOMGetSearchResults{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(nil), cdp.ErrCtxNotFound
		})
		p.MustSearch("click me")
	}

	// when node id is zero
	{
		t.mc.stub(1, proto.DOMGetSearchResults{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(proto.DOMGetSearchResultsResult{
				NodeIds: []proto.DOMNodeID{0},
			}), nil
		})
		p.MustSearch("click me")
	}

	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMPerformSearch{})
		p.MustSearch("click me")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.DOMGetSearchResults{})
		p.MustSearch("click me")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		p.MustSearch("click me")
	})
}

func (t T) SearchElements() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))

	{
		res, err := p.Search("button")
		t.E(err)

		c, err := res.All()
		t.E(err)

		t.Len(c, 4)

		t.mc.stubErr(1, proto.DOMGetSearchResults{})
		t.Err(res.All())

		t.mc.stubErr(1, proto.DOMResolveNode{})
		t.Err(res.All())
	}

	{ // disable retry
		sleeper := func() utils.Sleeper { return utils.CountSleeper(1) }
		_, err := p.Sleeper(sleeper).Search("not-exists")
		t.Err(err)
	}
}

func (t T) SearchIframes() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click-iframes.html"))
	el := p.MustSearch("button[onclick]")
	t.Eq("click me", el.MustText())
	t.True(el.MustClick().MustMatches("[a=ok]"))
}

func (t T) SearchIframesAfterReload() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click-iframes.html"))
	frame := p.MustElement("iframe").MustFrame().MustElement("iframe").MustFrame()
	frame.MustReload()
	el := p.MustSearch("button[onclick]")
	t.Eq("click me", el.MustText())
	t.True(el.MustClick().MustMatches("[a=ok]"))
}

func (t T) PageRace() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))

	p.Race().Element("button").MustHandle(func(e *rod.Element) { t.Eq("01", e.MustText()) }).MustDo()
	t.Eq("01", p.Race().Element("button").MustDo().MustText())

	p.Race().ElementX("//button").MustHandle(func(e *rod.Element) { t.Eq("01", e.MustText()) }).MustDo()
	t.Eq("01", p.Race().ElementX("//button").MustDo().MustText())

	p.Race().ElementR("button", "02").MustHandle(func(e *rod.Element) { t.Eq("02", e.MustText()) }).MustDo()
	t.Eq("02", p.Race().ElementR("button", "02").MustDo().MustText())

	p.Race().MustElementByJS("() => document.querySelector('button')", nil).
		MustHandle(func(e *rod.Element) { t.Eq("01", e.MustText()) }).MustDo()
	t.Eq("01", p.Race().MustElementByJS("() => document.querySelector('button')", nil).MustDo().MustText())

	el, err := p.Sleeper(func() utils.Sleeper { return utils.CountSleeper(2) }).Race().
		Element("not-exists").MustHandle(func(e *rod.Element) {}).
		ElementX("//not-exists").
		ElementR("not-exists", "test").MustHandle(func(e *rod.Element) {}).
		Do()
	t.Err(err)
	t.Nil(el)

	el, err = p.Race().MustElementByJS(`() => notExists()`, nil).Do()
	t.Err(err)
	t.Nil(el)
}

func (t T) PageRaceRetryInHandle() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	p.Race().Element("div").MustHandle(func(e *rod.Element) {
		go func() {
			utils.Sleep(0.5)
			e.MustElement("button").MustEval(`() => this.innerText = '04'`)
		}()
		e.MustElement("button").MustWait("() => this.innerText === '04'")
	}).MustDo()
}

func (t T) PageElementX() {
	t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	t.page.MustElement("body")
	name := t.page.MustElementX("//*[contains(text(), 'click')]").MustDescribe().LocalName
	t.Eq("button", name)
}

func (t T) PageElementsX() {
	t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	t.page.MustElement("body")
	list := t.page.MustElementsX("//button")
	t.Len(list, 4)
}

func (t T) ElementR() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	el := p.MustElementR("button", `\d1`)
	t.Eq("01", el.MustText())

	el = p.MustElement("div").MustElementR("button", `03`)
	t.Eq("03", el.MustText())

	p = t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el = p.MustElementR("input", `submit`)
	t.Eq("submit", el.MustText())

	el = p.MustElementR("input", `placeholder`)
	t.Eq("blur", *el.MustAttribute("id"))

	el = p.MustElementR("option", `/cc/i`)
	t.Eq("CC", el.MustText())
}

func (t T) ElementFromElement() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElement("button")
	t.Eq("02", el.MustText())
}

func (t T) ElementsFromElement() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("form")
	list := p.MustElement("form").MustElements("option")

	t.Len(list, 4)
	t.Eq("B", list[1].MustText())

	t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	t.Err(el.Elements("input"))
}

func (t T) ElementParent() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("input").MustParent()
	t.Eq("FORM", el.MustEval(`() => this.tagName`).String())
}

func (t T) ElementParents() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	t.Len(p.MustElement("option").MustParents("*"), 4)
	t.Len(p.MustElement("option").MustParents("form"), 1)
}

func (t T) ElementSiblings() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	el := p.MustElement("div")
	a := el.MustPrevious()
	b := el.MustNext()

	t.Eq(a.MustText(), "01")
	t.Eq(b.MustText(), "04")
}

func (t T) ElementFromElementX() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElementX("./button")
	t.Eq("02", el.MustText())
}

func (t T) ElementsFromElementsX() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	list := p.MustElement("div").MustElementsX("./button")
	t.Len(list, 2)
}

func (t T) ElementTracing() {
	t.browser.Trace(true)
	t.browser.Logger(utils.LoggerQuiet)
	defer func() {
		t.browser.Trace(defaults.Trace)
		t.browser.Logger(rod.DefaultLogger)
	}()

	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	t.Eq(`rod.element("code") html`, p.MustElement("html").MustElement("code").MustText())
}

func (t T) PageElementByJS() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))

	t.Eq(p.MustElementByJS(`() => document.querySelector('button')`).MustText(), "click me")

	_, err := p.ElementByJS(rod.Eval(`() => 1`))
	t.Is(err, &rod.ErrExpectElement{})
	t.Eq(err.Error(), "expect js to return an element, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
}

func (t T) PageElementsByJS() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html")).MustWaitLoad()

	t.Len(p.MustElementsByJS("() => document.querySelectorAll('button')"), 4)

	_, err := p.ElementsByJS(rod.Eval(`() => [1]`))
	t.Is(err, &rod.ErrExpectElements{})
	t.Eq(err.Error(), "expect js to return an array of elements, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJS(rod.Eval(`() => 1`))
	t.Eq(err.Error(), "expect js to return an array of elements, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJS(rod.Eval(`() => foo()`))
	t.Err(err)

	t.mc.stubErr(1, proto.RuntimeGetProperties{})
	_, err = p.ElementsByJS(rod.Eval(`() => [document.body]`))
	t.Err(err)

	t.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
	t.Err(p.Elements("button"))
}

func (t T) PageElementTimeout() {
	page := t.page.MustNavigate(t.blank())
	start := time.Now()
	_, err := page.Timeout(300 * time.Millisecond).Element("not-exists")
	t.Is(err, context.DeadlineExceeded)
	t.Gte(time.Since(start), 300*time.Millisecond)
}

func (t T) PageElementMaxRetry() {
	page := t.page.MustNavigate(t.blank())
	s := func() utils.Sleeper { return utils.CountSleeper(5) }
	_, err := page.Sleeper(s).Element("not-exists")
	t.Is(err, &utils.ErrMaxSleepCount{})
}

func (t T) ElementsOthers() {
	list := rod.Elements{}
	t.Nil(list.First())
	t.Nil(list.Last())
}

func (t T) PagesOthers() {
	list := rod.Pages{}
	t.Nil(list.First())
	t.Nil(list.Last())

	list = append(list, &rod.Page{})

	t.NotNil(list.First())
	t.NotNil(list.Last())
}
