// Package main ...
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("./chrome: error while loading shared libraries: libcairo.so.2: cannot open shared object file: No such file or directory")
	os.Exit(1)
}
