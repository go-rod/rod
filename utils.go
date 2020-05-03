package rod

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/proto"
)

// Array of any type
type Array []interface{}

// SprintFnApply is a helper to render template into js code
// js looks like "(a, b) => {}", the a and b are the params passed into the function
func sprintFnApply(js string, params Array) string {
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
func Method(event proto.Event) EventFilter {
	return func(e *cdp.Event) bool {
		return event.MethodName() == e.Method
	}
}

func isNilContextErr(err error) bool {
	if err == nil {
		return false
	}
	cdpErr, ok := err.(*cdp.Error)
	return ok && cdpErr.Code == -32000
}

func matchWithFilter(s string, includes, excludes []string) bool {
	for _, include := range includes {
		if regexp.MustCompile(include).MatchString(s) {
			for _, exclude := range excludes {
				if regexp.MustCompile(exclude).MatchString(s) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func saveScreenshot(bin []byte, toFile []string) {
	if len(toFile) == 0 {
		return
	}
	if toFile[0] == "" {
		toFile = []string{"tmp", "screenshots", fmt.Sprintf("%d", time.Now().UnixNano()) + ".png"}
	}
	kit.E(kit.OutputFile(filepath.Join(toFile...), bin, nil))
}
