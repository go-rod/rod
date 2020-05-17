package rod_test

import (
	"fmt"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/input"
	"github.com/ysmood/rod/lib/launcher"
	"github.com/ysmood/rod/lib/proto"
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

// Rod provides a lot of debug options, you can use set methods to enable them or use environment variables
// list at "lib/defaults".
func Example_debug_mode() {
	url := launcher.New().
		Headless(false). // run chrome on foreground, you can also use env "rod=show"
		Devtools(true).  // open devtools for each new tab
		Launch()

	browser := rod.New().
		ControlURL(url).
		Trace(true).             // show trace of each input action
		Slowmotion(time.Second). // each input action will take 1 second
		Connect().
		Timeout(time.Minute)

	// the monitor server that plays the screenshots of each tab, useful when debugging headlee mode
	browser.ServeMonitor(":9777")

	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/")

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

// If a button is moving too fast, you cannot click it as a human, to perfectly simulate human inputs
// the click trigger by Rod are based on mouse point location, so usually you need wait a button is stable before
// you can click it.
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

// Some page interaction finishes after some network requests, WaitRequestIdle is designed for it.
func Example_wait_for_request() {
	browser := rod.New().Connect().Timeout(time.Minute)
	defer browser.Close()

	page := browser.Page("https://duckduckgo.com/")

	// the page will send a request to fetch the suggestions
	wait := page.WaitRequestIdle()
	page.Element("#search_form_input_homepage").Click().Input("test")
	wait()

	// we must get several suggestion items
	fmt.Println(len(page.Elements(".search__autocomplete .acp")) > 0)

	// Output: true
}

// Useful when you want to customize the element query retry logic
func Example_customize_retry_strategy() {
	browser := rod.New().Connect().Timeout(time.Minute)
	defer browser.Close()

	page := browser.Page("https://github.com")

	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)

	// here we use low-level api ElementE other than Element to have more options,
	// use backoff algorithm to do the retry
	el, err := page.ElementE(backoff, "", "input")
	kit.E(err)

	fmt.Println(el.Eval(`() => this.name`).String())

	// Output: q
}

// To enable or disable some special chrome launch flags
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

	// The code here is how SetCookies works
	// Normally, you use something like browser.Page("").SetCookies(...).Navigate(url)

	page := browser.Page("").Timeout(time.Minute)

	// call cdp interface directly here
	// set the cookie before we visit the website
	// Doc: https://chromedevtools.github.io/devtools-protocol/tot/Network#method-setCookie
	res, err := proto.NetworkSetCookie{
		Name:  "rod",
		Value: "test",
		URL:   "https://github.com",
	}.Call(page)
	kit.E(err)

	fmt.Println(res.Success)

	page.Navigate("https://github.com")

	// eval js on the page to get the cookie
	cookie := page.Eval(`() => document.cookie`).String()

	fmt.Println(cookie[:9])

	// Or even more low-level way to use raw json to send request to chrome.
	ctx, client, sessionID := page.CallContext()
	_, _ = client.Call(ctx, sessionID, "Network.SetCookie", map[string]string{
		"name":  "rod",
		"value": "test",
		"url":   "https://github.com",
	})

	// Output:
	// true
	// rod=test;
}

// Shows how to subscribe events.
func Example_handle_events() {
	browser := rod.New().Connect()
	defer browser.Close()

	go browser.EachEvent(func(e *proto.TargetTargetCreated) {
		// if it's not a page return
		if e.TargetInfo.Type != proto.TargetTargetInfoTypePage {
			return
		}

		// create a page from the page id
		page := browser.PageFromTargetID(e.TargetInfo.TargetID)

		// set a global value on each newly created page
		page.Eval(`() => window.hey = "ok"`)
	})

	page := browser.Page("https://github.com")

	// you can also subscribe events only for a page
	// here we return an optional stop signal at the first event to stop the loop
	page.EachEvent(func(e *proto.PageLoadEventFired) bool {
		fmt.Println("loaded")
		return true
	})

	// the above is the same as below
	//
	// e := &proto.PageLoadEventFired{}
	// page.WaitEvent(e)()

	// create a new page and get the value of "hey"
	fmt.Println(page.Eval(`() => hey`).String())

	// Output:
	// loaded
	// ok
}
