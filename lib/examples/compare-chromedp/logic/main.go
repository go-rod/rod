package main

import (
	"log"
	"time"

	"github.com/go-rod/rod"
)

// On awesome-go page, finding the specified section sect,
// and retrieving the associated projects from the page.
func main() {
	page := rod.New().Connect().Timeout(time.Second * 15).Page("https://github.com/avelino/awesome-go")

	section := page.ElementMatches("p", "Selenium and browser control tools").Next()

	// query children elements of an element
	projects := section.Elements("li")

	for _, project := range projects {
		link := project.Element("a")
		log.Printf(
			"project %s (%s): '%s'",
			link.Text(),
			link.Property("href"),
			project.Text(),
		)
	}
}
