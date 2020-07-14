package rod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/kit"
)

// Array of any type
type Array []interface{}

// ArrayFromList converts a random list into Array type
func ArrayFromList(list interface{}) Array {
	arr := Array{}
	val := reflect.ValueOf(list)
	for i := 0; i < val.Len(); i++ {
		arr = append(arr, val.Index(i).Interface())
	}
	return arr
}

// SprintFnThis wrap js with this, wrap function call if it's js expression
func SprintFnThis(js string) string {
	if detectJSFunction(js) {
		return fmt.Sprintf(`function() { return (%s).apply(this, arguments) }`, js)
	}
	return fmt.Sprintf(`function() { return %s }`, js)
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

// Try try fn with recover, return the panic as value
func Try(fn func()) (err error) {
	defer func() {
		if val := recover(); val != nil {
			var ok bool
			err, ok = val.(error)
			if !ok {
				err = &Error{ErrValue, val}
			}
		}
	}()

	fn()

	return err
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

func mustToJSONForDev(value interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	kit.E(enc.Encode(value))

	return buf.String()
}

// detect if a js string is a function definition
var regFn = regexp.MustCompile(`\A\s*function\s*\(`)

// detect if a js string is a function definition
// Samples:
//
// function () {}
// a => {}
// (a, b, c) =>
// ({a: b}, ...list) => {}
func detectJSFunction(js string) bool {
	if regFn.MatchString(js) {
		return true
	}

	// The algorithm is pretty simple, the braces before "=>" must be balanced.
	// Such as "foo(() => {})", there are 2 "(", but only 1 ")".
	// Here we use a simple state machine.

	balanced := true
	last := ' '
	for _, r := range js {
		if r == '(' {
			if balanced {
				balanced = false
			} else {
				return false
			}
		}
		if r == ')' {
			if balanced {
				return false
			}
			balanced = true
		}

		if last == '=' && r == '>' {
			if balanced {
				return true
			}
			return false
		}
		last = r
	}
	return false
}
