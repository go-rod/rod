package rod_test

import (
	"errors"

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

	t.True(t.browser.MustPages().MustFind("button").MustHas("button"))
	t.True(t.browser.MustPages().MustFindByURL("click.html").MustHas("button"))

	t.Nil(t.browser.MustPages().Find("____"))
	t.Nil(t.browser.MustPages().MustFindByURL("____"))

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		t.browser.MustPages().MustFind("button")
	})
	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		t.browser.MustPages().MustFindByURL("____")
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

	_, err := p.Sleeper(nil).Search(0, 1, "not-exists")
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

func (t T) PageElementWithSelectors() {
	t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	el := t.page.MustElement("p", "button")
	t.Eq("01", el.MustText())
}

func (t T) PageRace() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))

	p.Race().MustElement("button", func(el *rod.Element) {
		t.Eq("01", el.MustText())
	}).MustDo()

	p.Race().MustElementX("//button", func(el *rod.Element) {
		t.Eq("01", el.MustText())
	}).MustDo()

	p.Race().MustElementR("button", "02", func(el *rod.Element) {
		t.Eq("02", el.MustText())
	}).MustDo()

	err := p.Sleeper(func() utils.Sleeper { return utils.CountSleeper(2) }).Race().
		MustElement("not-exists", func(el *rod.Element) {}).
		MustElementX("//not-exists", func(el *rod.Element) {}).
		MustElementR("not-exists", "test", func(el *rod.Element) {}).
		Do()
	t.Err(err)

	err = p.Race().MustElementByJS(`notExists()`, nil, nil).Do()
	t.Err(err)
}

func (t T) PageElementX() {
	t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	t.page.MustElement("body")
	name := t.page.MustElementX("//*[contains(text(), 'click')]").MustDescribe().LocalName
	t.Eq("button", name)
}

func (t T) PageElementsX() {
	t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	t.page.MustElement("body")
	list := t.page.MustElementsX("//input")
	t.Len(list, 5)
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
}

func (t T) ElementFromElement() {
	p := t.page.MustNavigate(t.srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElement("button")
	t.Eq("02", el.MustText())
}

func (t T) ElementsFromElement() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	p.MustElement("form")
	list := p.MustElement("form").MustElements("option")

	t.Len(list, 4)
	t.Eq("B", list[1].MustText())
}

func (t T) ElementParent() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("input").MustParent()
	t.Eq("FORM", el.MustEval(`this.tagName`).String())
}

func (t T) ElementParents() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	t.Len(p.MustElement("option").MustParents("*"), 4)
	t.Len(p.MustElement("option").MustParents("form"), 1)
}

func (t T) ElementSiblings() {
	p := t.page.MustNavigate(t.srcFile("fixtures/input.html"))
	el := p.MustElement("hr")
	a := el.MustPrevious()
	b := el.MustNext()

	t.Eq("INPUT", a.MustEval(`this.tagName`).String())
	t.Eq("SELECT", b.MustEval(`this.tagName`).String())
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
	t.Eq(`rod.element("code")`, p.MustElement("code").MustText())
}

func (t T) PageElementByJSErr() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html"))
	_, err := p.ElementByJS(rod.Eval(`1`))
	t.Is(err, &rod.ErrExpectElement{})
	t.Eq(err.Error(), "expect js to return an element, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
}

func (t T) PageElementsByJSErr() {
	p := t.page.MustNavigate(t.srcFile("fixtures/click.html")).MustWaitLoad()
	_, err := p.ElementsByJS(rod.Eval(`[1]`))
	t.Is(err, &rod.ErrExpectElements{})
	t.Eq(err.Error(), "expect js to return an array of elements, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJS(rod.Eval(`1`))
	t.Eq(err.Error(), "expect js to return an array of elements, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJS(rod.Eval(`foo()`))
	t.Err(err)

	t.mc.stubErr(1, proto.RuntimeGetProperties{})
	_, err = p.ElementsByJS(rod.Eval(`[document.body]`))
	t.Err(err)
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
