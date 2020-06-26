package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/proto"
	"log"
	"net/http"
	"time"
)

var flagPort = flag.Int("port", 8544, "port")

// This example demonstrates how we can modify the cookies on a web page.
func main() {
	flag.Parse()

	// start cookie server
	go kit.E(cookieServer(fmt.Sprintf(":%d", *flagPort)))

	browser := rod.New().Connect()
	defer browser.Close()

	host := fmt.Sprintf("http://localhost:%d", *flagPort)

	res := setcookies(browser.Page(""),
		host,
		"cookie1", "value1",
		"cookie2", "value2",
	)

	log.Printf("chrome received cookies: %s", res)
}

// cookieServer creates a simple HTTP server that logs any passed cookies.
func cookieServer(addr string) error {
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
		kit.E(fmt.Fprintf(res, indexHTML, string(buf)))
	})
	return http.ListenAndServe(addr, mux)
}

// setcookies runs a task to navigate to a host with the passed cookies set
// on the network request.
func setcookies(page *rod.Page, host string, cookies ...string) (res string) {
	if len(cookies)%2 != 0 {
		panic("length of cookies must be divisible by 2")
	}

	expr := proto.TimeSinceEpoch{Time: time.Now().Add(180 * 24 * time.Hour)}

	cookieList := make([]*proto.NetworkCookieParam, 0)
	for i := 0; i < len(cookies); i += 2 {
		cookieList = append(cookieList, &proto.NetworkCookieParam{
			Name:     cookies[i],
			Value:    cookies[i+1],
			Domain:   "localhost",
			HTTPOnly: true,
			Expires:  &expr,
		})
	}

	page.SetCookies(cookieList...)

	page.Navigate(host)

	res = page.Element(`#result`).Text()

	for i, cookie := range page.Cookies() {
		log.Printf("chrome cookie: %d: %+v", i, cookie)
	}

	return
}

const (
	indexHTML = `<!doctype html>
<html>
<body>
  <div id="result">%s</div>
</body>
</html>`
)
