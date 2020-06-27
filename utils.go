package rod

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/kit"
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

// Event helps to convert a cdp.Event to proto.Payload. Returns false if the conversion fails
func Event(msg *cdp.Event, evt proto.Payload) bool {
	if msg.Method == evt.MethodName() {
		err := json.Unmarshal(msg.Params, evt)
		return err == nil
	}
	return false
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

func saveScreenshot(bin []byte, toFile []string) error {
	if len(toFile) == 0 {
		return nil
	}
	if toFile[0] == "" {
		toFile = []string{"tmp", "screenshots", fmt.Sprintf("%d", time.Now().UnixNano()) + ".png"}
	}
	return kit.OutputFile(filepath.Join(toFile...), bin, nil)
}

func ginHTML(ctx kit.GinContext, body string) {
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = ctx.Writer.WriteString(body)
}
