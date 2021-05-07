package main

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

func main() {
	// To launch remote browsers, you must use a remote launcher service,
	// Don't launch the browser manually like "chrome --headless --remote-debugging-port=9222".
	// To connect to a running browser check the "../connect-browser" example.
	// Rod provides a docker image for beginers, run the below first:
	//
	//     docker run -p 7317:7317 ghcr.io/go-rod/rod
	//
	// For more information, check the doc of launcher.RemoteLauncher
	l := launcher.MustNewRemote("")

	// Manipulate flags like the example in examples_test.go
	l.Set("any-flag").Delete("any-flag")

	// Launch with headful mode
	l.Headless(false).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	browser := rod.New().Client(l.Client()).MustConnect()

	// You may want to start a server to watch the screenshots inside the docker
	launcher.Open(browser.ServeMonitor(""))

	fmt.Println(
		browser.MustPage("https://www.wikipedia.org/").MustEval("() => document.title"),
	)

	utils.Pause()
}
