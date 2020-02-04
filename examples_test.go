package rod_test

import (
	"fmt"
	"time"

	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/input"
)

func ExampleOpen() {
	browser := rod.Open(nil)
	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/")

	page.Element("#searchInput").Input("idempotent")

	page.Element("[type=submit]").Click()

	fmt.Println(page.Element("#firstHeading").Text())

	//// Output: Idempotence
}

func ExampleElement() {
	browser := rod.Open(&rod.Browser{
		Foreground: true,
		Trace:      true,
		Slowmotion: time.Second,
	})
	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/")

	page.Element("#searchLanguage").Select("[lang=zh]")
	page.Element("#searchInput").Input("热干面")
	page.Keyboard.Press(input.Enter)

	fmt.Println(page.Element("#firstHeading").Text())

	//// Output: 热干面
}
