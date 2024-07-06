package rod

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// CDPClient is usually used to make rod side-effect free. Such as proxy all IO of rod.
type CDPClient interface {
	Event() <-chan *cdp.Event
	Call(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error)
}

// Message represents a cdp.Event.
type Message struct {
	SessionID proto.TargetSessionID
	Method    string

	lock  *sync.Mutex
	data  json.RawMessage
	event reflect.Value
}

// Load data into e, returns true if e matches the event type.
func (msg *Message) Load(e proto.Event) bool {
	if msg.Method != e.ProtoEvent() {
		return false
	}

	eVal := reflect.ValueOf(e)
	if eVal.Kind() != reflect.Ptr {
		return true
	}
	eVal = reflect.Indirect(eVal)

	msg.lock.Lock()
	defer msg.lock.Unlock()
	if msg.data == nil {
		eVal.Set(msg.event)
		return true
	}

	utils.E(json.Unmarshal(msg.data, e))
	msg.event = eVal
	msg.data = nil
	return true
}

// DefaultLogger for rod.
var DefaultLogger = log.New(os.Stdout, "[rod] ", log.LstdFlags)

// DefaultSleeper generates the default sleeper for retry, it uses backoff to grow the interval.
// The growth looks like:
//
//	A(0) = 100ms, A(n) = A(n-1) * random[1.9, 2.1), A(n) < 1s
//
// Why the default is not RequestAnimationFrame or DOM change events is because of if a retry never
// ends it can easily flood the program. But you can always easily config it into what you want.
var DefaultSleeper = func() utils.Sleeper {
	return utils.BackoffSleeper(100*time.Millisecond, time.Second, nil)
}

// NewPagePool instance.
func NewPagePool(limit int) Pool[Page] {
	return NewPool[Page](limit)
}

// NewBrowserPool instance.
func NewBrowserPool(limit int) Pool[Browser] {
	return NewPool[Browser](limit)
}

// Pool is used to thread-safely limit the number of elements at the same time.
// It's a common practice to use a channel to limit concurrency, it's not special for rod.
// This helper is more like an example to use Go Channel.
// Reference: https://golang.org/doc/effective_go#channels
type Pool[T any] chan *T

// NewPool instance.
func NewPool[T any](limit int) Pool[T] {
	p := make(chan *T, limit)
	for i := 0; i < limit; i++ {
		p <- nil
	}
	return p
}

// Get a elem from the pool, allow error. Use the [Pool[T].Put] to make it reusable later.
func (p Pool[T]) Get(create func() (*T, error)) (elem *T, err error) {
	elem = <-p
	if elem == nil {
		elem, err = create()
	}
	return
}

// Put an elem back to the pool.
func (p Pool[T]) Put(elem *T) {
	p <- elem
}

// Cleanup helper.
func (p Pool[T]) Cleanup(iteratee func(*T)) {
	for i := 0; i < cap(p); i++ {
		select {
		case elem := <-p:
			if elem != nil {
				iteratee(elem)
			}
		default:
		}
	}
}

var _ io.ReadCloser = &StreamReader{}

// StreamReader for browser data stream.
type StreamReader struct {
	Offset *int

	c      proto.Client
	handle proto.IOStreamHandle
	buf    *bytes.Buffer
}

// NewStreamReader instance.
func NewStreamReader(c proto.Client, h proto.IOStreamHandle) *StreamReader {
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

// Close the stream, discard any temporary backing storage.
func (sr *StreamReader) Close() error {
	return proto.IOClose{Handle: sr.handle}.Call(sr.c)
}

// Try try fn with recover, return the panic as rod.ErrTry.
func Try(fn func()) (err error) {
	defer func() {
		if val := recover(); val != nil {
			err = &TryError{val, string(debug.Stack())}
		}
	}()

	fn()

	return err
}

func genRegMatcher(includes, excludes []string) func(string) bool {
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

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/Data_URIs
var regDataURI = regexp.MustCompile(`\Adata:(.+?)?(;base64)?,`)

func parseDataURI(uri string) (string, []byte) {
	matches := regDataURI.FindStringSubmatch(uri)
	l := len(matches[0])
	contentType := matches[1]

	bin, _ := base64.StdEncoding.DecodeString(uri[l:])
	return contentType, bin
}
