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

	page := browser.MustPage("https://www.bing.com/translator")

	wait := page.MustWaitRequestIdle()
	page.MustElement("#tta_input_ta").MustClick().MustInput(source)
	wait()

	result := page.MustElement("#tta_output_ta").MustText()

	fmt.Println(result)
}
