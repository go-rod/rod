package cdp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ysmood/kit"
)

func prettyJSON(s interface{}) string {
	raw, ok := s.(json.RawMessage)
	if ok {
		var val interface{}
		_ = json.Unmarshal(raw, &val)
		return kit.Sdump(val)
	}

	return kit.Sdump(raw)
}

func (cdp *Client) debugLog(obj interface{}) {
	if !cdp.debug {
		return
	}

	prefix := time.Now().Format("[cdp] [2006-01-02 15:04:05]")

	switch val := obj.(type) {
	case *Request:
		kit.E(fmt.Fprintf(
			kit.Stdout,
			"%s %s %d %s %s %s\n",
			prefix,
			kit.C("-> req", "green"),
			val.ID,
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		))
	case *response:
		kit.E(fmt.Fprintf(kit.Stdout,
			"%s %s %d %s %s\n",
			prefix,
			kit.C("<- res", "yellow"),
			val.ID,
			prettyJSON(val.Result),
			kit.Sdump(val.Error),
		))
	case *Event:
		kit.E(fmt.Fprintf(kit.Stdout,
			"%s %s %s %s %s\n",
			prefix,
			kit.C("evt", "blue"),
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		))

	default:
		kit.Err(kit.Sdump(obj))
	}
}
