package cdp

import (
	"os"

	"github.com/ysmood/kit"
)

func checkPanic(err error) {
	if err == nil {
		return
	}
	panic(kit.Sdump(err))
}

var isDebug = os.Getenv("debug_cdp") == "true"

func debug(obj interface{}) {
	if !isDebug {
		return
	}

	kit.Log(kit.Sdump(obj))
}
