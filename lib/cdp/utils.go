package cdp

import (
	"encoding/json"
	"log"
	"runtime/debug"

	"github.com/go-rod/rod/lib/utils"
)

func prettyJSON(s interface{}) string {
	raw, ok := s.(json.RawMessage)
	if ok {
		var val interface{}
		_ = json.Unmarshal(raw, &val)
		return utils.SDump(val)
	}

	return utils.SDump(s)
}

func defaultDebugLog(obj interface{}) {
	switch val := obj.(type) {
	case *Request:
		log.Printf(
			"[rod/cdp] %s %d %s %s %s\n",
			"->",
			val.ID,
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		)
	case *Response:
		log.Printf(
			"[rod/cdp] %s %d %s %s\n",
			"<-",
			val.ID,
			prettyJSON(val.Result),
			utils.SDump(val.Error),
		)
	case *Event:
		log.Printf(
			"[rod/cdp] %s %s %s %s\n",
			"evt",
			val.Method,
			val.SessionID,
			prettyJSON(val.Params),
		)

	default:
		log.Println(utils.SDump(obj), "\n"+string(debug.Stack()))
	}
}
