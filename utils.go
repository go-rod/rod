package rod

import (
	"strings"

	"github.com/ysmood/kit"
)

// Fn is a helper to render template into js code
// fn looks like "function(a, b) {}", the a and b are the params passed into the function
func Fn(fn string, params ...interface{}) string {
	const tpl = `function() {
		return ({{.fn}}).apply(this, {{.params}})
	}`

	return kit.S(tpl, "fn", fn, "params", kit.MustToJSON(params))
}

func docQuerySelector(selector string) string {
	selector = strings.ReplaceAll(selector, `"`, `\"`)
	return `document.querySelector("` + selector + `")`
}
