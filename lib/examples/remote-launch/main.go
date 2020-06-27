package main

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func main() {
	// Launch browser remotely
	// docker run -p 9222:9222 rodorg/rod
	client := launcher.NewRemote("ws://localhost:9222").Client()

	browser := rod.New().Client(client).Connect()

	// You may want to start a server to watch the screenshots inside the docker
	browser.ServeMonitor(":7777")

	fmt.Println(
		browser.Page("https://github.com").Eval("() => document.title"),
	)
}
