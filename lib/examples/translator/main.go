package main

import (
	"fmt"
	"os"

	"github.com/ysmood/rod"
)

func main() {
	// get the first commandline argument
	source := os.Args[1]

	browser := rod.New().Connect()

	page := browser.Page("https://www.bing.com/translator")

	wait := page.WaitRequestIdle()
	page.Element("#tta_input_ta").Click().Input(source)
	wait()

	result := page.Element("#tta_output_ta").Text()

	fmt.Println(result)
}
