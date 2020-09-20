package main
import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func main() {
	// This demo shows how to capture a screenshot
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
	// a specific element
	page.MustElement("body").MustScreenshot("ElementScreenshot.png")
	//entire browser viewport
	page.MustElement("html").MustScreenshot("Screenshot.png")
}
