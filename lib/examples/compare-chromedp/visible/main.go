package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-rod/rod"
)

var flagPort = flag.Int("port", 8544, "port")

func main() {
	flag.Parse()

	// run server
	go testServer(fmt.Sprintf(":%d", *flagPort))

	page := rod.New().Connect().Page(fmt.Sprintf("http://localhost:%d", *flagPort))
	page.Eval(makeVisibleScript)

	log.Printf("waiting 3s for box to become visible")

	page.Element("#box1").WaitVisible()
	log.Printf(">>>>>>>>>>>>>>>>>>>> BOX1 IS VISIBLE")

	page.Element("#box2").WaitVisible()
	log.Printf(">>>>>>>>>>>>>>>>>>>> BOX2 IS VISIBLE")
}

const (
	makeVisibleScript = `() => setTimeout(function() {
	document.querySelector('#box1').style.display = '';
}, 3000)`
)

// testServer is a simple HTTP server that serves a static html page.
func testServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(res, indexHTML)
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
