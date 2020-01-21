package rod

import "strings"

func docQuerySelector(selector string) string {
	selector = strings.ReplaceAll(selector, `"`, `\"`)
	return `document.querySelector("` + selector + `")`
}
