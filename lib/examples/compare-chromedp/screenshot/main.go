package main

import (
	"io/ioutil"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/kit"
)

// This example demonstrates how to take a screenshot of a specific element and
// of the entire browser viewport, as well as using `kit`
// to store it into a file.
func main() {
	browser := rod.New().Connect()

	// capture screenshot of an element
	browser.Page("https://google.com").Element("#main").Screenshot("elementScreenshot.png")

	// capture entire browser viewport, returning png with quality=90
	buf, err := browser.Page("https://brank.as/").ScreenshotE(true, &proto.PageCaptureScreenshot{
		Format:  "png",
		Quality: 90,
	})
	kit.E(err)
	kit.E(ioutil.WriteFile("fullScreenshot.png", buf, 0644))
}
