package main

import (
	"fmt"

	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/launcher"
)

func main() {
	// Also, use the env var "rod=remote" can achieve the same config.
	client := launcher.NewRemote("ws://localhost:9222").Client()

	browser := rod.New().Client(client).Connect()

	fmt.Println(
		browser.Page("https://github.com").Eval("() => document.title"),
	)
}
