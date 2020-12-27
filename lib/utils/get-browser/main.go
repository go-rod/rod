package main

import (
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

func main() {
	utils.E(launcher.NewBrowser().Get())
}
