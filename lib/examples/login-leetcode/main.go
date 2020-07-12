package main

import (
	"flag"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

var username = flag.String("u", "", "user name")
var password = flag.String("p", "", "password")

func main() {
	flag.Parse()

	// Launch a new browser with default options, and connect to it.
	browser := rod.New().Connect()

	page := browser.Page("https://leetcode.com/accounts/login/")

	page.Element("#id_login").Input(*username)
	page.Element("#id_password").Input(*password).Press(input.Enter)

	errSelector := ".error-message__27FL"
	el := page.Element(".nav-user-icon-base", errSelector)

	if el.Matches(errSelector) {
		panic(el.Text())
	}

	// print user name
	fmt.Println(*el.Attribute("title"))
}
