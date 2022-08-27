// Package main ...
package main

import (
	"fmt"
	"net/http"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/utils"
)

func main() {
	go serve()

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// Creating a Page Object
	page := browser.MustPage()

	// Evaluates given script in every frame upon creation
	// Disable all alerts by making window.alert no-op.
	page.MustEvalOnNewDocument(`window.alert = () => {}`)

	// Navigate to the website you want to visit
	page.MustNavigate("http://localhost:8080")

	fmt.Println(page.MustElement("script").MustText())
}

const testPage = `<html><script>alert("message")</script></html>`

// mock a server
func serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		utils.E(fmt.Fprint(res, testPage))
	})
	utils.E(http.ListenAndServe(":8080", mux))
}
