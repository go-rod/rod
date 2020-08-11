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
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://leetcode.com/accounts/login/")

	page.MustElement("#id_login").MustInput(*username)
	page.MustElement("#id_password").MustInput(*password).MustPress(input.Enter)

	errSelector := ".error-message__27FL"

	// Here we race two selectors, wait until one resolves
	el := page.MustElement(".nav-user-icon-base", errSelector)

	if el.MustMatches(errSelector) {
		panic(el.MustText())
	}

	// print user name
	fmt.Println(*el.MustAttribute("title"))

	b, err := json.Marshal(page.MustCookies())
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
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("").MustSetCookies(c...).MustNavigate("https://leetcode.com/accounts/login/")

	el := page.MustElement(".nav-user-icon-base")

	// print user name
	fmt.Println(*el.MustAttribute("title"))

}
