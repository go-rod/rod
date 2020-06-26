package main

import (
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/input"
	"log"
	"strings"
)

//This example demonstrates how to fill out and submit a form.
func main() {
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("https://github.com/search")

	page.ElementX(`//input[@name="q"]`).WaitVisible().Input("chromedp").Press(input.Enter)

	elems := page.ElementsX(`//*[@id="js-pjax-container"]//h2[contains(., 'Search more than')]`)
	if !elems.Empty() {
		elems[0].WaitInvisible()
	}

	res := page.ElementX(`(//*[@id="js-pjax-container"]//ul[contains(@class, "repo-list")]/li[1]//p)[1]`).Text()

	log.Printf("got: `%s`", strings.TrimSpace(res))
}
