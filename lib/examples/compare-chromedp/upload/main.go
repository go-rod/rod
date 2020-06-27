package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ysmood/kit"
)

var flagPort = flag.Int("port", 8544, "port")

// This example demonstrates how to upload a file on a form.
func main() {
	flag.Parse()

	// get wd
	wd, err := os.Getwd()
	kit.E(err)

	filepath := wd + "/main.go"

	// get some info about the file
	fi, err := os.Stat(filepath)
	kit.E(err)

	// start upload server
	result := make(chan int, 1)
	go kit.E(uploadServer(fmt.Sprintf(":%d", *flagPort), result))

	url := launcher.New().Headless(false).Launch()
	browser := rod.New().ControlURL(url).Connect()

	page := browser.Page(fmt.Sprintf("http://localhost:%d", *flagPort))

	page.Element(`input[name="upload"]`).SetFiles("./main.go")
	page.Element(`input[name="submit"]`).Click()

	page.Element("#result").Text()

	log.Printf("original size: %d, upload size: %d", fi.Size(), <-result)
}

func uploadServer(addr string, result chan int) error {
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

		result <- len(buf)
	})
	return http.ListenAndServe(addr, mux)
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
