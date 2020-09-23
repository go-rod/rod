package rod

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// Client interface
type Client interface {
	Connect(ctx context.Context) error
	Event() <-chan *cdp.Event
	Call(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error)
}

// DefaultSleeper generates the default sleeper for retry, it uses backoff to grow the interval.
// The growth looks like: A(0) = 100ms, A(n) = A(n-1) * random[1.9, 2.1), A(n) < 1s
var DefaultSleeper = func() utils.Sleeper {
	return utils.BackoffSleeper(100*time.Millisecond, time.Second, nil)
}

func ensureSleeper(gen func() utils.Sleeper) func() utils.Sleeper {
	if gen == nil {
		return func() utils.Sleeper { return nil }
	}
	return gen
}

// JSArgs for eval
type JSArgs []interface{}

// JSArgsFromString converts a string list into Array type
func JSArgsFromString(list []string) JSArgs {
	arr := JSArgs{}
	for _, s := range list {
		arr = append(arr, s)
	}
	return arr
}

// EvalOptions for EvalWithOptions
type EvalOptions struct {
	// If enabled the eval result will be a JSON value.
	// If disabled the eval result will be a RuntimeRemoteObjectID.
	ByValue bool

	// ThisID represents the "this" object in the JS
	ThisID proto.RuntimeRemoteObjectID

	// JS code to eval
	JS string

	// JSArgs represents the arguments in the JS if the JS is a function definition.
	// If an argument is proto.RuntimeRemoteObjectID type, the corresponding remote object will be used.
	// Or it will be passed as a JSON value.
	JSArgs JSArgs

	// Whether execution should be treated as initiated by user in the UI.
	UserGesture bool
}

// This set the ThisID
func (e *EvalOptions) This(id proto.RuntimeRemoteObjectID) *EvalOptions {
	e.ThisID = id
	return e
}

// ByObject disables ByValue.
func (e *EvalOptions) ByObject() *EvalOptions {
	e.ByValue = false
	return e
}

// ByUser enables UserGesture.
func (e *EvalOptions) ByUser() *EvalOptions {
	e.UserGesture = true
	return e
}

// NewEvalOptions instance. ByValue will be set to true.
func NewEvalOptions(js string, args JSArgs) *EvalOptions {
	return &EvalOptions{true, "", js, args, false}
}

const jsHelperID = proto.RuntimeRemoteObjectID("rodJSHelper")

// Convert name and jsArgs to Page.Eval, the name is method name in the "lib/assets/helper.js".
func jsHelper(name js.Name, args JSArgs) *EvalOptions {
	return &EvalOptions{
		JSArgs: append(JSArgs{jsHelperID}, args...),
		JS:     fmt.Sprintf(`(rod, ...args) => rod.%s.apply(this, args)`, name),
	}
}

// SprintFnThis wrap js with this, wrap function call if it's js expression
func SprintFnThis(js string) string {
	if detectJSFunction(js) {
		return fmt.Sprintf(`function() { return (%s).apply(this, arguments) }`, js)
	}
	return fmt.Sprintf(`function() { return %s }`, js)
}

var _ io.Reader = &StreamReader{}

// StreamReader for browser data stream
type StreamReader struct {
	Offset int64

	c      proto.Caller
	handle proto.IOStreamHandle
	buf    *bytes.Buffer
}

// NewStreamReader instance
func NewStreamReader(c proto.Caller, h proto.IOStreamHandle) *StreamReader {
	return &StreamReader{
		c:      c,
		handle: h,
		buf:    &bytes.Buffer{},
	}
}

func (sr *StreamReader) Read(p []byte) (n int, err error) {
	res, err := proto.IORead{
		Handle: sr.handle,
		Offset: sr.Offset,
	}.Call(sr.c)
	if err != nil {
		return 0, err
	}

	if !res.EOF {
		var bin []byte
		if res.Base64Encoded {
			bin, err = base64.StdEncoding.DecodeString(res.Data)
			if err != nil {
				return 0, err
			}
		} else {
			bin = []byte(res.Data)
		}

		_, _ = sr.buf.Write(bin)
	}

	return sr.buf.Read(p)
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
				err = newErr(ErrValue, val, utils.MustToJSON(val))
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

func genRegFilter(includes, excludes []string) func(string) bool {
	regIncludes := make([]*regexp.Regexp, len(includes))
	for i, p := range includes {
		regIncludes[i] = regexp.MustCompile(p)
	}

	regExcludes := make([]*regexp.Regexp, len(excludes))
	for i, p := range excludes {
		regExcludes[i] = regexp.MustCompile(p)
	}

	return func(s string) bool {
		for _, include := range regIncludes {
			if include.MatchString(s) {
				for _, exclude := range regExcludes {
					if exclude.MatchString(s) {
						goto end
					}
				}
				return true
			}
		}
	end:
		return false
	}
}

type saveFileType int

const (
	saveFileTypeScreenshot saveFileType = iota
	saveFileTypePDF
)

func saveFile(fileType saveFileType, bin []byte, toFile []string) error {
	if len(toFile) == 0 {
		return nil
	}
	if toFile[0] == "" {
		stamp := fmt.Sprintf("%d", time.Now().UnixNano())
		switch fileType {
		case saveFileTypeScreenshot:
			toFile = []string{"tmp", "screenshots", stamp + ".png"}
		case saveFileTypePDF:
			toFile = []string{"tmp", "pdf", stamp + ".pdf"}
		}
	}
	return utils.OutputFile(filepath.Join(toFile...), bin)
}

func httHTML(w http.ResponseWriter, body string) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func mustToJSONForDev(value interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	utils.E(enc.Encode(value))

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
