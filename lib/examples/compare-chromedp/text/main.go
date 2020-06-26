package main

import (
	"github.com/ysmood/rod"
	"log"
	"strings"
)

// This example demonstrates  how to extract text from a specific element.
func main() {
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("https://golang.org/pkg/time")

	res := page.Element("#pkg-overview").Text()
	log.Println(strings.TrimSpace(res))
}
