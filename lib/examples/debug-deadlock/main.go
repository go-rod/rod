// Package main ...
package main

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/ysmood/gotrace"
)

// This example shows how to detect the hanging points of golang code.
// It's actually a general way to debug any golang project.
func main() {
	defer checkLock()()

	go yourCodeHere()
}

// Put your code here, press Ctrl+C when you feel the program is hanging.
// Read each goroutine's stack that is related to your own code logic.
func yourCodeHere() {
	page := rod.New().MustConnect().MustPage("http://mdn.dev")
	go page.MustElement("not-exists")
}

// For this example you will find something like this below:

/*
goroutine 7 [select]:
github.com/go-rod/rod.(*Page).MustElement(0xc00037e000, 0xc00063a0f0, 0x1, 0x1, 0x0)
	rod/must.go:425 +0x4d
created by main.yourCodeHere
	rod/lib/examples/debug-deadlock/main.go:22 +0xb8
*/

// From it we know the line 22 is blocking the code.

func checkLock() func() {
	ctx := gotrace.Signal()
	ignored := gotrace.IgnoreCurrent()
	return func() {
		fmt.Println(gotrace.Wait(ctx, ignored))
	}
}
