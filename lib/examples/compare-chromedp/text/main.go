package main

import (
	"log"
	"strings"

	"github.com/go-rod/rod"
)

// This example demonstrates  how to extract text from a specific element.
func main() {
	page := rod.New().Connect().Page("https://golang.org/pkg/time")

	res := page.Element("#pkg-overview").Text()
	log.Println(strings.TrimSpace(res))
}
