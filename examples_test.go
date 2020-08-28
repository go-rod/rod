package rod_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Example_basic is a simple test that opens https://github.com/, searches for
// "git", and then gets the header element which gives the description for Git.
func Example_basic() {
	// Launch a new browser with default options, and connect to it.
	browser := rod.New().MustConnect()

	// Even you forget to close, rod will close it after main process ends.
	defer browser.MustClose()

	// Timeout will be passed to all chained function calls.
	// The code will panic out if any chained call is used after the timeout.
	page := browser.Timeout(time.Minute).MustPage("https://github.com")

	// Make sure viewport is always consistent.
	page.MustViewport(1200, 600, 1, false)

	// We use css selector to get the search input element and input "git"
	page.MustElement("input").MustInput("git").MustPress(input.Enter)

	// Wait until css selector get the element then get the text content of it.
	// You can also pass multiple selectors to race the result, useful when dealing with multiple possible results.
	text := page.MustElement(".codesearch-results p").MustText()

	fmt.Println(text)

	// Get all input elements. Rod supports query elements by css selector, xpath, and regex.
	// For more detailed usage, check the query_test.go file.
	fmt.Println(len(page.MustElements("input")))

	// Eval js on the page
	page.MustEval(`console.log("hello world")`)

	// Pass parameters as json objects to the js function. This one will return 3
	fmt.Println(page.MustEval(`(a, b) => a + b`, 1, 2).Int())

	// When eval on an element, you can use "this" to access the DOM element.
	fmt.Println(page.MustElement("title").MustEval(`this.innerText`).String())

	// To handle errors in rod, you can use rod.Try or E suffixed function family like "page.ElementE"
	// https://github.com/go-rod/rod#q-why-functions-dont-return-error-values
	err := rod.Try(func() {
		// Here we will catch timeout or query error
		page.Timeout(time.Second / 2).MustElement("element-not-exists")
	})
	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("after 0.5 seconds, the element is still not rendered")
	}

	// Output:
	// Git is the most widely used version control system.
	// 5
	// 3
	// Search · git · GitHub
	// after 0.5 seconds, the element is still not rendered
}

// Example_search shows how to use Search to get element inside nested iframes or shadow DOMs.
// It works the same as https://developers.google.com/web/tools/chrome-devtools/dom#search
func Example_search() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://developer.mozilla.org/en-US/docs/Web/HTML/Element/iframe")

	// get the code mirror editor inside the iframe
	el := page.MustSearch(".CodeMirror")

	fmt.Println(*el.MustAttribute("class"))

	// Output: CodeMirror cm-s-default CodeMirror-wrap
}

// Example_headless_with_debug shows how we can start a browser with debug
// information and headless mode disabled to show the browser in the foreground.
// Rod provides a lot of debug options, you can use the Set method to enable
// them or use environment variables. (Default environment variables can be
// found in "lib/defaults").
func Example_headless_with_debug() {
	// Headless runs the browser on foreground, you can also use env "rod=show"
	// Devtools opens the tab in each new tab opened automatically
	l := launcher.New().
		Headless(false).
		Devtools(true)
	defer l.Cleanup()
	url := l.MustLaunch()

	// Trace shows verbose debug information for each action executed
	// Slowmotion is a debug related function that waits 2 seconds between
	// each action, making it easier to inspect what your code is doing.
	browser := rod.New().
		Timeout(time.Minute).
		ControlURL(url).
		Trace(true).
		Slowmotion(2 * time.Second).
		MustConnect()

	// ServeMonitor plays screenshots of each tab. This feature is extremely
	// useful when debugging with headless mode.
	browser.ServeMonitor(":9777", true)

	defer browser.MustClose()

	page := browser.MustPage("https://www.wikipedia.org/")

	page.MustElement("#searchLanguage").MustSelect("[lang=zh]")
	page.MustElement("#searchInput").MustInput("热干面")
	page.Keyboard.MustPress(input.Enter)

	fmt.Println(page.MustElement("#firstHeading").MustText())

	// Response gets the binary of the image as a []byte.
	img := page.MustElement(`[alt="Hot Dry Noodles.jpg"]`).MustResource()
	fmt.Println(len(img)) // print the size of the image

	// Pause temporarily halts JavaScript execution on the website.
	// You can resume execution in the devtools window by clicking the resume
	// button in the "source" tab.
	page.MustPause()

	// Skip
	// Output: 热干面
}

// Example_wait_for_animation is an example to simulate humans more accurately.
// If a button is moving too fast, you cannot click it as a human. To more
// accurately simulate human inputs, actions triggered by Rod can be based on
// mouse point location, so you can allow Rod to wait for the button to become
// stable before it tries clicking it.
func Example_wait_for_animation() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://getbootstrap.com/docs/4.0/components/modal/")

	page.MustWaitLoad().MustElement("[data-target='#exampleModalLive']").MustClick()

	saveBtn := page.MustElementMatches("#exampleModalLive button", "Close")

	// Here, WaitStable will wait until the save button's position becomes
	// stable. The timeout is 5 seconds, after which it times out (or after 1
	// minute since the browser was created). Timeouts from parents are
	// inherited to the children as well.
	saveBtn.Timeout(5 * time.Second).MustWaitStable().MustClick().MustWaitInvisible()

	fmt.Println("done")

	// Output: done
}

// Example_wait_for_request shows an example where Rod will wait for all
// requests on the page to complete (such as network request) before interacting
// with the page.
func Example_wait_for_request() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://duckduckgo.com/")

	// WaitRequestIdle will wait for all possible ajax calls to complete before
	// continuing on with further execution calls.
	wait := page.MustWaitRequestIdle()
	page.MustElement("#search_form_input_homepage").MustClick().MustInput("test")
	time.Sleep(300 * time.Millisecond) // Wait for js debounce.
	wait()

	// We want to make sure that after waiting, there are some autocomplete
	// suggestions available.
	fmt.Println(len(page.MustElements(".search__autocomplete .acp")) > 0)

	// Output: true
}

// Example_customize_retry_strategy allows us to change the retry/polling
// options that is used to query elements. This is useful when you want to
// customize the element query retry logic.
func Example_customize_retry_strategy() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://github.com")

	// sleep for 0.5 seconds before every retry
	sleeper := func(context.Context) error {
		time.Sleep(time.Second / 2)
		return nil
	}
	el, _ := page.Sleeper(sleeper).Element("input")

	// If sleeper is nil page.ElementE will query without retrying.
	// If nothing found it will return an error.
	el, err := page.Sleeper(nil).Element("input")
	if errors.Is(err, rod.ErrElementNotFound) {
		fmt.Println("element not found")
	} else if err != nil {
		panic(err)
	}

	fmt.Println(el.MustEval(`this.name`).String())

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
		MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	// Adding authentication to the proxy, for the next auth request.
	// We use CLI tool "mitmproxy --proxyauth user:pass" as an example.
	browser.MustHandleAuth("user", "pass")

	// mitmproxy needs a cert config to support https. We use http here instead,
	// for example
	fmt.Println(browser.MustPage("http://example.com/").MustElement("title").MustText())

	// Skip
	// Output: Example Domain
}

// Example_direct_cdp shows how we can use Rod when it doesn't have a function
// or a feature that you would like to use. You can easily call the cdp
// interface.
func Example_direct_cdp() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	// The code here shows how SetCookies works.
	// Normally, you use something like
	// browser.Page("").SetCookies(...).Navigate(url).

	page := browser.MustPage("")

	// Call the cdp interface directly.
	// We set the cookie before we visit the website.
	// The "proto" lib contains every JSON schema you may need to communicate
	// with browser
	res, err := proto.NetworkSetCookie{
		Name:  "rod",
		Value: "test",
		URL:   "https://example.com",
	}.Call(page)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.Success)

	page.MustNavigate("https://example.com")

	// Eval injects a script into the page. We use this to return the cookies
	// that JS detects to validate our cdp call.
	cookie := page.MustEval(`document.cookie`).String()

	fmt.Println(cookie)

	// You can also use your own raw JSON to send a json request.
	params, _ := json.Marshal(map[string]string{
		"name":  "rod",
		"value": "test",
		"url":   "https://example.com",
	})
	ctx, client, sessionID := page.CallContext()
	_, _ = client.Call(ctx, sessionID, "Network.SetCookie", params)

	// Output:
	// true
	// rod=test
}

// Example_handle_events is an example showing how we can use Rod to subscribe
// to events.
func Example_handle_events() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

	done := make(chan int)

	// Listen to all events of console output.
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		log := page.MustObjectsToJSON(e.Args).Join(" ")
		fmt.Println(log)
		close(done)
	})()

	wait := page.WaitEvent(&proto.PageLoadEventFired{})
	page.MustNavigate("https://example.com")
	wait()

	// EachEvent allows us to achieve the same functionality as above.
	if false {
		// Subscribe events before they happen, run the "wait()" to start consuming
		// the events. We can return an optional stop signal unsubscribe events.
		wait := page.EachEvent(func(e *proto.PageLoadEventFired) (stop bool) {
			return true
		})
		page.MustNavigate("https://example.com")
		wait()
	}

	page.MustEval(`console.log("hello", "world")`)

	<-done

	// Output:
	// hello world
}

// Example_hijack_requests shows how we can intercept requests and modify
// both the request and the response.
func Example_hijack_requests() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	router := browser.HijackRequests()
	defer router.MustStop()

	router.MustAdd("*.js", func(ctx *rod.Hijack) {
		// Here we update the request's header. Rod gives functionality to
		// change or update all parts of the request. Refer to the documentation
		// for more information.
		ctx.Request.Req().Header.Set("My-Header", "test")

		// LoadResponse runs the default request to the destination of the request.
		// Not calling this will require you to mock the entire response.
		// This can be done with the SetXxx (Status, Header, Body) functions on the
		// response struct.
		ctx.MustLoadResponse()

		// Here we append some code to every js file.
		// The code will update the document title to "hi"
		ctx.Response.SetBody(ctx.Response.Body() + "\n document.title = 'hi' ")
	})

	go router.Run()

	browser.MustPage("https://www.wikipedia.org/").MustWait(`document.title === 'hi'`)

	fmt.Println("done")

	// Output: done
}

// Example_states allows us to update the state of the current page.
// In this example we enable network access.
func Example_states() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

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
