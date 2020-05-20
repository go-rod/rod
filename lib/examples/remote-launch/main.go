package main

import (
	"fmt"

	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/launcher"
)

func main() {
	// Launch chrome remotely
	client := launcher.NewRemote("ws://localhost:9222").Client()

	browser := rod.New().Client(client).Connect()

	// You may want to start a server to watch the screenshots inside the docker
	browser.ServeMonitor(":7777")

	fmt.Println(
		browser.Page("https://github.com").Eval("() => document.title"),
	)
}
