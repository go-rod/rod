package rod_test

import (
	"fmt"
	"time"

	digto "github.com/ysmood/digto/client"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/input"
	"github.com/ysmood/rod/lib/launcher"
)

func Example_basic() {
	browser := rod.New().Connect()

	// Even you forget to close, rod will close it after main process ends
	defer browser.Close()

	// timeout will be passed to chained function calls
	page := browser.Page("https://www.wikipedia.org/").Timeout(time.Minute)

	page.Element("#searchInput").Input("idempotent")

	page.Element("[type=submit]").Click()

	text := page.Element("#firstHeading").Text()

	fmt.Println(text)

	// Output: Idempotence
}

// Such as you logged in your github account and you want to reuse the login session,
// you may want to launch Chrome like this example.
func Example_reuse_sessions() {
	url := launcher.NewUserMode().Launch()

	browser := rod.New().ControlURL(url).Connect()

	browser.Page("https://github.com")

	fmt.Println("done")

	// Skip
	// Output: done
}

func Example_debug_mode() {
	// run chrome on foreground
	url := launcher.New().Headless(false).Launch()

	browser := rod.New().
		ControlURL(url).
		DebugCDP(true).          // log all cdp traffic
		Trace(true).             // show trace of each input action
		Slowmotion(time.Second). // each input action will take 1 second
		Connect()

	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/").Timeout(time.Minute)

	// enable auto screenshot before each input action
	page.TraceDir("tmp/screenshots")

	page.Element("#searchLanguage").Select("[lang=zh]")
	page.Element("#searchInput").Input("热干面")
	page.Keyboard.Press(input.Enter)

	fmt.Println(page.Element("#firstHeading").Text())

	// get the image binary
	img := page.Element(`[alt="Hot Dry Noodles.jpg"]`)
	_ = kit.OutputFile("tmp/img.jpg", img.Resource(), nil)

	// pause the js execution
	// you can resume by open the devtools and click the resume button on source tab
	page.Pause()

	// Skip
	// Output: 热干面
}

func Example_wait_for_animation() {
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("https://getbootstrap.com/docs/4.0/components/modal/").Timeout(time.Minute)

	page.WaitLoad().Element("[data-target='#exampleModalLive']").Click()

	saveBtn := page.ElementMatches("#exampleModalLive button", "Close")

	// wait until the save button's position is stable
	// and we don't wait more than 5s, saveBtn will also inherit the 1min timeout from the page
	saveBtn.Timeout(5 * time.Second).WaitStable().Click().WaitInvisible()

	fmt.Println("done")

	// Output: done
}

func Example_wait_for_request() {
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("https://www.google.com/").Timeout(time.Minute)

	wait := page.WaitRequestIdle()

	page.WaitLoad().Element(`[name="q"]`).Input("idempotent")

	wait()

	// should be able to find the search suggestion after the ajax request
	fmt.Println(page.HasMatches("div > span", "idempotent"))

	// Output: true
}

func Example_customize_retry_strategy() {
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("https://duckduckgo.com/")

	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)

	// here low-level api ElementE other than Element to have more options,
	// use backoff algorithm to do the retry
	el, err := page.Timeout(time.Minute).ElementE(backoff, "", "#search_form_input_homepage")
	kit.E(err)

	fmt.Println(el.Eval(`() => this.name`))

	// Output: q
}

func Example_customize_chrome_launch() {
	// set custom chrome options
	url := launcher.New().
		Set("disable-sync").         // add flag
		Delete("use-mock-keychain"). // delete flag
		Launch()

	browser := rod.New().ControlURL(url).Connect()
	defer browser.Close()

	el := browser.Page("https://www.wikipedia.org/").Element("title")

	fmt.Println(el.Text())

	// Output: Wikipedia
}

// Useful when rod doesn't have the function you want, you can call the cdp interface directly easily.
func Example_direct_cdp() {
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("about:blank").Timeout(time.Minute)

	// call cdp interface directly here
	// set the cookie before we visit the website
	// Doc: https://chromedevtools.github.io/devtools-protocol/tot/Network#method-setCookie
	page.Call("Network.setCookie", &cdp.Object{
		"name":  "rod",
		"value": "test",
		"url":   "https://www.wikipedia.org",
	})

	page.Navigate("https://www.wikipedia.org/").WaitLoad()

	// eval js on the page to get the cookie
	cookie := page.Eval(`() => document.cookie`).String()

	fmt.Println(cookie[:9])

	// Output: rod=test;
}

// An example to handle 3DS stripe callback.
// It shows how to use Frame method to handle iframes.
func Example_stripe_callback() {
	// use digto to reverse proxy public request to local
	// how it works: https://github.com/ysmood/digto
	dig := digto.New(kit.RandString(8))

	authHeader := []string{"Authorization", "Bearer sk_test_4eC39HqLyjWDarjtT1zdp7dc"}

	cardToken := kit.Req("https://api.stripe.com/v1/tokens").Post().Form(
		"card", map[string]interface{}{
			"number":    "4000000000003220",
			"exp_month": "7",
			"exp_year":  "2025",
			"cvc":       "314",
		},
	).Header(authHeader...).MustJSON().Get("id").String()

	redirectURL := kit.Req("https://api.stripe.com/v1/payment_intents").Post().Form(
		"amount", "2000",
		"currency", "usd",
		"payment_method_data", map[string]interface{}{
			"type": "card",
			"card": map[string]interface{}{
				"token": cardToken,
			},
		},
		"confirm", "true",
		"return_url", dig.PublicURL(),
	).Header(authHeader...).MustJSON().Get("next_action.redirect_to_url.url").String()

	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page(redirectURL)

	frame01 := page.Timeout(3 * time.Minute).Element("[name=__privateStripeFrame4]").Frame()
	frame02 := frame01.Element("#challengeFrame").Frame() // an iframe inside frame01
	frame01.Element(".Spinner").WaitInvisible()           // wait page loading
	frame02.ElementMatches("button", "Complete").Click()

	_, res, err := dig.Next()
	kit.E(err)
	kit.E(res(200, nil, nil))

	fmt.Println("done")

	// Output: done
}
