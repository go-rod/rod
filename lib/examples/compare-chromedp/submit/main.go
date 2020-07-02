package main

import (
	"log"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

//This example demonstrates how to fill out and submit a form.
func main() {
	page := rod.New().Connect().Page("https://github.com/search")

	page.Element(`input[name=q]`).WaitVisible().Input("chromedp").Press(input.Enter)

	res := page.ElementMatches("a", "chromedp").Parent().Next().Text()

	log.Printf("got: `%s`", strings.TrimSpace(res))
}
