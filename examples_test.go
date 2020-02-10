package rod_test

import (
	"fmt"
	"time"

	digto "github.com/ysmood/digto/client"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/input"
)

func Example_basic() {
	browser := rod.Open(nil)
	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/").Timeout(time.Minute)

	page.Element("#searchInput").Input("idempotent")

	page.Element("[type=submit]").Click()

	fmt.Println(page.Element("#firstHeading").Text())

	// Output: Idempotence
}

func Example_debug_mode() {
	browser := rod.Open(&rod.Browser{
		Foreground: true,
		Trace:      true,
		Slowmotion: 2 * time.Second,
	})
	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/").Timeout(time.Minute)

	page.Element("#searchLanguage").Select("[lang=zh]")
	page.Element("#searchInput").Input("热干面")
	page.Keyboard.Press(input.Enter)

	fmt.Println(page.Element("#firstHeading").Text())

	// pause the js execution
	// you can resume by open the devtools and click the resume button on source tab
	page.Pause()

	// Skip
	// Output: 热干面
}

// An example to handle 3DS stripe callback
func Example_stripe_callback() {
	req := func(url string) *kit.ReqContext {
		return kit.Req(url).Header("Authorization", "Bearer sk_test_4eC39HqLyjWDarjtT1zdp7dc")
	}

	dig := digto.New(kit.RandString(8))

	token := req("https://api.stripe.com/v1/tokens").Post().Form(
		"card", map[string]interface{}{
			"number":    "4000000000003220",
			"exp_month": "7",
			"exp_year":  "2025",
			"cvc":       "314",
		},
	).MustJSON().Get("id").String()

	url := req("https://api.stripe.com/v1/payment_intents").Post().Form(
		"amount", "2000",
		"currency", "usd",
		"payment_method_data", map[string]interface{}{
			"type": "card",
			"card": map[string]interface{}{
				"token": token,
			},
		},
		"confirm", "true",
		"return_url", dig.PublicURL(),
	).MustJSON().Get("next_action.redirect_to_url.url").String()

	browser := rod.Open(nil)
	defer browser.Close()
	page := browser.Page(url)
	frame01 := page.Timeout(time.Minute).Element("[name=__privateStripeFrame4]").Frame()
	frame02 := frame01.Element("#challengeFrame").Frame() // an iframe inside frame01

	frame01.Element(".Spinner").WaitInvisible() // wait page loading to be done
	frame02.ElementMatches("button", "Complete").Click()

	_, res, err := dig.Next()
	kit.E(err)
	kit.E(res(200, nil, nil))

	fmt.Println("done")

	// Output: done
}
