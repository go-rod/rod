// Package main ...
package main

import (
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

// This example demonstrates how to use scroll screenshot to capture long page.
func main() {
	browser := rod.New().MustConnect()

	// capture entire browser viewport, returning jpg with quality=90
	buf, err := browser.MustPage("https://desktop.github.com/").MustWaitStable().ScrollScreenshot(&proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson.Int(90),
	})
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("fullScreenshot.png", buf, 0o644)
	if err != nil {
		panic(err)
	}
}
