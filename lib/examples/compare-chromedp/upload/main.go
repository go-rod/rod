// Package main ...
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-rod/rod"
)

// This example demonstrates how to upload a file on a form.
func main() {
	host := uploadServer()

	page := rod.New().MustConnect().MustPage(host)

	page.MustElement(`input[name="upload"]`).MustSetFiles("./main.go")
	page.MustElement(`input[name="submit"]`).MustClick()

	log.Printf(
		"original size: %d, upload size: %s",
		size("./main.go"),
		page.MustElement("#result").MustText(),
	)
}

// get some info about the file
func size(file string) int {
	fi, err := os.Stat(file)
	if err != nil {
		panic(err)
	}
	return int(fi.Size())
}

func uploadServer() string {
	// create http server and result channel
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		_, _ = fmt.Fprint(res, uploadHTML)
	})
	mux.HandleFunc("/upload", func(res http.ResponseWriter, req *http.Request) {
		f, _, err := req.FormFile("upload")
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		defer func() { _ = f.Close() }()

		buf, err := ioutil.ReadAll(f)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		_, _ = fmt.Fprintf(res, resultHTML, len(buf))
	})
	l, _ := net.Listen("tcp4", "127.0.0.1:0")
	go func() { _ = http.Serve(l, mux) }()
	return "http://" + l.Addr().String()
}

const (
	uploadHTML = `<!doctype html>
<html>
<body>
  <form method="POST" action="/upload" enctype="multipart/form-data">
    <input name="upload" type="file"/>
    <input name="submit" type="submit"/>
  </form>
</body>
</html>`

	resultHTML = `<!doctype html>
<html>
<body>
  <div id="result">%d</div>
</body>
</html>`
)
