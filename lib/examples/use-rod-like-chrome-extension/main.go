package main

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// For example, when you log into your github account, and you want to reuse the login session for automation task.
// You can use this example to achieve such functionality. Rod will be just like your browser extension.
func main() {
	// Make sure you have closed your browser completely, UserMode can't control a browser that is not launched by it.
	// Launches a new browser with the "new user mode" option, and returns the URL to control that browser.
	url := launcher.NewUserMode().MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect().DefaultViewport(nil)

	browser.MustPage("https://github.com")
}
