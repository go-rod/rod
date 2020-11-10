package main

import (
	"fmt"
	"os"

	"github.com/go-rod/rod"
)

func main() {
	// get the first commandline argument
	source := os.Args[1]

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://translate.google.com")

	el := page.MustElement(`textarea[aria-label="Source text"]`)

	wait := page.MustWaitRequestIdle("https://accounts.google.com")
	el.MustInput(source)
	wait()

	result := page.MustElement("[role=region] span[lang]").MustText()

	fmt.Println(result)
}
