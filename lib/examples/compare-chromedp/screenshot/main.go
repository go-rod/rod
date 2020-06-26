package main

import (
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/proto"
	"io/ioutil"
)

// This example demonstrates how to take a screenshot of a specific element and
// of the entire browser viewport, as well as using `kit`
// to store it into a file.
func main() {
	browser := rod.New().Connect()
	defer browser.Close()

	browser.Page("https://google.com").Element("#main").Screenshot("elementScreenshot.png")

	buf, err := browser.Page("https://brank.as/").ScreenshotE(true, &proto.PageCaptureScreenshot{
		Format:  "png",
		Quality: 90,
	})
	kit.E(err)
	kit.E(ioutil.WriteFile("fullScreenshot.png", buf, 0644))
}
