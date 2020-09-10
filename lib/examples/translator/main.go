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

	wait := page.MustWaitRequestIdle()
	page.MustElement("#source").MustInput(source)
	wait()

	result := page.MustElement(".tlid-translation").MustText()

	fmt.Println(result)
}
