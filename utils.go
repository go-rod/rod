package rod

import (
	"bytes"
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

func mustToJSONForDev(value interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	kit.E(enc.Encode(value))

	return buf.String()
}
