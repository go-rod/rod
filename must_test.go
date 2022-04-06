package rod_test

import (
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func TestBrowserWithPanic(t *testing.T) {
	g := setup(t)

	var triggers int
	trigger := func(x interface{}) {
		triggers++
		panic(x)
	}

	browser := g.browser.Sleeper(rod.NotFoundSleeper).WithPanic(trigger)
	g.Panic(func() { browser.MustPage("____") })
	g.Eq(1, triggers)

	page := browser.MustPage(g.blank())
	defer page.MustClose()

	g.Panic(func() { page.MustElement("____") })
	g.Eq(2, triggers)

	el := page.MustElement("html")

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	g.Eq(3, triggers)
}

func TestPageWithPanic(t *testing.T) {
	g := setup(t)

	var triggers int
	trigger := func(x interface{}) {
		triggers++
		panic(x)
	}

	browser := g.browser.Sleeper(rod.NotFoundSleeper)
	g.Panic(func() { browser.MustPage("____") })
	g.Eq(0, triggers)

	page := browser.MustPage(g.blank()).WithPanic(trigger)
	defer page.MustClose()

	g.Panic(func() { page.MustElement("____") })
	g.Eq(1, triggers)

	el := page.MustElement("html")

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	g.Eq(2, triggers)
}

func TestElementWithPanic(t *testing.T) {
	g := setup(t)

	var triggers int
	trigger := func(x interface{}) {
		triggers++
		panic(x)
	}

	browser := g.browser.Sleeper(rod.NotFoundSleeper)
	g.Panic(func() { browser.MustPage("____") })
	g.Eq(0, triggers)

	page := browser.MustPage(g.blank())
	defer page.MustClose()

	g.Panic(func() { page.MustElement("____") })
	g.Eq(0, triggers)

	el := page.MustElement("html").WithPanic(trigger)

	g.Panic(func() {
		g.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	g.Eq(1, triggers)
}
