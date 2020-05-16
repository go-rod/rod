package launcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// HeaderName for remote launch
const HeaderName = "Rod-Launcher"

// NewRemote create a Launcher instance from remote defaults
func NewRemote(remoteURL string) *Launcher {
	u, err := url.Parse(remoteURL)
	kit.E(err)

	toHTTP(u)

	l := New()
	l.url = remoteURL
	l.Flags = nil

	kit.E(json.Unmarshal(kit.Req(u.String()).MustBytes(), l))

	return l
}

// JSON serialization
func (l *Launcher) JSON() []byte {
	return kit.MustToJSONBytes(l)
}

// Client for launching chrome remotely
func (l *Launcher) Client() *cdp.Client {
	header := http.Header{}
	header.Add(HeaderName, kit.MustToJSON(l))
	return cdp.New(l.url).Header(header)
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

func (p *Proxy) defaults(w http.ResponseWriter, _ *http.Request) {
	l := New()
	kit.E(w.Write(l.JSON()))
}

func (p *Proxy) launch(w http.ResponseWriter, r *http.Request) {
	l := New().Log(p.Log)

	options := r.Header.Get(HeaderName)
	if options != "" {
		l.Flags = nil
		kit.E(json.Unmarshal([]byte(options), l))
	}

	u := l.Launch()
	defer l.kill()

	parsedURL, err := url.Parse(u)
	kit.E(err)

	if p.Log != nil {
		p.Log(fmt.Sprintln(kit.C("Launch", "cyan"), u, l.FormatArgs()))
		defer func() { p.Log(fmt.Sprintln(kit.C("Close", "cyan"), u)) }()
	}

	parsedWS, err := url.Parse(u)
	kit.E(err)
	parsedURL.Path = parsedWS.Path
	toHTTP(parsedURL)

	httputil.NewSingleHostReverseProxy(parsedURL).ServeHTTP(w, r)
}
