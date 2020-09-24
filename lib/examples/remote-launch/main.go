package main

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

func main() {
	// Launch browser remotely
	// docker run -p 9222:9222 rodorg/rod
	l := launcher.MustNewRemote("ws://localhost:9222")

	// Manipulate flags like the example in examples_test.go
	l.Set("window-size", "1920,1080").Delete("any-flag")

	browser := rod.New().Client(l.Client()).MustConnect()

	// You may want to start a server to watch the screenshots inside the docker
	launcher.NewBrowser().Open(browser.ServeMonitor(""))

	fmt.Println(
		browser.MustPage("https://github.com").MustEval("() => document.title"),
	)

	utils.Pause()
}
