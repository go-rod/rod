package main

import (
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/launcher"
	"log"
	"time"
)

// This example demonstrates how to use a selector to click on an element.
func main() {
	// Using manual launcher to enable console debugging.
	// You can also use the environment variable rod=cdp with rod.New() directly
	url := launcher.New().Launch()
	client := cdp.New(url).Debug(true)

	browser := rod.New().Timeout(15 * time.Second).Client(client).Connect()
	defer browser.Close()

	page := browser.Page("https://golang.org/pkg/time/")
	// Element will wait till an element with the selector is found.
	page.Element(`body > footer`)
	// Click will expand the dropdown menu for the example.
	page.Element(`#pkg-examples > div`).Click()
	// Text will extract the example's content.
	example := page.Element(`#example_After .play .input textarea`).Text()

	log.Printf("Go's time.After example:\n%s", example)
}
