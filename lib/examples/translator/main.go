// Package main ...
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/go-rod/rod"
)

func main() {
	flag.Parse()

	// get the commandline arguments
	source := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if source == "" {
		log.Fatal("usage: go run main.go -- 'This is the phrase to translate to Spanish.'")
	}

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://translate.google.com/?sl=auto&tl=es&op=translate")

	el := page.MustElement(`textarea[aria-label="Source text"]`)

	wait := page.MustWaitRequestIdle("https://accounts.google.com")
	el.MustInput(source)
	wait()

	result := page.MustElement("[role=region] span[lang]").MustText()

	fmt.Println(result)
}
