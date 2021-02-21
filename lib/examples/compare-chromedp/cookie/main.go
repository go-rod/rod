package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

var flagPort = flag.Int("port", 8544, "port")

// This example demonstrates how we can modify the cookies on a web page.
func main() {
	flag.Parse()

	// start cookie server
	go cookieServer(fmt.Sprintf(":%d", *flagPort))

	host := fmt.Sprintf("http://localhost:%d", *flagPort)
	expr := proto.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour).Unix())

	page := rod.New().MustConnect().MustPage("")

	page.MustSetCookies(&proto.NetworkCookieParam{
		Name:     "cookie1",
		Value:    "value1",
		Domain:   "localhost",
		HTTPOnly: true,
		Expires:  expr,
	}, &proto.NetworkCookieParam{
		Name:     "cookie2",
		Value:    "value2",
		Domain:   "localhost",
		HTTPOnly: true,
		Expires:  expr,
	})

	page.MustNavigate(host)

	// read network values
	log.Printf("%+v\n", page.MustCookies())

	// chrome received cookies
	log.Printf("chrome received cookies: %s", page.MustElement(`#result`).MustText())
}

// cookieServer creates a simple HTTP server that logs any passed cookies.
func cookieServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
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
	})
	_ = http.ListenAndServe(addr, mux)
}

const (
	indexHTML = `<!doctype html>
<html>
<body>
  <div id="result">%s</div>
</body>
</html>`
)
