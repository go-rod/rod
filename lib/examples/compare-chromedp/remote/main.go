// Package main ...
package main

import (
	"flag"
	"log"

	"github.com/go-rod/rod"
)

var flagDevToolWsURL = flag.String("devtools-ws-url", "", "DevTools WebSocket URL")

// This example demonstrates how to connect to an existing Chrome DevTools
// instance using a remote WebSocket URL.
func main() {
	flag.Parse()
	if *flagDevToolWsURL == "" {
		log.Fatal("must specify -devtools-ws-url")
	}

	page := rod.New().ControlURL(*flagDevToolWsURL).MustConnect().MustPage("https://duckduckgo.com")

	page.MustElement("#logo_homepage_link").MustWaitVisible()

	log.Println("Body of duckduckgo.com starts with:")
	log.Println(page.MustHTML()[0:100])
}
