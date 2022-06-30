package main

import (
	"fmt"

	"github.com/TommyLeng/go-rod/lib/launcher"
	"github.com/TommyLeng/go-rod/lib/utils"
)

func main() {
	p, err := launcher.NewBrowser().Get()
	utils.E(err)

	fmt.Println(p)
}
