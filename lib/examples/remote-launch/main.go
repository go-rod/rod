package main

import (
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/launcher"
)

func main() {
	// Also, use the env var "rod=remote,url=wss://a.com" can achieve the same config.

	lc := launcher.
		NewRemote("ws://localhost:9222").
		Set("disable-sync") // config chrome flags

	browser := rod.New().Remote(lc).Connect()

	browser.Page("https://github.com")
}
