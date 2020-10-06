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

func (c C) PageElements() {
	c.page.MustNavigate(srcFile("fixtures/input.html"))
	c.page.MustElement("input")
	list := c.page.MustElements("input")
	c.Eq("input", list.First().MustDescribe().LocalName)
	c.Eq("submit", list.Last().MustText())
}

func (c C) Pages() {
	c.page.MustNavigate(srcFile("fixtures/click.html")).MustWaitLoad()

	c.True(c.browser.MustPages().MustFind("button").MustHas("button"))
	c.True(c.browser.MustPages().MustFindByURL("click.html").MustHas("button"))

	c.Nil(c.browser.MustPages().Find("____"))
	c.Nil(c.browser.MustPages().MustFindByURL("____"))

	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		c.browser.MustPages().MustFind("button")
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		c.browser.MustPages().MustFindByURL("____")
	})
}

func (c C) PageHas() {
	c.page.MustNavigate(srcFile("fixtures/selector.html"))
	c.page.MustElement("body")
	c.True(c.page.MustHas("span"))
	c.False(c.page.MustHas("a"))
	c.True(c.page.MustHasX("//span"))
	c.False(c.page.MustHasX("//a"))
	c.True(c.page.MustHasR("button", "03"))
	c.False(c.page.MustHasR("button", "11"))
}

func (c C) ElementHas() {
	c.page.MustNavigate(srcFile("fixtures/selector.html"))
	b := c.page.MustElement("body")
	c.True(b.MustHas("span"))
	c.False(b.MustHas("a"))
	c.True(b.MustHasX("//span"))
	c.False(b.MustHasX("//a"))
	c.True(b.MustHasR("button", "03"))
	c.False(b.MustHasR("button", "11"))
}

func (c C) Search() {
	wait := c.page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	wait()

	el := p.MustSearch("click me")
	c.Eq("click me", el.MustText())
	c.True(el.MustClick().MustMatches("[a=ok]"))

	_, err := p.Sleeper(nil).Search(0, 1, "not-exists")
	c.True(errors.Is(err, rod.ErrElementNotFound))

	// when search result is not ready
	{
		c.mc.stub(1, proto.DOMGetSearchResults{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(nil), &cdp.Error{Code: -32000}
		})
		p.MustSearch("click me")
	}

	// when node id is zero
	{
		c.mc.stub(1, proto.DOMGetSearchResults{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(proto.DOMGetSearchResultsResult{
				NodeIds: []proto.DOMNodeID{0},
			}), nil
		})
		p.MustSearch("click me")
	}

	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMPerformSearch{})
		p.MustSearch("click me")
	})
	c.Panic(func() {
		c.mc.stubErr(1, proto.DOMGetSearchResults{})
		p.MustSearch("click me")
	})
	c.Panic(func() {
		c.mc.stubErr(2, proto.RuntimeCallFunctionOn{})
		p.MustSearch("click me")
	})
}

func (c C) SearchIframes() {
	p := c.page.MustNavigate(srcFile("fixtures/click-iframes.html"))
	el := p.MustSearch("button[onclick]")
	c.Eq("click me", el.MustText())
	c.True(el.MustClick().MustMatches("[a=ok]"))
}

func (c C) SearchIframesAfterReload() {
	p := c.page.MustNavigate(srcFile("fixtures/click-iframes.html"))
	frame := p.MustElement("iframe").MustFrame().MustElement("iframe").MustFrame()
	frame.MustReload().MustWaitLoad()
	el := p.MustSearch("button[onclick]")
	c.Eq("click me", el.MustText())
	c.True(el.MustClick().MustMatches("[a=ok]"))
}

func (c C) PageElementWithSelectors() {
	c.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := c.page.MustElement("p", "button")
	c.Eq("01", el.MustText())
}

func (c C) PageRace() {
	p := c.page.MustNavigate(srcFile("fixtures/selector.html"))

	p.Race().MustElement("button", func(el *rod.Element) {
		c.Eq("01", el.MustText())
	}).MustDo()

	p.Race().MustElementX("//button", func(el *rod.Element) {
		c.Eq("01", el.MustText())
	}).MustDo()

	p.Race().MustElementR("button", "02", func(el *rod.Element) {
		c.Eq("02", el.MustText())
	}).MustDo()

	err := p.Sleeper(func() utils.Sleeper { return utils.CountSleeper(2) }).Race().
		MustElement("not-exists", func(el *rod.Element) {}).
		MustElementX("//not-exists", func(el *rod.Element) {}).
		MustElementR("not-exists", "test", func(el *rod.Element) {}).
		Do()
	c.Err(err)

	err = p.Race().MustElementByJS(`notExists()`, nil, nil).Do()
	c.Err(err)
}

func (c C) PageElementX() {
	c.page.MustNavigate(srcFile("fixtures/click.html"))
	c.page.MustElement("body")
	name := c.page.MustElementX("//*[contains(text(), 'click')]").MustDescribe().LocalName
	c.Eq("button", name)
}

func (c C) PageElementsX() {
	c.page.MustNavigate(srcFile("fixtures/input.html"))
	c.page.MustElement("body")
	list := c.page.MustElementsX("//input")
	c.Len(list, 5)
}

func (c C) ElementR() {
	p := c.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := p.MustElementR("button", `\d1`)
	c.Eq("01", el.MustText())

	el = p.MustElement("div").MustElementR("button", `03`)
	c.Eq("03", el.MustText())

	p = c.page.MustNavigate(srcFile("fixtures/input.html"))
	el = p.MustElementR("input", `submit`)
	c.Eq("submit", el.MustText())
}

func (c C) ElementFromElement() {
	p := c.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElement("button")
	c.Eq("02", el.MustText())
}

func (c C) ElementsFromElement() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	p.MustElement("form")
	list := p.MustElement("form").MustElements("option")

	c.Len(list, 4)
	c.Eq("B", list[1].MustText())
}

func (c C) ElementParent() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("input").MustParent()
	c.Eq("FORM", el.MustEval(`this.tagName`).String())
}

func (c C) ElementParents() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	c.Len(p.MustElement("option").MustParents("*"), 4)
	c.Len(p.MustElement("option").MustParents("form"), 1)
}

func (c C) ElementSiblings() {
	p := c.page.MustNavigate(srcFile("fixtures/input.html"))
	el := p.MustElement("hr")
	a := el.MustPrevious()
	b := el.MustNext()

	c.Eq("INPUT", a.MustEval(`this.tagName`).String())
	c.Eq("SELECT", b.MustEval(`this.tagName`).String())
}

func (c C) ElementFromElementX() {
	p := c.page.MustNavigate(srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElementX("./button")
	c.Eq("02", el.MustText())
}

func (c C) ElementsFromElementsX() {
	p := c.page.MustNavigate(srcFile("fixtures/selector.html"))
	list := p.MustElement("div").MustElementsX("./button")
	c.Len(list, 2)
}

func (c C) ElementTracing() {
	c.browser.Trace(true)
	c.browser.Logger(utils.LoggerQuiet)
	defer func() {
		c.browser.Trace(defaults.Trace)
		c.browser.Logger(rod.DefaultLogger)
	}()

	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	c.Eq(`rod.element("code")`, p.MustElement("code").MustText())
}

func (c C) PageElementByJS_Err() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html"))
	_, err := p.ElementByJS(rod.NewEval(`1`))
	c.Eq(err.Error(), `{"type":"number","value":1,"description":"1"}: expect js to return an element`)
}

func (c C) PageElementsByJS_Err() {
	p := c.page.MustNavigate(srcFile("fixtures/click.html")).MustWaitLoad()
	_, err := p.ElementsByJS(rod.NewEval(`[1]`))
	c.Eq(err.Error(), `{"type":"number","value":1,"description":"1"}: expect js to return an array of elements`)
	_, err = p.ElementsByJS(rod.NewEval(`1`))
	c.Eq(err.Error(), `{"type":"number","value":1,"description":"1"}: expect js to return an array of elements`)
	_, err = p.ElementsByJS(rod.NewEval(`foo()`))
	c.Err(err)

	c.mc.stubErr(1, proto.RuntimeGetProperties{})
	_, err = p.ElementsByJS(rod.NewEval(`[document.body]`))
	c.Err(err)
}

func (c C) ElementsOthers() {
	list := rod.Elements{}
	c.Nil(list.First())
	c.Nil(list.Last())
}

func (c C) PagesOthers() {
	list := rod.Pages{}
	c.Nil(list.First())
	c.Nil(list.Last())

	list = append(list, &rod.Page{})

	c.NotNil(list.First())
	c.NotNil(list.Last())
}
