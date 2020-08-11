package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

var flagPort = flag.Int("port", 8544, "port")

// This example demonstrates how to send key events to an element.
func main() {

	flag.Parse()

	// run server
	go testServer(fmt.Sprintf(":%d", *flagPort))

	host := fmt.Sprintf("http://localhost:%d", *flagPort)

	page := rod.New().MustConnect().MustPage(host)

	val1 := page.MustElement("#input1").MustText()
	val2 := page.MustElement("#textarea1").MustInput("\\b\\b\\n\\naoeu\\n\\ntest1\\n\\nblah2\\n\\n\\t\\t\\t\\b\\bother box!\\t\\ntest4").MustText()
	val3 := page.MustElement("#input2").MustInput("test3").MustText()
	val4 := page.MustElement("#select1").MustPress(input.ArrowDown).MustPress(input.ArrowDown).MustEval("() => this.value").Raw

	log.Printf("#input1 value: %s", val1)
	log.Printf("#textarea1 value: %s", val2)
	log.Printf("#input2 value: %s", val3)
	log.Printf("#select1 value: %s", val4)
}

// testServer is a simple HTTP server that displays elements and inputs
func testServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(res, indexHTML)
	})
	_ = http.ListenAndServe(addr, mux)
}

const indexHTML = `<!doctype html>
<html>
<head>
  <title>example</title>
</head>
<body>
  <div id="box1" style="display:none">
    <div id="box2">
      <p>box2</p>
    </div>
  </div>
  <div id="box3">
    <h2>box3</h3>
    <p id="box4">
      box4 text
      <input id="input1" value="some value"><br><br>
      <textarea id="textarea1" style="width:500px;height:400px">textarea</textarea><br><br>
      <input id="input2" type="submit" value="Next">
      <select id="select1">
        <option value="one">1</option>
        <option value="two">2</option>
        <option value="three">3</option>
        <option value="four">4</option>
      </select>
    </p>
  </div>
</body>
</html>`
