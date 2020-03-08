package rod

import (
	"context"
	"fmt"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// SprintFnApply is a helper to render template into js code
// js looks like "(a, b) => {}", the a and b are the params passed into the function
func sprintFnApply(js string, params cdp.Array) string {
	const tpl = `(%s).apply(this, %s)`

	return fmt.Sprintf(tpl, js, kit.MustToJSON(params))
}

// SprintFnThis wrap js with this
func SprintFnThis(js string) string {
	return fmt.Sprintf(`function() { return (%s).apply(this, arguments) }`, js)
}

// CancelPanic graceful panic
func CancelPanic(err error) {
	if err != nil && err != context.Canceled {
		panic(err)
	}
}

// Method creates a method filter
func Method(name string) EventFilter {
	return func(e *cdp.Event) bool {
		return name == e.Method
	}
}

func isNilContextErr(err error) bool {
	if err == nil {
		return false
	}
	cdpErr, ok := err.(*cdp.Error)
	return ok && cdpErr.Code == -32000
}
