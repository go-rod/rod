package main

import (
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/go-rod/rod"
)

// This example shows how to detect the hanging points of golang code.
// It's actually a general way to debug any golang project.
func main() {
	debug.SetTraceback("all")

	go func() {
		wait()
		panic("exit")
	}()

	yourCodeHere()
}

// Put your code here, press Ctrl+C when you feel the program is hanging.
// Read each goroutine's stack that is related to your own code logic.
func yourCodeHere() {
	rod.New().MustConnect().MustPage("http://example.com").MustElement("not-exists")
}

// For this example you will find something like this below:

/*
goroutine 1 [select]:
github.com/go-rod/rod.(*Page).Element(0xc000434000, 0xc000098d10, 0x1, 0x1, 0xc000000300)
	rod/sugar.go:363 +0x8e
main.yourCodeHere()
	rod/lib/examples/debug-deadlock/main.go:26 +0xa4
main.main()
	rod/lib/examples/debug-deadlock/main.go:20 +0x53
*/

// Now you know the line 26's Element is blocking the code.

func wait(signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	if len(signals) == 0 {
		signals = append(signals, os.Interrupt)
	}
	signal.Notify(c, signals...)
	<-c
	signal.Stop(c)
	close(c)
}
