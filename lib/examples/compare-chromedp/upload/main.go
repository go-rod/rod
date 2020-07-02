package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-rod/rod"
	"github.com/ysmood/kit"
)

var flagPort = flag.Int("port", 8544, "port")

// This example demonstrates how to upload a file on a form.
func main() {
	flag.Parse()

	// start upload server
	go uploadServer(fmt.Sprintf(":%d", *flagPort))

	page := rod.New().Connect().Page(fmt.Sprintf("http://localhost:%d", *flagPort))

	page.Element(`input[name="upload"]`).SetFiles("./main.go")
	page.Element(`input[name="submit"]`).Click()

	log.Printf(
		"original size: %d, upload size: %s",
		size("./main.go"),
		page.Element("#result").Text(),
	)
}

// get some info about the file
func size(file string) int64 {
	fi, err := os.Stat(file)
	kit.E(err)
	return fi.Size()
}

func uploadServer(addr string) {
	// create http server and result channel
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		kit.E(fmt.Fprintf(res, uploadHTML))
	})
	mux.HandleFunc("/upload", func(res http.ResponseWriter, req *http.Request) {
		f, _, err := req.FormFile("upload")
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		defer kit.E(f.Close())

		buf, err := ioutil.ReadAll(f)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		kit.E(fmt.Fprintf(res, resultHTML, len(buf)))
	})
	kit.E(http.ListenAndServe(addr, mux))
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
