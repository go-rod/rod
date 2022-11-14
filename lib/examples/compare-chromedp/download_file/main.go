// Package main ...
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func main() {
	// get working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://github.com/chromedp/examples")

	page.MustElementR("summary", "Code").MustClick()

	wait := page.Browser().WaitDownload(wd)

	go browser.EachEvent(func(e *proto.PageDownloadProgress) bool {
		completed := "(unknown)"
		if e.TotalBytes != 0 {
			completed = fmt.Sprintf("%0.2f%%", e.ReceivedBytes/e.TotalBytes*100.0)
		}
		log.Printf("state: %s, completed: %s\n", e.State, completed)
		return e.State == proto.PageDownloadProgressStateCompleted
	})()

	page.MustElementR("a", "Download ZIP").MustClick()

	res := wait()

	log.Printf("wrote %s", filepath.Join(wd, res.GUID))
}
