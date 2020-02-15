package cdp

import (
	"errors"
	"net"
	"os"

	"github.com/ysmood/kit"
)

func isClosedErr(err error) bool {
	var netErr *net.OpError
	return errors.As(err, &netErr) &&
		netErr.Err.Error() == "use of closed network connection"
}

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
