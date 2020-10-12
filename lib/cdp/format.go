package cdp

import (
	"fmt"

	"github.com/go-rod/rod/lib/utils"
)

func (req Request) String() string {
	return fmt.Sprintf(
		"=> #%d %s %s %s",
		req.ID,
		fSessionID(req.SessionID),
		req.Method,
		dump(req.Params),
	)
}

func (res Response) String() string {
	if res.Error != nil {
		return fmt.Sprintf(
			"<= #%d error: %s",
			res.ID,
			dump(res.Error),
		)
	}
	return fmt.Sprintf(
		"<= #%d %s",
		res.ID,
		dump(res.Result),
	)
}

func (e Event) String() string {
	return fmt.Sprintf(
		"<- %s %s %s",
		fSessionID(e.SessionID),
		e.Method,
		dump(e.Params),
	)
}

func fSessionID(s string) string {
	if s == "" {
		s = "00000000"
	}
	s = s[:8]
	return "@" + s
}

func dump(v interface{}) string {
	return utils.MustToJSON(v)
}
