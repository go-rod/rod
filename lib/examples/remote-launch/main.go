package main

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ysmood/kit"
)

func main() {
	// Launch browser remotely
	// docker run -p 9222:9222 rodorg/rod
	l := launcher.NewRemote("ws://localhost:9222")

	// Manipulate flags like the example in examples_test.go
	l.Set("window-size", "1920,1080").Delete("any-flag")

	browser := rod.New().Client(l.Client()).MustConnect()

	// You may want to start a server to watch the screenshots inside the docker
	browser.ServeMonitor(":7777", true)

	fmt.Println(
		browser.MustPage("https://github.com").MustEval("() => document.title"),
	)

	kit.Pause()
}
