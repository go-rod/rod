package launcher

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-rod/rod/lib/utils"
)

var _ io.Writer = &URLParser{}

// URLParser to get control url from stderr
type URLParser struct {
	URL chan string

	done bool
	buf  string
}

// NewURLParser instance
func NewURLParser() *URLParser {
	return &URLParser{
		URL: make(chan string),
	}
}

var regWS = regexp.MustCompile(`ws://.+/`)

// Write interface
func (r *URLParser) Write(p []byte) (n int, err error) {
	if !r.done {
		r.buf += string(p)

		str := regWS.FindString(r.buf)
		if str != "" {
			u, err := url.Parse(strings.TrimSpace(str))
			utils.E(err)

			r.URL <- "http://" + u.Host
			r.done = true
			r.buf = ""
		}
	}

	return len(p), nil
}

// GetWebSocketDebuggerURL from browser remote url
func GetWebSocketDebuggerURL(u string) (string, error) {
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

	return utils.ReadJSONPathAsString(res.Body, "webSocketDebuggerUrl")
}
