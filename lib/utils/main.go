package utils

import (
	"fmt"
	"runtime"

	"github.com/mgutz/ansi"
)

// E if the last arg is error, panic it
func E(args ...interface{}) []interface{} {
	err, ok := args[len(args)-1].(error)
	if ok {
		panic(err)
	}
	return args
}

// C color the string in console log
// It will be disabled on windows
func C(str, color string) string {
	if runtime.GOOS == "windows" {
		return str
	}
	return ansi.Color(fmt.Sprint(str), color)
}
