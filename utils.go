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
