package rod_test

import (
	"fmt"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/input"
	"github.com/ysmood/rod/lib/launcher"
)

// Open wikipedia, search for "idempotent", and print the title of result page
func Example_basic() {
	// launch and connect to a browser
	browser := rod.New().Connect()

	// Even you forget to close, rod will close it after main process ends
	defer browser.Close()

	// timeout will be passed to chained function calls
	page := browser.Timeout(time.Minute).Page("https://github.com")

	// make sure windows size is consistent
	page.Window(0, 0, 1200, 600)

	// use css selector to get the search input element and input "git"
	page.Element("input").Input("git").Press(input.Enter)

	// wait until css selector get the element then get the text content of it
	text := page.Element(".codesearch-results p").Text()

	fmt.Println(text)

	// Output: Git is the most widely used version control system.
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
	url := launcher.New().
		Headless(false). // run chrome on foreground
		Devtools(true).  // open devtools for each new tab
		Launch()

	browser := rod.New().
		ControlURL(url).
		DebugCDP(true).          // log all cdp traffic
		Trace(true).             // show trace of each input action
		Slowmotion(time.Second). // each input action will take 1 second
		Connect().
		Timeout(time.Minute)

	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/")

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
	browser := rod.New().Connect().Timeout(time.Minute)
	defer browser.Close()

	page := browser.Page("https://getbootstrap.com/docs/4.0/components/modal/")

	page.WaitLoad().Element("[data-target='#exampleModalLive']").Click()

	saveBtn := page.ElementMatches("#exampleModalLive button", "Close")

	// wait until the save button's position is stable
	// and we don't wait more than 5s, saveBtn will also inherit the 1min timeout from the page
	saveBtn.Timeout(5 * time.Second).WaitStable().Click().WaitInvisible()

	fmt.Println("done")

	// Output: done
}

func Example_wait_for_request() {
	browser := rod.New().Connect().Timeout(time.Minute)
	defer browser.Close()

	page := browser.Page("https://www.bing.com/")

	wait := page.WaitRequestIdle()
	page.Element("#sb_form_q").Click().Input("test")
	wait()

	fmt.Println(page.Has("#sw_as li"))

	// Output: true
}

func Example_customize_retry_strategy() {
	browser := rod.New().Connect().Timeout(time.Minute)
	defer browser.Close()

	page := browser.Page("https://github.com")

	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)

	// here we use low-level api ElementE other than Element to have more options,
	// use backoff algorithm to do the retry
	el, err := page.ElementE(backoff, "", "input")
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

	browser := rod.New().ControlURL(url).Connect().Timeout(time.Minute)
	defer browser.Close()

	el := browser.Page("https://github.com").Element("title")

	fmt.Println(el.Text())

	// Output: The world’s leading software development platform · GitHub
}

// Useful when rod doesn't have the function you want, you can call the cdp interface directly easily.
func Example_direct_cdp() {
	browser := rod.New().Connect()
	defer browser.Close()

	// The code here is how Page.SetCookies works
	// Normally, you use something like browser.Page("").SetCookies(...).Navigate(url)

	page := browser.Page("").Timeout(time.Minute)

	// call cdp interface directly here
	// set the cookie before we visit the website
	// Doc: https://chromedevtools.github.io/devtools-protocol/tot/Network#method-setCookie
	page.Call("Network.setCookie", cdp.Object{
		"name":  "rod",
		"value": "test",
		"url":   "https://github.com",
	})

	page.Navigate("https://github.com")

	// eval js on the page to get the cookie
	cookie := page.Eval(`() => document.cookie`).String()

	fmt.Println(cookie[:9])

	// Output: rod=test;
}
