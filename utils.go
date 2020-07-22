package rod

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/kit"
)

// Sleeper returns the default sleeper for retry, it uses backoff to grow the interval.
// The growth looks like: A(0) = 100ms, A(n) = A(n-1) * random[1.9, 2.1), A(n) < 1s
var Sleeper = func() kit.Sleeper {
	return kit.BackoffSleeper(100*time.Millisecond, time.Second, nil)
}

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

const jsHelperID = proto.RuntimeRemoteObjectID("rodJSHelper")

// Convert name and jsArgs to Page.Eval, the name is method name in the "lib/assets/helper.js".
func jsHelper(name string, jsArgs Array) (string, Array) {
	jsArgs = append(Array{jsHelperID}, jsArgs...)
	js := fmt.Sprintf(`(rod, ...args) => rod.%s.apply(this, args)`, name)
	return js, jsArgs
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
				err = fmt.Errorf("%w: %s", newErr(ErrValue, val), kit.MustToJSON(val))
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
	return ok && cdpErr.Code == -32000 && cdpErr.Message != "Argument should belong to the same JavaScript world as target object"
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

		if last == '=' {
			if r == '>' {
				if balanced {
					return true
				}
				return false
			}
			return false
		}
		last = r
	}
	return false
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/Data_URIs
var regDataURI = regexp.MustCompile(`\Adata:(.+?)?(;base64)?,`)

func parseDataURI(uri string) (string, []byte) {
	matches := regDataURI.FindStringSubmatch(uri)
	l := len(matches[0])
	contentType := matches[1]
	codec := matches[2]

	if codec == ";base64" {
		bin, _ := base64.StdEncoding.DecodeString(uri[l:])
		return contentType, bin
	}

	s, _ := url.PathUnescape(uri[l:])
	return matches[1], []byte(s)
}
