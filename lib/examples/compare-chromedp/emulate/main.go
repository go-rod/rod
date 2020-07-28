package main

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
)

func main() {
	page := rod.New().Connect().Page("")

	// emulate iPhone 7 landscape
	err := page.EmulateE(devices.IPhone6or7or8, true)
	if err != nil {
		panic(err)
	}

	page.Navigate("https://www.whatsmyua.info/")
	page.Screenshot("screenshot1.png")

	// reset
	page.Emulate("")

	page.Viewport(1920, 2000, 1, false)
	page.Navigate("https://www.whatsmyua.info/?a")
	page.Screenshot("screenshot2.png")
}
