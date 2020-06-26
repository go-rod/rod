package main

import (
	"github.com/ysmood/rod"
	"log"
)

// This example shows how we can use Eval to run scripts in the page.
// Note: `this` in the eval function will refer to the element that Eval is
// called  on. This can be useful for things such as blurring elements.
func main() {
	browser := rod.New().Connect()

	page := browser.Page("https://www.google.com/")
	res := page.Element("#main").Eval("() => Object.keys(window)")

	log.Printf("window object keys: %v", res)
}
