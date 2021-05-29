package rod_test

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func (t T) BrowserWithPanic() {
	var triggers int
	trigger := func(x interface{}) {
		triggers++
		panic(x)
	}

	browser := t.browser.Sleeper(rod.NotFoundSleeper).WithPanic(trigger)
	t.Panic(func() { browser.MustPage("____") })
	t.Eq(1, triggers)

	page := browser.MustPage(t.blank())
	defer page.MustClose()

	t.Panic(func() { page.MustElement("____") })
	t.Eq(2, triggers)

	el := page.MustElement("html")

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	t.Eq(3, triggers)
}

func (t T) PageWithPanic() {
	var triggers int
	trigger := func(x interface{}) {
		triggers++
		panic(x)
	}

	browser := t.browser.Sleeper(rod.NotFoundSleeper)
	t.Panic(func() { browser.MustPage("____") })
	t.Eq(0, triggers)

	page := browser.MustPage(t.blank()).WithPanic(trigger)
	defer page.MustClose()

	t.Panic(func() { page.MustElement("____") })
	t.Eq(1, triggers)

	el := page.MustElement("html")

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	t.Eq(2, triggers)
}

func (t T) ElementWithPanic() {
	var triggers int
	trigger := func(x interface{}) {
		triggers++
		panic(x)
	}

	browser := t.browser.Sleeper(rod.NotFoundSleeper)
	t.Panic(func() { browser.MustPage("____") })
	t.Eq(0, triggers)

	page := browser.MustPage(t.blank())
	defer page.MustClose()

	t.Panic(func() { page.MustElement("____") })
	t.Eq(0, triggers)

	el := page.MustElement("html").WithPanic(trigger)

	t.Panic(func() {
		t.mc.stubErr(1, proto.RuntimeCallFunctionOn{})
		el.MustClick()
	})
	t.Eq(1, triggers)
}
