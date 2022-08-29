// Package main ...
package main

import "testing"

// test case: 1 + 2 = 3
func TestAdd(t *testing.T) {
	g := setup(t)

	p := g.page("https://ahfarmer.github.io/calculator")

	p.MustElementR("button", "1").MustClick()
	p.MustElementR("button", `^\+$`).MustClick()
	p.MustElementR("button", "2").MustClick()
	p.MustElementR("button", "=").MustClick()

	// assert the result with t.Eq
	g.Eq(p.MustElement(".component-display").MustText(), "3")
}

// test case: 2 * 3 = 6
func TestMultiple(t *testing.T) {
	g := setup(t)

	p := g.page("https://ahfarmer.github.io/calculator")

	// use for-loop to click each button
	for _, regex := range []string{"2", "x", "3", "="} {
		p.MustElementR("button", regex).MustClick()
	}

	g.Eq(p.MustElement(".component-display").MustText(), "6")
}
