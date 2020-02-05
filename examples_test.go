package rod_test

import (
	"fmt"
	"time"

	digto "github.com/ysmood/digto/client"
	"github.com/ysmood/kit"
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

// ExampleBrowser is an example to do 3DS stripe payment
func ExampleBrowser() {
	type kv map[string]interface{}

	req := func(url string) *kit.ReqContext {
		return kit.Req(url).Header("Authorization", "Bearer sk_test_4eC39HqLyjWDarjtT1zdp7dc")
	}

	dig := digto.New(kit.RandString(16))

	token := req("https://api.stripe.com/v1/tokens").Post().Form(
		"card", kv{
			"number":    "4000000000003220",
			"exp_month": "7",
			"exp_year":  "2025",
			"cvc":       "314",
		},
	).MustJSON().Get("id").String()

	url := req("https://api.stripe.com/v1/payment_intents").Post().Form(
		"amount", "2000",
		"currency", "usd",
		"payment_method_data", kv{
			"type": "card",
			"card": kv{
				"token": token,
			},
		},
		"confirm", "true",
		"return_url", dig.PublicURL(),
	).MustJSON().Get("next_action.redirect_to_url.url").String()

	browser := rod.Open(nil)
	defer browser.Close()
	browser.Page(url).
		Element("[name=__privateStripeFrame4]").Frame().
		Element("#challengeFrame").Frame().
		Element("#test-source-authorize-3ds").Click()

	_, res, err := dig.Next()
	kit.E(err)
	kit.E(res(200, nil, nil))

	fmt.Sprintln("done")

	// Output: done
}
