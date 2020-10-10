package main

import (
	"fmt"
	"net/http"

	"github.com/go-rod/rod"
)

const (
	indexHTML = `<!doctype html>
<html>
<body>
  
<script language="javascript" type="text/javascript">
alert("Information does not exist");
</script>

</body>
</html>`
)

// Server creates a simple HTTP server that pop-up alert .
func Server(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		_, _ = fmt.Fprintf(res, indexHTML)
	})
	_ = http.ListenAndServe(addr, mux)
}

var port = 8544
var host = "http://localhost:8544"

func main() {
	// start cookie server
	go Server(fmt.Sprintf(":%d", port))

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// Creating a Page Object
	page := browser.MustPage("")

	// Evaluates given script in every frame upon creation
	// Disable all alerts by making window.alert no-op.
	page.MustEvalOnNewDocument(`window.alert = () => {}`)

	// Navigate to the website you want to visit
	page.MustNavigate(host)

	fmt.Println(page.MustElement("script").Text())

}
