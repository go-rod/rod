// Package main ...
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/go-rod/rod"
)

func main() {
	page := rod.New().MustConnect().MustPage(testServer())
	page.MustEval(makeVisibleScript)

	log.Printf("waiting 3s for box to become visible")

	page.MustElement("#box1").MustWaitVisible()
	log.Printf(">>>>>>>>>>>>>>>>>>>> BOX1 IS VISIBLE")

	page.MustElement("#box2").MustWaitVisible()
	log.Printf(">>>>>>>>>>>>>>>>>>>> BOX2 IS VISIBLE")
}

const (
	makeVisibleScript = `() => setTimeout(function() {
	document.querySelector('#box1').style.display = '';
}, 3000)`
)

// testServer is a simple HTTP server that serves a static html page.
func testServer() string {
	l, _ := net.Listen("tcp4", "127.0.0.1:0")
	go func() {
		_ = http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, indexHTML)
		}))
	}()
	return "http://" + l.Addr().String()
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
