// Package main ...
package main

import (
	"io/ioutil"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

// This example demonstrates how to take a screenshot of a specific element and
// of the entire browser viewport, as well as using `kit`
// to store it into a file.
func main() {
	browser := rod.New().MustConnect()

	// capture screenshot of an element
	browser.MustPage("https://google.com").MustElement("body div").MustScreenshot("elementScreenshot.png")

	// capture entire browser viewport, returning jpg with quality=90
	buf, err := browser.MustPage("https://brank.as/").Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson.Int(90),
	})
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("fullScreenshot.png", buf, 0o644)
	if err != nil {
		panic(err)
	}
}
