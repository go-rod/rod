package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"log"
	"net/http"
)

var flagPort = flag.Int("port", 8544, "port")

// This example demonstrates how to set a HTTP header on requests.
func main() {
	flag.Parse()

	// run server
	go kit.E(headerServer(fmt.Sprintf(":%d", *flagPort)))

	browser := rod.New().Connect()
	defer browser.Close()

	host := fmt.Sprintf("http://localhost:%d", *flagPort)

	page := browser.Page(host)

	page.SetExtraHeaders("X-Header", "my request header")
	page.Navigate(host)
	res := page.Element("#result").Text()

	log.Printf("received headers: %s", res)
}

// headerServer is a simple HTTP server that displays the passed headers in the html.
func headerServer(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		buf, err := json.MarshalIndent(req.Header, "", "  ")
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		kit.E(fmt.Fprintf(res, indexHTML, string(buf)))
	})
	return http.ListenAndServe(addr, mux)
}

const indexHTML = `<!doctype html>
<html>
<body>
  <div id="result">%s</div>
</body>
</html>`
