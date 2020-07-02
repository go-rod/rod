package main

import (
	"flag"
	"log"

	"github.com/go-rod/rod"
)

var flagDevToolWsURL = flag.String("devtools-ws-url", "", "DevTools WebSsocket URL")

// This example demonstrates how to connect to an existing Chrome DevTools
// instance using a remote WebSocket URL.
func main() {
	flag.Parse()
	if *flagDevToolWsURL == "" {
		log.Fatal("must specify -devtools-ws-url")
	}

	page := rod.New().ControlURL(*flagDevToolWsURL).Connect().Page("https://duckduckgo.com")

	page.Element("#logo_homepage_link").WaitVisible()
	body := page.Element("html").HTML()

	log.Println("Body of duckduckgo.com starts with:")
	log.Println(body[0:100])
}
