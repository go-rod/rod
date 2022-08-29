// Package main ...
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/go-rod/rod"
)

// This example demonstrates how to set a HTTP header on requests.
func main() {
	host := headerServer()

	page := rod.New().MustConnect().MustPage(host)

	page.MustSetExtraHeaders("X-Header", "my request header")
	page.MustNavigate(host)
	res := page.MustElement("#result").MustText()

	log.Printf("received headers: %s", res)
}

// headerServer is a simple HTTP server that displays the passed headers in the html.
func headerServer() string {
	l, _ := net.Listen("tcp4", "127.0.0.1:0")
	go func() {
		_ = http.Serve(l, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			buf, err := json.MarshalIndent(req.Header, "", "  ")
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			_, _ = fmt.Fprintf(res, indexHTML, string(buf))
		}))
	}()
	return "http://" + l.Addr().String()
}

const indexHTML = `<!doctype html>
<html>
<body>
  <div id="result">%s</div>
</body>
</html>`
