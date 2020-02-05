package rod

import (
	"fmt"

	"github.com/ysmood/kit"
)

// SprintFnApply is a helper to render template into js code
// js looks like "(a, b) => {}", the a and b are the params passed into the function
func SprintFnApply(js string, params []interface{}) string {
	const tpl = `(
		%s
	).apply(this, %s)`

	return fmt.Sprintf(tpl, js, kit.MustToJSON(params))
}

// SprintFnThis wrap js with this
func SprintFnThis(js string) string {
	return fmt.Sprintf(`function() {
		return (%s).apply(this, arguments)
	}`, js)
}

// IsEmpty checks if the js value is null or undefined
func IsEmpty(val kit.JSONResult) bool {
	theType := val.Get("type").String()
	subType := val.Get("subtype").String()

	switch theType {
	case "object":
		return subType == "null"
	case "undefined":
		return true
	default:
		return false
	}
}

// Pause execution
func Pause() {
	<-make(chan kit.Nil)
}
