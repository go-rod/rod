package main

import (
	"fmt"
	"testing"

	"github.com/go-rod/rod"
)

func Test(t *testing.T) {
	for i := 0; i < 256; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			browser := rod.New().MustConnect()
			browser.MustPage("file:///Users/ys/repos/rod/fixtures/blank.html")
			browser.MustClose()
		})
	}
}
