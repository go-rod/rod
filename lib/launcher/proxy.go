package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ysmood/kit"
)

// HeaderName for remote launch
const HeaderName = "Rod-Launcher"

// NewRemote create a Launcher instance from remote defaults
func NewRemote(remoteURL string) *Launcher {
	u, err := url.Parse(remoteURL)
	kit.E(err)

	if u.Scheme == "ws" {
		u.Scheme = "http"
	}
	if u.Scheme == "wss" {
		u.Scheme = "https"
	}

	l := New()
	l.Flags = nil
	kit.E(json.Unmarshal(kit.Req(u.String()).MustBytes(), l))
	return l
}

// JSON serialization
func (l *Launcher) JSON() []byte {
	return kit.MustToJSONBytes(l)
}

// Header for launching chrome remotely
func (l *Launcher) Header() http.Header {
	header := http.Header{}
	header.Add(HeaderName, kit.MustToJSON(l))
	return header
}

// Proxy to help launch chrome remotely.
// Any http request will return a default Launcher based on remote OS environment.
// Any websocket request will start a new chrome and the request will be proxied to the chrome.
// The websocket header "Rod-Launcher" holds the options to launch chrome.
// If the websocket is closed, the chrome will be killed.
type Proxy struct {
	Log func(string)
}

var _ http.Handler = &Proxy{}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
		p.defaults(w, r)
		return
	}

	p.launch(w, r)
}

func (p *Proxy) defaults(w http.ResponseWriter, r *http.Request) {
	l := New()
	kit.E(w.Write(l.JSON()))
}

func (p *Proxy) launch(w http.ResponseWriter, r *http.Request) {
	l := New().Log(p.Log)
	l.Flags = nil
	kit.E(json.Unmarshal([]byte(r.Header.Get(HeaderName)), l))

	u := l.Launch()
	defer l.kill()

	parsedURL, err := url.Parse(u)
	kit.E(err)

	wsURL, err := GetWebSocketDebuggerURL(context.Background(), u)
	kit.E(err)

	if p.Log != nil {
		p.Log(fmt.Sprintln("launch:", wsURL, l.FormatArgs()))
		defer func() { p.Log(fmt.Sprintln("close:", wsURL)) }()
	}

	parsedWS, err := url.Parse(wsURL)
	kit.E(err)
	parsedURL.Path = parsedWS.Path

	httputil.NewSingleHostReverseProxy(parsedURL).ServeHTTP(w, r)
}
