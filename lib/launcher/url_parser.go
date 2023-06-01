package launcher

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

var _ io.Writer = &URLParser{}

// URLParser to get control url from stderr
type URLParser struct {
	URL    chan string
	Buffer string // buffer for the browser stdout

	lock *sync.Mutex
	ctx  context.Context
	done bool
}

// NewURLParser instance
func NewURLParser() *URLParser {
	return &URLParser{
		URL:  make(chan string),
		lock: &sync.Mutex{},
		ctx:  context.Background(),
	}
}

var regWS = regexp.MustCompile(`ws://.+/`)

// Context sets the context
func (r *URLParser) Context(ctx context.Context) *URLParser {
	r.ctx = ctx
	return r
}

// Write interface
func (r *URLParser) Write(p []byte) (n int, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.done {
		r.Buffer += string(p)

		str := regWS.FindString(r.Buffer)
		if str != "" {
			u, err := url.Parse(strings.TrimSpace(str))
			utils.E(err)

			select {
			case <-r.ctx.Done():
			case r.URL <- "http://" + u.Host:
			}

			r.done = true
			r.Buffer = ""
		}
	}

	return len(p), nil
}

// Err returns the common error parsed from stdout and stderr
func (r *URLParser) Err() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	msg := "[launcher] Failed to get the debug url: "

	if strings.Contains(r.Buffer, "error while loading shared libraries") {
		msg = "[launcher] Failed to launch the browser, the doc might help https://go-rod.github.io/#/compatibility?id=os: "
	}

	return errors.New(msg + r.Buffer)
}

// MustResolveURL is similar to ResolveURL
func MustResolveURL(u string) string {
	u, err := ResolveURL(u)
	utils.E(err)
	return u
}

var (
	regPort     = regexp.MustCompile(`^\:?(\d+)$`)
	regProtocol = regexp.MustCompile(`^\w+://`)
)

// ResolveURL by requesting the u, it will try best to normalize the u.
// The format of u can be "9222", ":9222", "host:9222", "ws://host:9222", "wss://host:9222",
// "https://host:9222" "http://host:9222". The return string will look like:
// "ws://host:9222/devtools/browser/4371405f-84df-4ad6-9e0f-eab81f7521cc"
func ResolveURL(u string) (string, error) {
	if u == "" {
		u = "9222"
	}

	u = strings.TrimSpace(u)
	u = regPort.ReplaceAllString(u, "127.0.0.1:$1")

	if !regProtocol.MatchString(u) {
		u = "http://" + u
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	parsed = toHTTP(*parsed)
	parsed.Path = "/json/version"

	res, err := http.Get(parsed.String())
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	data, err := ioutil.ReadAll(res.Body)
	utils.E(err)

	return gson.New(data).Get("webSocketDebuggerUrl").Str(), nil
}
