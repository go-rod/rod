package rod_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/input"
	"github.com/ysmood/rod/lib/launcher"
	"github.com/ysmood/rod/lib/proto"
)

// Open github, search for "git"
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
// you may want to launch the browser like this example.
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
		Headless(false). // run browser on foreground, you can also use env "rod=show"
		Devtools(true).  // open devtools for each new tab
		Launch()

	browser := rod.New().
		Timeout(time.Minute).
		ControlURL(url).
		Trace(true).                 // show trace of each input action
		Slowmotion(2 * time.Second). // each input action will take 2 second
		Connect()

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
	browser := rod.New().Timeout(time.Minute).Connect()
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
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	page := browser.Page("https://duckduckgo.com/")

	// the page will send a request to fetch the suggestions
	wait := page.WaitRequestIdle()
	page.Element("#search_form_input_homepage").Click().Input("test")
	time.Sleep(300 * time.Millisecond) // wait for js debounce
	wait()

	// we must be able to get several suggestion items
	fmt.Println(len(page.Elements(".search__autocomplete .acp")) > 0)

	// Output: true
}

// Useful when you want to customize the element query retry logic
func Example_customize_retry_strategy() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	page := browser.Page("https://github.com")

	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)

	// here we use low-level api ElementE other than Element to have more options,
	// use backoff algorithm to do the retry
	el, err := page.Timeout(10*time.Second).ElementE(backoff, "", "input")
	if err == context.DeadlineExceeded {
		fmt.Println("we can't find the element before timeout")
	} else {
		kit.E(err)
	}

	// get element without retry
	el, err = page.ElementE(nil, "", "input")
	if rod.IsError(err, rod.ErrElementNotFound) {
		fmt.Println("element not found")
	} else {
		kit.E(err)
	}

	fmt.Println(el.Eval(`() => this.name`).String())

	// Output: q
}

// The launcher lib comes with a lot of default switches (flags) to launch browser,
// this example shows how to add or delete switches.
func Example_customize_browser_launch() {
	// set custom browser options
	// use IDE to check the doc of launcher.New you will find out more info
	url := launcher.New().
		Set("proxy-server", "127.0.0.1:8080"). // add a flag, here we set a http proxy
		Delete("use-mock-keychain").           // delete a flag
		Launch()

	browser := rod.New().ControlURL(url).Connect()
	defer browser.Close()

	// auth the proxy
	// here we use cli tool "mitmproxy --proxyauth user:pass" as an example
	browser.HandleAuth("user", "pass")

	// mitmproxy needs cert config to support https, use http here as an example
	fmt.Println(browser.Page("http://example.com/").Element("title").Text())

	// Skip
	// Output: Example Domain
}

// Useful when rod doesn't have the function you want, you can call the cdp interface directly easily.
func Example_direct_cdp() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	// The code here is how SetCookies works
	// Normally, you use something like browser.Page("").SetCookies(...).Navigate(url)

	page := browser.Page("")

	// call cdp interface directly here
	// set the cookie before we visit the website
	// the "proto" lib contains every JSON schema you need to communicate with browser
	res, err := proto.NetworkSetCookie{
		Name:  "rod",
		Value: "test",
		URL:   "https://example.com",
	}.Call(page)
	kit.E(err)

	fmt.Println(res.Success)

	page.Navigate("https://example.com")

	// eval js on the page to get the cookie
	cookie := page.Eval(`() => document.cookie`).String()

	fmt.Println(cookie)

	// Or even more low-level way to use raw json to send request to browser.
	data, _ := json.Marshal(map[string]string{
		"name":  "rod",
		"value": "test",
		"url":   "https://example.com",
	})
	_, _ = browser.Call(page.GetContext(), string(page.SessionID), "Network.SetCookie", data)

	// Output:
	// true
	// rod=test
}

// Shows how to subscribe events.
func Example_handle_events() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	go browser.EachEvent(func(e *proto.TargetTargetCreated) {
		// if it's not a page return
		if e.TargetInfo.Type != proto.TargetTargetInfoTypePage {
			return
		}

		// create a page from the page id
		page02 := browser.PageFromTargetID(e.TargetInfo.TargetID)

		// log "hey" on each newly created page
		page02.Eval(`() => console.log("hey")`)
	})()

	page01 := browser.Page("")

	// print all "console.log" outputs
	go page01.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		page01.ObjectsToJSON(e.Args).Join(" ")
	})()

	// Subscribe events before they happen, run the "wait()" to start consuming the events.
	// Here we return an optional stop signal at the first event to stop the loop.
	wait := page01.EachEvent(func(e *proto.PageLoadEventFired) (stop bool) {
		return true
	})

	page01.Navigate("https://example.com")

	wait()

	// the above is the same as below
	if false {
		page01.WaitEvent(&proto.PageLoadEventFired{})()
	}

	fmt.Println("done")

	// Output:
	// done
}

// Request interception example to modify request or response.
func Example_hijack_requests() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	router := browser.HijackRequests()
	defer router.Stop()

	router.Add("*.js", func(ctx *rod.Hijack) {
		// Send request load response from real destination as the default value to hijack.
		// If you want to safe bandwidth and don't call it, you have to mock the entire response (status code, headers, body).
		ctx.LoadResponse()

		// override response body, we let all js log string "rod"
		ctx.Response.SetBody(ctx.Response.StringBody() + "\n document.title = 'hi' ")
	})

	go router.Run()

	browser.Page("https://www.wikipedia.org/").Wait(`() => document.title == 'hi'`)

	fmt.Println("done")

	// Output: done
}

func Example_states() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	page := browser.Page("")

	// to detect if network is enabled or not
	fmt.Println(page.LoadState(&proto.NetworkEnable{}))

	_ = proto.NetworkEnable{}.Call(page)

	// to detect if network is enabled or not
	fmt.Println(page.LoadState(&proto.NetworkEnable{}))

	// Output:
	// false
	// true
}
