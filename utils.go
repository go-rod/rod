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

// Message represents a cdp.Event
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

// DefaultLogger for rod
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

// PagePool to thread-safely limit the number of pages at the same time.
// It's a common practice to use a channel to limit concurrency, it's not special for rod.
// This helper is more like an example to use Go Channel.
// Reference: https://golang.org/doc/effective_go#channels
type PagePool chan *Page

// NewPagePool instance
func NewPagePool(limit int) PagePool {
	pp := make(chan *Page, limit)
	for i := 0; i < limit; i++ {
		pp <- nil
	}
	return pp
}

// Get a page from the pool. Use the PagePool.Put to make it reusable later.
func (pp PagePool) Get(create func() *Page) *Page {
	p := <-pp
	if p == nil {
		p = create()
	}
	return p
}

// Put a page back to the pool
func (pp PagePool) Put(p *Page) {
	pp <- p
}

// Cleanup helper
func (pp PagePool) Cleanup(iteratee func(*Page)) {
	for i := 0; i < cap(pp); i++ {
		p := <-pp
		if p != nil {
			iteratee(p)
		}
	}
}

// BrowserPool to thread-safely limit the number of browsers at the same time.
// It's a common practice to use a channel to limit concurrency, it's not special for rod.
// This helper is more like an example to use Go Channel.
// Reference: https://golang.org/doc/effective_go#channels
type BrowserPool chan *Browser

// NewBrowserPool instance
func NewBrowserPool(limit int) BrowserPool {
	pp := make(chan *Browser, limit)
	for i := 0; i < limit; i++ {
		pp <- nil
	}
	return pp
}

// Get a browser from the pool. Use the BrowserPool.Put to make it reusable later.
func (bp BrowserPool) Get(create func() *Browser) *Browser {
	p := <-bp
	if p == nil {
		p = create()
	}
	return p
}

// Put a browser back to the pool
func (bp BrowserPool) Put(p *Browser) {
	bp <- p
}

// Cleanup helper
func (bp BrowserPool) Cleanup(iteratee func(*Browser)) {
	for i := 0; i < cap(bp); i++ {
		p := <-bp
		if p != nil {
			iteratee(p)
		}
	}
}

var _ io.ReadCloser = &StreamReader{}

// StreamReader for browser data stream
type StreamReader struct {
	Offset *int

	c      proto.Client
	handle proto.IOStreamHandle
	buf    *bytes.Buffer
}

// NewStreamReader instance
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

// Try try fn with recover, return the panic as rod.ErrTry
func Try(fn func()) (err error) {
	defer func() {
		if val := recover(); val != nil {
			err = &ErrTry{val, string(debug.Stack())}
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
