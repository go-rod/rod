// Package main ...
package main

import (
	"log"
	"time"

	"github.com/go-rod/rod"
)

// On awesome-go page, finding the specified section sect,
// and retrieving the associated projects from the page.
func main() {
	page := rod.New().MustConnect().Timeout(time.Second * 15).MustPage("https://github.com/avelino/awesome-go")

	section := page.MustElementR("p", "Selenium and browser control tools").MustNext()

	// query children elements of an element
	projects := section.MustElements("li")

	for _, project := range projects {
		link := project.MustElement("a")
		log.Printf(
			"project %s (%s): '%s'",
			link.MustText(),
			link.MustProperty("href"),
			project.MustText(),
		)
	}
}
