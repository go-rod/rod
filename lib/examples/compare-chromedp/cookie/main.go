// Package main ...
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// This example demonstrates how we can modify the cookies on a web page.
func main() {
	expr := proto.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour).Unix())

	page := rod.New().MustConnect().MustPage()

	page.MustSetCookies(&proto.NetworkCookieParam{
		Name:     "cookie1",
		Value:    "value1",
		Domain:   "127.0.0.1",
		HTTPOnly: true,
		Expires:  expr,
	}, &proto.NetworkCookieParam{
		Name:     "cookie2",
		Value:    "value2",
		Domain:   "127.0.0.1",
		HTTPOnly: true,
		Expires:  expr,
	})

	page.MustNavigate(cookieServer())

	// read network values
	for i, cookie := range page.MustCookies() {
		log.Printf("chrome cookie %d: %+v", i, cookie)
	}

	// chrome received cookies
	log.Printf("chrome received cookies: %s", page.MustElement(`#result`).MustText())
}

// cookieServer creates a simple HTTP server that logs any passed cookies.
func cookieServer() string {
	l, _ := net.Listen("tcp4", "127.0.0.1:0")
	go func() {
		_ = http.Serve(l, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			cookies := req.Cookies()
			for i, cookie := range cookies {
				log.Printf("from %s, server received cookie %d: %v", req.RemoteAddr, i, cookie)
			}
			buf, err := json.MarshalIndent(req.Cookies(), "", "  ")
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			_, _ = fmt.Fprintf(res, indexHTML, string(buf))
		}))
	}()
	return "http://" + l.Addr().String()
}

const (
	indexHTML = `<!doctype html>
<html>
<body>
  <div id="result">%s</div>
</body>
</html>`
)
