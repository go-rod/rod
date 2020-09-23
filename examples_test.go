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
	"github.com/go-rod/rod/lib/utils"
)

// This example opens https://github.com/, searches for "git",
// and then gets the header element which gives the description for Git.
func Example() {
	// Launch a new browser with default options, and connect to it.
	browser := rod.New().MustConnect()

	// Even you forget to close, rod will close it after main process ends.
	defer browser.MustClose()

	// Timeout will be passed to all chained function calls.
	// The code will panic out if any chained call is used after the timeout.
	page := browser.Timeout(time.Minute).MustPage("https://github.com")

	// We use css selector to get the search input element and input "git"
	page.MustElement("input").MustInput("git").MustPress(input.Enter)

	// Wait until css selector get the element then get the text content of it.
	text := page.MustElement(".codesearch-results p").MustText()

	fmt.Println(text)

	// Get all input elements. Rod supports query elements by css selector, xpath, and regex.
	// For more detailed usage, check the query_test.go file.
	fmt.Println("Found", len(page.MustElements("input")), "input elements")

	// Eval js on the page
	page.MustEval(`console.log("hello world")`)

	// Pass parameters as json objects to the js function. This MustEval will result 3
	fmt.Println("1 + 2 =", page.MustEval(`(a, b) => a + b`, 1, 2).Int())

	// When eval on an element, "this" in the js is the current DOM element.
	fmt.Println(page.MustElement("title").MustEval(`this.innerText`).String())

	// Output:
	// Git is the most widely used version control system.
	// Found 5 input elements
	// 1 + 2 = 3
	// Search · git · GitHub
}

// Shows how to disable headless mode and debug.
// Rod provides a lot of debug options, you can set them with setter methods or use environment variables.
// Doc for environment variables: https://pkg.go.dev/github.com/go-rod/rod/lib/defaults
func Example_disable_headless_to_debug() {
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
	// You can also enable it with env rod=monitor
	launcher.NewBrowser().Open(browser.ServeMonitor(""))

	defer browser.MustClose()

	page := browser.MustPage("https://www.wikipedia.org/")

	page.MustElement("#searchLanguage").MustSelect("[lang=zh]")
	page.MustElement("#searchInput").MustInput("热干面")
	page.Keyboard.MustPress(input.Enter)

	fmt.Println(page.MustElement("#firstHeading").MustText())

	// Response gets the binary of the image as a []byte.
	img := page.MustElement(`[alt="Hot Dry Noodles.jpg"]`).MustResource()
	fmt.Println(len(img)) // print the size of the image

	utils.Pause() // pause goroutine
}

// Usage of timeout context
func Example_timeout_handling() {
	page := rod.New().MustConnect().MustPage("https://github.com")

	page.
		// Set a 5-second timeout for all chained actions
		Timeout(5 * time.Second).

		// The total time for MustWaitLoad and MustElement must be less than 5 seconds
		MustWaitLoad().
		MustElement("title").

		// Actions after CancelTimeout won't be affected by the 5-second timeout
		CancelTimeout().

		// Set a 10-second timeout for all chained actions
		Timeout(10 * time.Second).

		// Panics if it takes more than 10 seconds
		MustText()

	// The two code blocks below are basically the same:
	{
		page.Timeout(5 * time.Second).MustElement("a").CancelTimeout()
	}
	{
		// Use this way you can customize your own way to cancel long-running task
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		page.Context(ctx).MustElement("a")
		cancel()
	}
}

// We use "Must" prefixed functions to write example code. But in production you may want to use
// the no-prefix version of them.
// About why we use "Must" as the prefix, it's similar with https://golang.org/pkg/regexp/#MustCompile
func Example_error_handling() {
	page := rod.New().MustConnect().MustPage("https://example.com")

	// The two code blocks below are basically the same:

	// The block below is better for production code. It follows the standards of golang error handling.
	// Usually, this style will make error handling more consistent and precisely.
	{
		el, err := page.Element("a")
		if err != nil {
			fmt.Print(err)
			return
		}
		html, err := el.HTML()
		if err != nil {
			fmt.Print(err)
			return
		}
		fmt.Println(html)
	}

	// The block below is better for example code or quick scripting. We use panic to short-circuit logics.
	// So that we can code in fluent style: https://en.wikipedia.org/wiki/Fluent_interface
	// It will reduce the code to type, but it may also catch extra errors (less consistent and precisely).
	{
		err := rod.Try(func() {
			fmt.Println(page.MustElement("a").MustHTML())
		})
		fmt.Print(err)
	}

	// Catch specified error types
	{
		_, err := page.Timeout(3 * time.Second).Eval(`foo()`)
		if errors.Is(err, context.DeadlineExceeded) { // timeout error
			fmt.Println("timeout err")
		} else if errors.Is(err, rod.ErrEval) { // eval error
			// print the stack trace
			fmt.Printf("%+v\n", err)

			// print more details
			utils.Dump(rod.AsError(err).Details)
		} else if err != nil {
			panic(err)
		}
	}
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

func Example_page_screenshot() {
	page := rod.New().MustConnect().MustPage("")

	wait := page.MustWaitNavigation()
	page.MustNavigate("https://github.com")
	wait() // until the navigation to settle down

	// simple version
	page.MustScreenshot("my.png")

	// customization version
	img, _ := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: 90,
		Clip: &proto.PageViewport{
			X:      0,
			Y:      0,
			Width:  300,
			Height: 200,
			Scale:  1,
		},
		FromSurface: true,
	})
	_ = utils.OutputFile("my.jpg", img)
}

func Example_page_pdf() {
	page := rod.New().MustConnect().MustPage("")

	wait := page.MustWaitNavigation()
	page.MustNavigate("https://github.com")
	wait() // until the navigation to settle down

	// simple version
	page.MustPDF("my.pdf")

	// customization version
	pdf, _ := page.PDF(&proto.PagePrintToPDF{
		PaperWidth:              8.5,
		PaperHeight:             11,
		PageRanges:              "1-3",
		IgnoreInvalidPageRanges: false,
		DisplayHeaderFooter:     true,
	})
	_ = utils.OutputFile("my.pdf", pdf)
}

// Show how to handle multiple results of an action.
// Such as when you login a page, the result can be success or wrong password.
func Example_race_selectors() {
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("http://testing-ground.scraping.pro/login")

	page.MustElement("#usr").MustInput("admin")
	page.MustElement("#pwd").MustInput("12345").MustPress(input.Enter)

	// It will keep retrying until one selector has found a match
	page.Race().MustElement("h3.success", func(el *rod.Element) {
		// when successful login
		fmt.Println(el.MustText())
	}).MustElement("h3.error", func(el *rod.Element) {
		// when wrong username or password
		fmt.Println(el.MustText())
	}).MustDo()

	// Output: WELCOME :)
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

	saveBtn := page.MustElementR("#exampleModalLive button", "Close")

	// Here, WaitStable will wait until the save button's position becomes
	// stable. The timeout is 5 seconds, after which it times out (or after 1
	// minute since the browser was created). Timeouts from parents are
	// inherited to the children as well.
	saveBtn.Timeout(5 * time.Second).MustWaitStable().MustClick().MustWaitInvisible()

	fmt.Println("done")

	// Output: done
}

// When you want to wait for an ajax request to complete, this example will be useful.
func Example_wait_for_request() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://duckduckgo.com/")

	// Start to analyze request events
	wait := page.MustWaitRequestIdle()

	// This will trigger the search ajax request
	page.MustElement("#search_form_input_homepage").MustClick().MustInput("test")

	// Wait until there's no active requests
	wait()

	// We want to make sure that after waiting, there are some autocomplete
	// suggestions available.
	fmt.Println(len(page.MustElements(".search__autocomplete .acp")) > 0)

	// Output: true
}

// Shows how to change the retry/polling options that is used to query elements.
// This is useful when you want to customize the element query retry logic.
func Example_customize_retry_strategy() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://github.com")

	// sleep for 0.5 seconds before every retry
	sleeper := func() utils.Sleeper {
		return func(context.Context) error {
			time.Sleep(time.Second / 2)
			return nil
		}
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

// Shows how we can further customize the browser with the launcher library.
// Usually you use launcher lib to set the browser's command line flags (switches).
// Doc for flags: https://peter.sh/experiments/chromium-command-line-switches
func Example_customize_browser_launch() {
	url := launcher.New().
		Proxy("127.0.0.1:8080").     // set flag "--proxy-server=127.0.0.1:8080"
		Delete("use-mock-keychain"). // delete flag "--use-mock-keychain"
		MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	// So that we don't have to self issue certs for MITM
	browser.MustIgnoreCertErrors(true)

	// Adding authentication to the proxy, for the next auth request.
	// We use CLI tool "mitmproxy --proxyauth user:pass" as an example.
	browser.MustHandleAuth("user", "pass")

	// mitmproxy needs a cert config to support https. We use http here instead,
	// for example
	fmt.Println(browser.MustPage("https://example.com/").MustElement("title").MustText())
}

// When rod doesn't have a feature that you need. You can easily call the cdp to achieve it.
// List of cdp API: https://chromedevtools.github.io/devtools-protocol
func Example_direct_cdp() {
	page := rod.New().MustConnect().MustPage("")

	// Rod doesn't have a method to enable AD blocking,
	// but you can call cdp interface directly to achieve it.
	// Doc: https://chromedevtools.github.io/devtools-protocol/tot/Page/#method-setAdBlockingEnabled
	_ = proto.PageSetAdBlockingEnabled{
		Enabled: true,
	}.Call(page)

	// You can even use JSON directly to do the same thing above.
	params, _ := json.Marshal(map[string]bool{
		"enabled": true,
	})
	ctx, client, id := page.CallContext()
	_, _ = client.Call(ctx, id, "Page.setAdBlockingEnabled", params)
}

// Shows how to listen for events.
func Example_handle_events() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	page := browser.Context(ctx).MustPage("")

	done := make(chan int)

	// Listen for all events of console output.
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

// Shows how to intercept requests and modify
// both the request and the response.
// The entire process of hijacking one request:
//    browser --req-> rod ---> server ---> rod --res-> browser
// The --req-> and --res-> are the parts that can be modified.
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
		// ctx.Response struct.
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

// Shows how to share a remote object reference between two Eval
func Example_reuse_remote_object() {
	page := rod.New().MustConnect().MustPage("")

	fn, _ := page.EvalWithOptions(&rod.EvalOptions{JS: `Math.random`})

	res, _ := page.EvalWithOptions(&rod.EvalOptions{
		ByValue: true,
		JSArgs: rod.JSArgs{
			fn.ObjectID, // use remote function as the js argument x
		},
		JS: `x => x()`,
	})

	// print a random number
	fmt.Println(res.Value.Num)
}

// Shows how to update the state of the current page.
// In this example we enable the network domain.
func Example_states() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

	// LoadState detects whether the network domain is enabled or not.
	fmt.Println(page.LoadState(&proto.NetworkEnable{}))

	_ = proto.NetworkEnable{}.Call(page)

	// Check if the network domain is successfully enabled.
	fmt.Println(page.LoadState(&proto.NetworkEnable{}))

	// Output:
	// false
	// true
}
