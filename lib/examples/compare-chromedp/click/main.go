// Package main ...
package main

import (
	"log"
	"time"

	"github.com/go-rod/rod"
)

// This example demonstrates how to use a selector to click on an element.
func main() {
	page := rod.New().
		MustConnect().
		Trace(true). // log useful info about what rod is doing
		Timeout(15 * time.Second).
		MustPage("https://pkg.go.dev/time/")

	// wait for footer element is visible (ie, page is loaded)
	page.MustElement(`body > footer`).MustWaitVisible()

	// find and click "Expand All" link
	page.MustElement(`#pkg-examples`).MustClick()

	// retrieve the value of the textarea
	example := page.MustElement(`#example-After textarea`).MustText()

	log.Printf("Go's time.After example:\n%s", example)
}
