package rod_test

import (
	"fmt"

	"github.com/ysmood/rod"
)

func ExampleOpen() {
	browser := rod.Open(nil)
	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/")

	page.Element("#searchInput").Input("idempotent")

	page.Element("[type=submit]").Click()

	fmt.Println(page.Element("#firstHeading").Text())
}
