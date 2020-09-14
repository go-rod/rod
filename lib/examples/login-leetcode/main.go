package main

import (
	"fmt"
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

var username = os.Args[1]
var password = os.Args[2]

func main() {
	browser := rod.New().MustConnect()

	page := browser.MustPage("https://leetcode.com/accounts/login/")

	page.MustElement("#id_login").MustInput("user")
	page.MustElement("#id_password").MustInput("password").MustPress(input.Enter)

	// It will keep retrying until one selector has found a match
	page.Race().MustElement(".nav-user-icon-base", func(el *rod.Element) {
		// print the username after successful login
		fmt.Println(*el.MustAttribute("title"))
	}).MustElement("[data-cy=sign-in-error]", func(el *rod.Element) {
		// when wrong username or password
		panic(el.MustText())
	}).MustDo()
}
