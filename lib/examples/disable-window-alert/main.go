package main

import (
	"fmt"
	"net/http"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const (
	indexHTML = `<!doctype html>
<html>
<body>
  
<script language="javascript" type="text/javascript">
alert("信息不存在");
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

	url, _ := launcher.New().Headless(false).Launch()
	browser := rod.New().ControlURL(url).MustConnect()

	// Creating a Page Object
	page, _ := browser.Page("")

	// Evaluates given script in every frame upon creation
	// Disable all alerts by making window.alert no-op.
	page.MustEvalOnNewDocument(`window.alert = () => {}`)

	// Navigate to the website you want to visit
	page.Navigate(host)

	fmt.Println(page.MustElement("script").Text())

}
