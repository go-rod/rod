package rod_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func TestPageElements(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	g.page.MustElement("input")
	list := g.page.MustElements("input")
	g.Eq("input", list.First().MustDescribe().LocalName)
	g.Eq("submit", list.Last().MustText())
}

func TestPagesQuery(t *testing.T) {
	g := setup(t)

	b := g.browser

	b.MustPage(g.srcFile("fixtures/click.html")).MustWaitLoad()
	pages := b.MustPages()

	g.True(pages.MustFind("button").MustHas("button"))
	g.Panic(func() { rod.Pages{}.MustFind("____") })
	g.True(pages.MustFindByURL("click.html").MustHas("button"))
	g.Panic(func() { rod.Pages{}.MustFindByURL("____") })

	_, err := pages.Find("____")
	g.Err(err)
	g.Eq(err.Error(), "cannot find page")
	g.Panic(func() {
		pages.MustFindByURL("____")
	})

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		pages.MustFind("button")
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		pages.MustFindByURL("____")
	})
}

func TestPagesOthers(t *testing.T) {
	g := setup(t)

	list := rod.Pages{}
	g.Nil(list.First())
	g.Nil(list.Last())

	list = append(list, &rod.Page{})

	g.NotNil(list.First())
	g.NotNil(list.Last())
}

func TestPageHas(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	g.page.MustElement("body")
	g.True(g.page.MustHas("span"))
	g.False(g.page.MustHas("a"))
	g.True(g.page.MustHasX("//span"))
	g.False(g.page.MustHasX("//a"))
	g.True(g.page.MustHasR("button", "03"))
	g.False(g.page.MustHasR("button", "11"))

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(g.page.HasX("//a"))

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(g.page.HasR("button", "03"))
}

func TestElementHas(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	b := g.page.MustElement("body")
	g.True(b.MustHas("span"))
	g.False(b.MustHas("a"))
	g.True(b.MustHasX("//span"))
	g.False(b.MustHasX("//a"))
	g.True(b.MustHasR("button", "03"))
	g.False(b.MustHasR("button", "11"))
}

func TestSearch(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))

	el := p.MustSearch("click me")
	g.Eq("click me", el.MustText())
	g.True(el.MustClick().MustMatches("[a=ok]"))

	_, err := p.Sleeper(rod.NotFoundSleeper).Search("not-exists")
	g.True(errors.Is(err, &rod.ErrElementNotFound{}))
	g.Eq(err.Error(), "cannot find element")

	// when search result is not ready
	{
		g.mc.stub(1, proto.DOMGetSearchResults{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(nil), cdp.ErrCtxNotFound
		})
		p.MustSearch("click me")
	}

	// when node id is zero
	{
		g.mc.stub(1, proto.DOMGetSearchResults{}, func(send StubSend) (gson.JSON, error) {
			return gson.New(proto.DOMGetSearchResultsResult{
				NodeIds: []proto.DOMNodeID{0},
			}), nil
		})
		p.MustSearch("click me")
	}

	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMPerformSearch{})
		p.MustSearch("click me")
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.DOMGetSearchResults{})
		p.MustSearch("click me")
	})
	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		p.MustSearch("click me")
	})
}

func TestSearchElements(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))

	{
		res, err := p.Search("button")
		g.E(err)

		c, err := res.All()
		g.E(err)

		g.Len(c, 4)

		g.mc.stubErr(1, proto.DOMGetSearchResults{})
		g.Err(res.All())

		g.mc.stubErr(1, proto.DOMResolveNode{})
		g.Err(res.All())
	}

	{ // disable retry
		sleeper := func() utils.Sleeper { return utils.CountSleeper(1) }
		_, err := p.Sleeper(sleeper).Search("not-exists")
		g.Err(err)
	}
}

func TestSearchIframes(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click-iframes.html"))
	el := p.MustSearch("button[onclick]")
	g.Eq("click me", el.MustText())
	g.True(el.MustClick().MustMatches("[a=ok]"))
}

func TestSearchIframesAfterReload(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click-iframes.html"))
	frame := p.MustElement("iframe").MustFrame().MustElement("iframe").MustFrame()
	frame.MustReload()
	el := p.MustSearch("button[onclick]")
	g.Eq("click me", el.MustText())
	g.True(el.MustClick().MustMatches("[a=ok]"))
}

func TestPageRace(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))

	p.Race().Element("button").MustHandle(func(e *rod.Element) { g.Eq("01", e.MustText()) }).MustDo()
	g.Eq("01", p.Race().Element("button").MustDo().MustText())

	p.Race().ElementX("//button").MustHandle(func(e *rod.Element) { g.Eq("01", e.MustText()) }).MustDo()
	g.Eq("01", p.Race().ElementX("//button").MustDo().MustText())

	p.Race().ElementR("button", "02").MustHandle(func(e *rod.Element) { g.Eq("02", e.MustText()) }).MustDo()
	g.Eq("02", p.Race().ElementR("button", "02").MustDo().MustText())

	p.Race().MustElementByJS("() => document.querySelector('button')", nil).
		MustHandle(func(e *rod.Element) { g.Eq("01", e.MustText()) }).MustDo()
	g.Eq("01", p.Race().MustElementByJS("() => document.querySelector('button')", nil).MustDo().MustText())

	p.Race().Search("button").MustHandle(func(e *rod.Element) { g.Eq("01", e.MustText()) }).MustDo()
	g.Eq("01", p.Race().Search("button").MustDo().MustText())

	raceFunc := func(p *rod.Page) (*rod.Element, error) {
		el := p.MustElement("button")
		g.Eq("01", el.MustText())
		return el, nil
	}
	p.Race().ElementFunc(raceFunc).MustHandle(func(e *rod.Element) { g.Eq("01", e.MustText()) }).MustDo()
	g.Eq("01", p.Race().ElementFunc(raceFunc).MustDo().MustText())

	el, err := p.Sleeper(func() utils.Sleeper { return utils.CountSleeper(2) }).Race().
		Element("not-exists").MustHandle(func(e *rod.Element) {}).
		ElementX("//not-exists").
		ElementR("not-exists", "test").MustHandle(func(e *rod.Element) {}).
		Do()
	g.Err(err)
	g.Nil(el)

	el, err = p.Race().MustElementByJS(`() => notExists()`, nil).Do()
	g.Err(err)
	g.Nil(el)
}

func TestPageRaceRetryInHandle(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	p.Race().Element("div").MustHandle(func(e *rod.Element) {
		go func() {
			utils.Sleep(0.5)
			e.MustElement("button").MustEval(`() => this.innerText = '04'`)
		}()
		e.MustElement("button").MustWait("() => this.innerText === '04'")
	}).MustDo()
}

func TestPageRaceSearchCrossIframe(t *testing.T) {
	g := setup(t)
	g.srcFile("fixtures/selector.html")
	p := g.page.MustNavigate(g.srcFile("fixtures/iframe.html"))

	race := p.Race()
	race.Element("not exist").MustHandle(func(e *rod.Element) { panic("element not exist") })
	race.Search("span").MustHandle(func(e *rod.Element) { g.Eq("01", e.MustText()) })
	race.MustDo()
}

func TestPageElementX(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	g.page.MustElement("body")
	txt := g.page.MustElementX("//div").MustElementX("./button").MustText()
	g.Eq(txt, "02")
}

func TestPageElementsX(t *testing.T) {
	g := setup(t)

	g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	g.page.MustElement("body")
	list := g.page.MustElementsX("//button")
	g.Len(list, 4)
}

func TestElementR(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	el := p.MustElementR("button", `\d1`)
	g.Eq("01", el.MustText())

	el = p.MustElement("div").MustElementR("button", `03`)
	g.Eq("03", el.MustText())

	p = g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el = p.MustElementR("input", `submit`)
	g.Eq("submit", el.MustText())

	el = p.MustElementR("input", `placeholder`)
	g.Eq("blur", *el.MustAttribute("id"))

	el = p.MustElementR("option", `/cc/i`)
	g.Eq("CC", el.MustText())
}

func TestElementFromElement(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElement("button")
	g.Eq("02", el.MustText())
}

func TestElementsFromElement(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("form")
	list := p.MustElement("form").MustElements("option")

	g.Len(list, 4)
	g.Eq("B", list[1].MustText())

	g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
	g.Err(el.Elements("input"))
}

func TestElementParent(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	el := p.MustElement("input").MustParent()
	g.Eq("FORM", el.MustEval(`() => this.tagName`).String())
}

func TestElementParents(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/input.html"))
	g.Len(p.MustElement("option").MustParents("*"), 4)
	g.Len(p.MustElement("option").MustParents("form"), 1)
}

func TestElementSiblings(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	el := p.MustElement("div")
	a := el.MustPrevious()
	b := el.MustNext()

	g.Eq(a.MustText(), "01")
	g.Eq(b.MustText(), "04")
}

func TestElementFromElementX(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	el := p.MustElement("div").MustElementX("./button")
	g.Eq("02", el.MustText())
}

func TestElementsFromElementsX(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html"))
	list := p.MustElement("div").MustElementsX("./button")
	g.Len(list, 2)
}

func TestElementTracing(t *testing.T) {
	g := setup(t)

	g.browser.Trace(true)
	g.browser.Logger(utils.LoggerQuiet)
	defer func() {
		g.browser.Trace(defaults.Trace)
		g.browser.Logger(rod.DefaultLogger)
	}()

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))
	g.Eq(`rod.element("code") html`, p.MustElement("html").MustElement("code").MustText())
}

func TestPageElementByJS(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/click.html"))

	g.Eq(p.MustElementByJS(`() => document.querySelector('button')`).MustText(), "click me")

	_, err := p.ElementByJS(rod.Eval(`() => 1`))
	g.Is(err, &rod.ErrExpectElement{})
	g.Eq(err.Error(), "expect js to return an element, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
}

func TestPageElementsByJS(t *testing.T) {
	g := setup(t)

	p := g.page.MustNavigate(g.srcFile("fixtures/selector.html")).MustWaitLoad()

	g.Len(p.MustElementsByJS("() => document.querySelectorAll('button')"), 4)

	_, err := p.ElementsByJS(rod.Eval(`() => [1]`))
	g.Is(err, &rod.ErrExpectElements{})
	g.Eq(err.Error(), "expect js to return an array of elements, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJS(rod.Eval(`() => 1`))
	g.Eq(err.Error(), "expect js to return an array of elements, but got: {\"type\":\"number\",\"value\":1,\"description\":\"1\"}")
	_, err = p.ElementsByJS(rod.Eval(`() => foo()`))
	g.Err(err)

	g.mc.stubErr(1, proto.RuntimeGetProperties{})
	_, err = p.ElementsByJS(rod.Eval(`() => [document.body]`))
	g.Err(err)

	g.mc.stubErr(4, proto.RuntimeCallFunctionOn{})
	g.Err(p.Elements("button"))
}

func TestPageElementTimeout(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.blank())
	start := time.Now()
	_, err := page.Timeout(300 * time.Millisecond).Element("not-exists")
	g.Is(err, context.DeadlineExceeded)
	g.Gte(time.Since(start), 300*time.Millisecond)
}

func TestPageElementMaxRetry(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.blank())
	s := func() utils.Sleeper { return utils.CountSleeper(5) }
	_, err := page.Sleeper(s).Element("not-exists")
	g.Is(err, &utils.ErrMaxSleepCount{})
}

func TestElementsOthers(t *testing.T) {
	g := setup(t)

	list := rod.Elements{}
	g.Nil(list.First())
	g.Nil(list.Last())
}
