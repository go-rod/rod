package main

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"io/ioutil"
)
func main() {
	// print webpage to pdf
	//
	browser := rod.New().MustConnect()
	page := browser.MustPage("")
	var e proto.NetworkResponseReceived

	wait := page.WaitEvent(&e)
	err := page.Navigate("https://www.google.com")
	if err != nil{
		panic(err)
	}
	wait() //waitting load complete
	//entire browser viewport
	page.MustElement("html")
	//parameters details https://chromedevtools.github.io/devtools-protocol/tot/Page/#method-printToPDF
	parameters  :=proto.PagePrintToPDF{Scale:1}
	pdf,_:= page.PDF(&parameters)
	
	err = ioutil.WriteFile("google.pdf",pdf,0666)
	if err != nil{
		panic(err)
	}

}
