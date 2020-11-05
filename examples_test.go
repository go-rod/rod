package rod_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
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

	defer l.Cleanup() // remove user-data-dir

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

// Rod use https://golang.org/pkg/context to handle cancelations for IO blocking operations, most times it's timeout.
// Context will be recursively passed to all sub-methods.
// For example, methods like Page.Context(ctx) will return a clone of the page with the ctx,
// all the methods of the returned page will use the ctx if they have IO blocking operations.
// Page.Timeout or Page.WithCancel is just a shortcut for Page.Context.
// Of course, Browser or Element works the same way.
func Example_context_and_timeout() {
	page := rod.New().MustConnect().MustPage("https://github.com")

	page.
		// Set a 5-second timeout for all chained methods
		Timeout(5 * time.Second).

		// The total time for MustWaitLoad and MustElement must be less than 5 seconds
		MustWaitLoad().
		MustElement("title").

		// Methods after CancelTimeout won't be affected by the 5-second timeout
		CancelTimeout().

		// Set a 10-second timeout for all chained methods
		Timeout(10 * time.Second).

		// Panics if it takes more than 10 seconds
		MustText()

	// The two code blocks below are basically the same:
	{
		page.Timeout(5 * time.Second).MustElement("a").CancelTimeout()
	}
	{
		// Use this way you can customize your own way to cancel long-running task
		page, cancel := page.WithCancel()
		go func() {
			time.Sleep(time.Duration(rand.Int())) // cancel after randomly time
			cancel()
		}()
		page.MustElement("a")
	}
}

// We use "Must" prefixed functions to write example code. But in production you may want to use
// the no-prefix version of them.
// About why we use "Must" as the prefix, it's similar to https://golang.org/pkg/regexp/#MustCompile
func Example_error_handling() {
	page := rod.New().MustConnect().MustPage("https://example.com")

	// We use Go's standard way to check error types, no magic.
	check := func(err error) {
		var evalErr *rod.ErrEval
		if errors.Is(err, context.DeadlineExceeded) { // timeout error
			fmt.Println("timeout err")
		} else if errors.As(err, &evalErr) { // eval error
			fmt.Println(evalErr.LineNumber)
		} else if err != nil {
			fmt.Println("can't handle", err)
		}
	}

	// The two code blocks below are doing the same thing in two styles:

	// The block below is better for debugging or quick scripting. We use panic to short-circuit logics.
	// So that we can take advantage of fluent interface (https://en.wikipedia.org/wiki/Fluent_interface)
	// and fail-fast (https://en.wikipedia.org/wiki/Fail-fast).
	// This style will reduce code, but it may also catch extra errors (less consistent and precise).
	{
		err := rod.Try(func() {
			fmt.Println(page.MustElement("a").MustHTML()) // use "Must" prefixed functions
		})
		check(err)
	}

	// The block below is better for production code. It's the standard way to handle errors.
	// Usually, this style is more consistent and precise.
	{
		el, err := page.Element("a")
		if err != nil {
			check(err)
			return
		}
		html, err := el.HTML()
		if err != nil {
			check(err)
			return
		}
		fmt.Println(html)
	}
}

// Example_search shows how to use Search to get element inside nested iframes or shadow DOMs.
// It works the same as https://developers.google.com/web/tools/chrome-devtools/dom#search
func Example_search() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://developer.mozilla.org/en-US/docs/Web/HTML/Element/iframe")

	// Click the zoom-in button of the OpenStreetMap
	page.MustSearch(".leaflet-control-zoom-in").MustClick()

	fmt.Println("done")

	// Output: done
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
	const username = ""
	const password = ""

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://leetcode.com/accounts/login/")

	page.MustElement("#id_login").MustInput(username)
	page.MustElement("#id_password").MustInput(password).MustPress(input.Enter)

	// It will keep retrying until one selector has found a match
	page.Race().MustElement(".nav-user-icon-base", func(el *rod.Element) {
		// print the username after successful login
		fmt.Println(*el.MustAttribute("title"))
	}).MustElement("[data-cy=sign-in-error]", func(el *rod.Element) {
		// when wrong username or password
		panic(el.MustText())
	}).MustDo()
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
	if errors.Is(err, &rod.ErrElementNotFound{}) {
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

	// The two code blocks below are equal to enable AD blocking

	{
		_ = proto.PageSetAdBlockingEnabled{
			Enabled: true,
		}.Call(page)
	}

	{
		// Interact with the cdp JSON API directly
		_, _ = page.Call(context.TODO(), "", "Page.setAdBlockingEnabled", map[string]bool{
			"enabled": true,
		})
	}
}

// Shows how to listen for events.
func Example_handle_events() {
	browser := rod.New().Timeout(time.Minute).MustConnect()
	defer browser.MustClose()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	page := browser.Context(ctx).MustPage("")

	done := make(chan int)

	// Listen for all events of console output. You can even listen to multiple types of events at that same time,
	// check the doc of EachEvent for details.
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		fmt.Println(page.MustObjectsToJSON(e.Args))
		close(done)
	})()

	wait := page.WaitEvent(&proto.PageLoadEventFired{})
	page.MustNavigate("https://example.com")
	wait()

	// EachEvent allows us to achieve the same functionality as above.
	if false {
		// Subscribe events before they happen, run the "wait()" to start consuming
		// the events. We can return an optional stop signal to unsubscribe events.
		wait := page.EachEvent(func(e *proto.PageLoadEventFired) (stop bool) {
			return true
		})
		page.MustNavigate("https://example.com")
		wait()
	}

	// Or the for-loop style to handle events to do the same thing above.
	if false {
		page.MustNavigate("https://example.com")

		for msg := range page.Event() {
			e := proto.PageLoadEventFired{}
			if msg.Load(&e) {
				break
			}
		}
	}

	page.MustEval(`console.log("hello", "world")`)

	<-done

	// Output:
	// [hello world]
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
func Example_eval_reuse_remote_object() {
	page := rod.New().MustConnect().MustPage("")

	fn := page.MustEvaluate(rod.Eval(`Math.random`).ByObject())

	res := page.MustEval(`f => f()`, fn)

	// print a random number
	fmt.Println(res.Num())
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

// It's a common practice to concurrently use a pool of resources in Go, it's not special for rod.
func ExamplePage_pool() {
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// We create a pool that will hold at most 3 pages
	pool := rod.NewPagePool(3)

	// Create a page if needed. If you want pages to share cookies with each remove the MustIncognito()
	create := func() *rod.Page { return browser.MustIncognito().MustPage("") }

	yourJob := func() {
		page := pool.Get(create)
		defer pool.Put(page)

		page.MustNavigate("http://example.com").MustWaitLoad()
		fmt.Println(page.MustInfo().Title)
	}

	// Run jobs concurrently
	wg := sync.WaitGroup{}
	for range "...." {
		wg.Add(1)
		go func() {
			defer wg.Done()
			yourJob()
		}()
	}
	wg.Wait()

	// cleanup pool
	pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	// Output:
	// Example Domain
	// Example Domain
	// Example Domain
	// Example Domain
}

func Example_load_extension() {
	extPath, _ := filepath.Abs("fixtures/chrome-extension")

	u := launcher.New().
		Set("load-extension", extPath). // must use abs path for an extension
		Headless(false).                // headless mode doesn't support extension yet
		MustLaunch()

	page := rod.New().ControlURL(u).MustConnect().MustPage("http://example.com")

	page.MustWait(`document.title === 'test-extension'`)

	fmt.Println("ok")

	// Skip
	// Output: ok
}
