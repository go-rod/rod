package main

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"io/ioutil"
	"time"
)
func main() {
	// save webpage to pdf
	//
	browser := rod.New().MustConnect()
	page := browser.MustPage("")
	err := page.Navigate("https://news.ycombinator.com")
	if err != nil{
		panic(err)
	}
	include :=[]string{"ycombinator"} //regular expressions that match the waitting url. 
	exclude :=[]string{"none"} // reg that filter noise.
	wait := page.WaitRequestIdle(1*time.Second,include,exclude)
	wait()

	//parameters details https://chromedevtools.github.io/devtools-protocol/tot/Page/#method-printToPDF
	parameters  :=proto.PagePrintToPDF{Scale:1}
	pdf,_:= page.PDF(&parameters)

	err = ioutil.WriteFile("hn.pdf",pdf,0666)
	if err != nil{
		panic(err)
	}

}

