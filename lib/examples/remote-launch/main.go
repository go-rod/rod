package main

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

func main() {
	// To launch remote browsers, you need a remote launcher service,
	// Rod provides a docker image for beginers, make sure have started:
	// docker run -p 9222:9222 ghcr.io/go-rod/rod
	//
	// For more information, check the doc of launcher.RemoteLauncher
	l := launcher.MustNewRemote("ws://localhost:9222")

	// Manipulate flags like the example in examples_test.go
	l.Set("any-flag").Delete("any-flag")

	// Launch with headful mode
	l.Headless(false).XVFB()

	browser := rod.New().Client(l.Client()).MustConnect()

	// You may want to start a server to watch the screenshots inside the docker
	launcher.Open(browser.ServeMonitor(""))

	fmt.Println(
		browser.MustPage("https://www.wikipedia.org/").MustEval("() => document.title"),
	)

	utils.Pause()
}

// To manually launch a browser
func _() {
	// You can also manually launch a browser in the image:
	// docker run -p 9222:9222 ghcr.io/go-rod/rod chromium-browser --headless --no-sandbox --remote-debugging-port=9222 --remote-debugging-address=0.0.0.0
	u := launcher.MustResolveURL("9222")

	browser := rod.New().ControlURL(u).MustConnect()

	fmt.Println(
		browser.MustPage("https://github.com").MustEval("() => document.title"),
	)
}
