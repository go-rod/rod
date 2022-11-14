// Package main ...
package main

import (
	"io/ioutil"
	"log"

	"github.com/go-rod/rod"
)

func main() {
	urlstr := "https://avatars.githubusercontent.com/u/33149672"

	browser := rod.New().MustConnect()

	page := browser.MustPage(urlstr).MustWaitLoad()

	b, err := page.GetResource(urlstr)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("download.png", b, 0644); err != nil {
		log.Fatal(err)
	}
	log.Print("wrote download.png")
}
