package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

var username = flag.String("u", "", "user name")
var password = flag.String("p", "", "password")

func main() {
	flag.Parse()

	// This is an example of exporting and importing cookies

	cookies := login()

	reuse(cookies)

}

func login() string {

	// Launch a new browser with default options, and connect to it.
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("https://leetcode.com/accounts/login/")

	page.Element("#id_login").Input(*username)
	page.Element("#id_password").Input(*password).Press(input.Enter)

	errSelector := ".error-message__27FL"

	// Here we race two selectors, wait until one resolves
	el := page.Element(".nav-user-icon-base", errSelector)

	if el.Matches(errSelector) {
		panic(el.Text())
	}

	// print user name
	fmt.Println(*el.Attribute("title"))

	b, err := json.Marshal(page.Cookies())
	if err != nil {
		panic(err)
	}

	return string(b)

}

func reuse(cookies string) {

	var c []*proto.NetworkCookieParam
	err := json.Unmarshal([]byte(cookies), &c)
	if err != nil {
		panic(err)
	}

	// Launch a new browser with default options, and connect to it.
	browser := rod.New().Connect()
	defer browser.Close()

	page := browser.Page("").SetCookies(c...).Navigate("https://leetcode.com/accounts/login/")

	el := page.Element(".nav-user-icon-base")

	// print user name
	fmt.Println(*el.Attribute("title"))

}
