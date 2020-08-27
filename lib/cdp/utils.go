package cdp

import (
	"encoding/json"
	"log"
	"runtime/debug"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/kit"
)

func prettyJSON(s interface{}) string {
	raw, ok := s.(json.RawMessage)
	if ok {
		var val interface{}
		_ = json.Unmarshal(raw, &val)
		return kit.Sdump(val)
	}

	return kit.Sdump(s)
}

func defaultDebugLog(obj interface{}) {
	switch val := obj.(type) {
	case *Request:
		log.Printf(
			"[rod/cdp] %s %d %s %s %s\n",
			utils.C("->", "green"),
			val.ID,
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		)
	case *Response:
		log.Printf(
			"[rod/cdp] %s %d %s %s\n",
			utils.C("<-", "yellow"),
			val.ID,
			prettyJSON(val.Result),
			kit.Sdump(val.Error),
		)
	case *Event:
		log.Printf(
			"[rod/cdp] %s %s %s %s\n",
			utils.C("evt", "blue"),
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		)

	default:
		log.Println(kit.Sdump(obj), "\n"+string(debug.Stack()))
	}
}
