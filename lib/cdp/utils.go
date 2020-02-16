package cdp

import (
	"encoding/json"
	"errors"
	"fmt"
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

// Debug is the flag to enable debug log to stdout. The default value is os.Getenv("debug_cdp") == "true"
var Debug = os.Getenv("debug_cdp") == "true"

func prettyJSON(s *JSON) string {
	if s == nil {
		return ""
	}
	var val interface{}
	kit.E(json.Unmarshal([]byte(s.Raw), &val))
	return kit.Sdump(val)
}

func debug(obj interface{}) {
	if !Debug {
		return
	}

	if obj == nil {
		return
	}

	switch val := obj.(type) {
	case *Request:
		kit.E(fmt.Fprintf(
			kit.Stdout,
			"[cdp] %s %d %s %s %s\n",
			kit.C("req", "green"),
			val.ID,
			val.Method,
			val.SessionID,
			kit.Sdump(val.Params),
		))
	case *Response:
		kit.E(fmt.Fprintf(kit.Stdout,
			"[cdp] %s %d %s %s\n",
			kit.C("res", "yellow"),
			val.ID,
			prettyJSON(val.Result),
			kit.Sdump(val.Error),
		))
	case *Event:
		kit.E(fmt.Fprintf(kit.Stdout,
			"[cdp] %s %s %s %s\n",
			kit.C("evt", "blue"),
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		))
	default:
		kit.Log(kit.Sdump(obj))
	}
}
