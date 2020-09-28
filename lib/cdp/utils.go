package cdp

import (
	"log"

	"github.com/go-rod/rod/lib/utils"
)

func defaultDebugLog(obj interface{}) {
	switch val := obj.(type) {
	case *Request:
		log.Printf(
			"[rod/cdp] %s %d %s %s %s\n",
			"->",
			val.ID,
			val.Method,
			val.SessionID,
			utils.SDump(val.Params),
		)
	case *Response:
		log.Printf(
			"[rod/cdp] %s %d %s %s\n",
			"<-",
			val.ID,
			utils.SDump(val.Result),
			utils.SDump(val.Error),
		)
	case *Event:
		log.Printf(
			"[rod/cdp] %s %s %s %s\n",
			"evt",
			val.Method,
			val.SessionID,
			utils.SDump(val.Params),
		)

	default:
		log.Println(utils.SDump(obj))
	}
}
