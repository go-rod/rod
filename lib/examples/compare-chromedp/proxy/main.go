// Package main ...
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func main() {
	p := httptest.NewServer(newProxy())
	defer p.Close()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "test")
	}))
	defer s.Close()

	url := launcher.New().Proxy(p.URL).Set("proxy-bypass-list", "<-loopback>").MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	browser.MustIgnoreCertErrors(true)

	go browser.MustHandleAuth("u", "p")()

	page := browser.MustPage(s.URL)

	page.MustNavigate(s.URL + "/tab")
}

// newProxy creates a proxy that requires authentication.
func newProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			if dump, err := httputil.DumpRequest(r, true); err == nil {
				log.Printf("%s", dump)
			}
			// hardcode username/password "u:p" (base64 encoded: dTpw ) to make it simple
			if auth := r.Header.Get("Proxy-Authorization"); auth != "Basic dTpw" {
				r.Header.Set("X-Failed", "407")
			}
		},
		Transport: &transport{http.DefaultTransport},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err.Error() == "407" {
				log.Println("proxy: not authorized")
				w.Header().Add("Proxy-Authenticate", `Basic realm="Proxy Authorization"`)
				w.WriteHeader(407)
			} else {
				w.WriteHeader(http.StatusBadGateway)
			}
		},
	}
}

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	if h := r.Header.Get("X-Failed"); h != "" {
		return nil, fmt.Errorf(h)
	}
	return t.RoundTripper.RoundTrip(r)
}
