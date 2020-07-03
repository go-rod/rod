package main

import (
	"log"
	"time"

	"github.com/go-rod/rod"
)

// This example demonstrates how to use a selector to click on an element.
func main() {
	page := rod.New().
		Connect().
		Trace(true). // log useful info about what rod is doing
		Timeout(15 * time.Second).
		Page("https://golang.org/pkg/time/")

	// wait for footer element is visible (ie, page is loaded)
	page.Element(`body > footer`).WaitVisible()

	// find and click "Expand All" link
	page.Element(`#pkg-examples > div`).Click()

	// retrieve the value of the textarea
	example := page.Element(`#example_After .play .input textarea`).Text()

	log.Printf("Go's time.After example:\n%s", example)
}
