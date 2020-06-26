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

// Example_basic is a simple test that opens https://github.com/, searches for
// "git", and then gets the header element which gives the description for Git.
func Example_basic() {
	// Launch a new browser with default options, and connect to it.
	browser := rod.New().Connect()

	// Even you forget to close, rod will close it after main process ends.
	defer browser.Close()

	// Timeout will be passed to all chained function calls.
	// The code will panic out if any chained call is used after the timeout.
	page := browser.Timeout(time.Minute).Page("https://github.com")

	// Resize the window make sure window size is always consistent.
	page.Window(0, 0, 1200, 600)

	// We use css selector to get the search input element and input "git"
	page.Element("input").Input("git").Press(input.Enter)

	// Wait until css selector get the element then get the text content of it.
	text := page.Element(".codesearch-results p").Text()

	fmt.Println(text)

	// Output: Git is the most widely used version control system.
}

// Example_reuse_sessions allows you to use the same session to reduce
// boilerplate code between multiple actions. An example is when you log into
// your github account, and you want to reuse the login session in a different
// task. You can use this method to achieve such functionality.
func Example_reuse_sessions() {
	// Launches a new browser with the "new user mode" option, and returns
	// the URL to control that browser. You can reuse the browser url in
	// multiple places instead of creating a new browser for each task.
	url := launcher.NewUserMode().Launch()

	// Using the ControlURL function, we control our single browser instance
	// instead of creating a new one.
	browser := rod.New().ControlURL(url).Connect()

	browser.Page("https://github.com")

	fmt.Println("done")

	// Skip
	// Output: done
}

// Example_headless_with_debug shows how we can start a browser with debug
// information and headless mode disabled to show the browser in the foreground.
// Rod provides a lot of debug options, you can use the Set method to enable
// them or use environment variables. (Default environment variables can be
// found in "lib/defaults").
func Example_headless_with_debug() {
	// Headless runs the browser on foreground, you can also use env "rod=show"
	// Devtools opens the tab in each new tab opened automatically
	url := launcher.New().
		Headless(false).
		Devtools(true).
		Launch()

	// Trace shows verbose debug information for each action executed
	// Slowmotion is a debug related function that waits 2 seconds between
	// each action, making it easier to inspect what your code is doing.
	browser := rod.New().
		Timeout(time.Minute).
		ControlURL(url).
		Trace(true).
		Slowmotion(2 * time.Second).
		Connect()

	// ServeMonitor plays screenshots of each tab. This feature is extremely
	// useful when debugging with headless mode. Run this example and visit
	// http://localhost:9777 to see how it works.
	browser.ServeMonitor(":9777")

	defer browser.Close()

	page := browser.Page("https://www.wikipedia.org/")

	page.Element("#searchLanguage").Select("[lang=zh]")
	page.Element("#searchInput").Input("热干面")
	page.Keyboard.Press(input.Enter)

	fmt.Println(page.Element("#firstHeading").Text())

	// Response gets the binary of the image as a []byte.
	// We use OutputFile to write the content of the image into ./tmp/img.jpg
	img := page.Element(`[alt="Hot Dry Noodles.jpg"]`)
	_ = kit.OutputFile("tmp/img.jpg", img.Resource(), nil)

	// Pause temporarily halts JavaScript execution on the website.
	// You can resume execution in the devtools window by clicking the resume
	// button in the "source" tab.
	page.Pause()

	// Skip
	// Output: 热干面
}

// Example_wait_for_animation is an example to simulate humans more accurately.
// If a button is moving too fast, you cannot click it as a human. To more
// accurately simulate human inputs, actions triggered by Rod can be based on
// mouse point location, so you can allow Rod to wait for the button to become
// stable before it tries clicking it.
func Example_wait_for_animation() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	page := browser.Page("https://getbootstrap.com/docs/4.0/components/modal/")

	page.WaitLoad().Element("[data-target='#exampleModalLive']").Click()

	saveBtn := page.ElementMatches("#exampleModalLive button", "Close")

	// Here, WaitStable will wait until the save button's position becomes
	// stable. The timeout is 5 seconds, after which it times out (or after 1
	// minute since the browser was created). Timeouts from parents are
	// inherited to the children as well.
	saveBtn.Timeout(5 * time.Second).WaitStable().Click().WaitInvisible()

	fmt.Println("done")

	// Output: done
}

// Example_wait_for_request shows an example where Rod will wait for all
// requests on the page to complete (such as network request) before interacting
// with the page.
func Example_wait_for_request() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	page := browser.Page("https://duckduckgo.com/")

	// WaitRequestIdle will wait for all possible ajax calls to complete before
	// continuing on with further execution calls.
	wait := page.WaitRequestIdle()
	page.Element("#search_form_input_homepage").Click().Input("test")
	time.Sleep(300 * time.Millisecond) // Wait for js debounce.
	wait()

	// We want to make sure that after waiting, there are some autocomplete
	// suggestions available.
	fmt.Println(len(page.Elements(".search__autocomplete .acp")) > 0)

	// Output: true
}

// Example_customize_retry_strategy allows us to change the retry/polling
// options that is used to query elements. This is useful when you want to
// customize the element query retry logic.
func Example_customize_retry_strategy() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	page := browser.Page("https://github.com")

	backoff := kit.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)

	// ElementE is used in this context instead of Element. When The XxxxxE
	// version of functions are used, there will be more access to customise
	// options, like give access to the backoff algorithm.
	el, err := page.Timeout(10*time.Second).ElementE(backoff, "", "input")
	if err == context.DeadlineExceeded {
		fmt.Println("unable to find the input element before timeout")
	} else {
		kit.E(err)
	}

	// ElementE with the Sleeper parameter being nil will get the element
	// without retrying. Instead returning an error.
	el, err = page.ElementE(nil, "", "input")
	if rod.IsError(err, rod.ErrElementNotFound) {
		fmt.Println("element not found")
	} else {
		kit.E(err)
	}

	fmt.Println(el.Eval(`() => this.name`).String())

	// Output: q
}

// Example_customize_browser_launch will show how we can further customise the
// browser with the launcher library. The launcher lib comes with many default
// flags (switches), this example adds and removes a few.
func Example_customize_browser_launch() {
	// Documentation for default switches can be found at the source of the
	// launcher.New function, as well as at
	// https://peter.sh/experiments/chromium-command-line-switches/.
	url := launcher.New().
		// Set a flag- Adding the HTTP proxy server.
		Set("proxy-server", "127.0.0.1:8080").
		// Delete a flag- remove the mock-keychain flag
		Delete("use-mock-keychain").
		Launch()

	browser := rod.New().ControlURL(url).Connect()
	defer browser.Close()

	// Adding authentication to the proxy, for the next auth request.
	// We use CLI tool "mitmproxy --proxyauth user:pass" as an example.
	browser.HandleAuth("user", "pass")

	// mitmproxy needs a cert config to support https. We use http here instead,
	// for example
	fmt.Println(browser.Page("http://example.com/").Element("title").Text())

	// Skip
	// Output: Example Domain
}

// Example_direct_cdp shows how we can use Rod when it doesn't have a function
// or a feature that you would like to use. You can easily call the cdp
// interface.
func Example_direct_cdp() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	// The code here shows how SetCookies works.
	// Normally, you use something like
	// browser.Page("").SetCookies(...).Navigate(url).

	page := browser.Page("")

	// Call the cdp interface directly.
	// We set the cookie before we visit the website.
	// The "proto" lib contains every JSON schema you may need to communicate
	// with browser
	res, err := proto.NetworkSetCookie{
		Name:  "rod",
		Value: "test",
		URL:   "https://example.com",
	}.Call(page)
	kit.E(err)

	fmt.Println(res.Success)

	page.Navigate("https://example.com")

	// Eval injects a script into the page. We use this to return the cookies
	// that JS detects to validate our cdp call.
	cookie := page.Eval(`() => document.cookie`).String()

	fmt.Println(cookie)

	// You can also use your own raw JSON to send a json request.
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

// Example_handle_events is an example showing how we can use Rod to subscribe
// to events.
func Example_handle_events() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	go browser.EachEvent(func(e *proto.TargetTargetCreated) {
		// We only want to listen to events which are when a Page is created.
		// This filters out all other target types are created, e.g service
		// workers, shared workers, and background pages.
		if e.TargetInfo.Type != proto.TargetTargetInfoTypePage {
			return
		}

		// We use the ID to obtain the page ourselves to use.
		page02 := browser.PageFromTargetID(e.TargetInfo.TargetID)

		// Log "hey" on each newly created page.
		page02.Eval(`() => console.log("hey")`)
	})()

	page01 := browser.Page("")

	// Listen to all events of console output.
	go page01.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		page01.ObjectsToJSON(e.Args).Join(" ")
	})()

	// You subscribe to events before they occur. To start listening and
	// consuming to the elements, you must run wait().
	// Subscribe events before they happen, run the "wait()" to start consuming
	// the events. We return an optional stop signal at the first event to halt
	// the event subscription.
	wait := page01.EachEvent(func(e *proto.PageLoadEventFired) (stop bool) {
		return true
	})

	page01.Navigate("https://example.com")

	wait()

	// WaitEvent allows us to achieve the same functionality as above.
	// It waits until the event is called once.
	if false {
		page01.WaitEvent(&proto.PageLoadEventFired{})()
	}

	fmt.Println("done")

	// Output:
	// done
}

// Example_hijack_requests shows how we can intercept requests and modify
// both the request or the response.
func Example_hijack_requests() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	router := browser.HijackRequests()
	defer router.Stop()

	router.Add("*.js", func(ctx *rod.Hijack) {
		// Here we update the request's header. Rod gives functionality to
		// change or update all parts of the request. Refer to the documentation
		// for more information.
		ctx.Request.SetHeader("My-Header", "test")

		// LoadResponse runs the default request to the destination of the request.
		// Not calling this will require you to mock the entire response.
		// This can be done with the SetXxx (Status, Header, Body) functions on the
		// response struct.
		ctx.LoadResponse()

		// Here we update the body of all requests to update the document title to "hi"
		ctx.Response.SetBody(ctx.Response.StringBody() + "\n document.title = 'hi' ")
	})

	go router.Run()

	browser.Page("https://www.wikipedia.org/").Wait(`() => document.title === 'hi'`)

	fmt.Println("done")

	// Output: done
}

// Example_states allows us to update the state of the current page.
// In this example we enable network access.
func Example_states() {
	browser := rod.New().Timeout(time.Minute).Connect()
	defer browser.Close()

	page := browser.Page("")

	// LoadState detects whether the  network is enabled or not.
	fmt.Println(page.LoadState(&proto.NetworkEnable{}))

	_ = proto.NetworkEnable{}.Call(page)

	// Now that we called the request on the page, we check see if the state
	// result updates to true.
	fmt.Println(page.LoadState(&proto.NetworkEnable{}))

	// Output:
	// false
	// true
}
