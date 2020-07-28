package main

import (
	"io/ioutil"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// This example demonstrates how to take a screenshot of a specific element and
// of the entire browser viewport, as well as using `kit`
// to store it into a file.
func main() {
	browser := rod.New().Connect()

	// capture screenshot of an element
	browser.Page("https://google.com").Element("#main").Screenshot("elementScreenshot.png")

	// capture entire browser viewport, returning jpg with quality=90
	buf, err := browser.Page("https://brank.as/").ScreenshotE(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: 90,
	})
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("fullScreenshot.png", buf, 0644)
	if err != nil {
		panic(err)
	}
}
