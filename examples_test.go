package rod_test

import (
	"fmt"
	"time"

	"github.com/ysmood/rod"
)

func ExampleOpen() {
	browser := rod.Open(&rod.Browser{
		Foreground: true,
		Trace:      true,
		Slowmotion: time.Second,
	})
	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/")

	page.Element("#searchInput").Input("idempotent")

	page.Element("[type=submit]").Click()

	fmt.Println(page.Element("#firstHeading").Text())

	//// Output: Idempotence
}
