// Package main ...
package main

import (
	"log"
	"os"

	"github.com/go-rod/rod"
)

func main() {
	u := "https://avatars.githubusercontent.com/u/33149672"

	browser := rod.New().MustConnect()

	page := browser.MustPage(u).MustWaitLoad()

	b, err := page.GetResource(u)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("download.png", b, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Print("wrote download.png")
}
