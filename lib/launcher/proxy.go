package launcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"

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

// KeepUserDataDir after remote browser is closed. By default user-data-dir will be removed.
func (l *Launcher) KeepUserDataDir() *Launcher {
	l.Set("keep-user-data-dir")
	return l
}

// JSON serialization
func (l *Launcher) JSON() []byte {
	return kit.MustToJSONBytes(l)
}

// Client for launching browser remotely
func (l *Launcher) Client() *cdp.Client {
	header := http.Header{}
	header.Add(HeaderName, kit.MustToJSON(l))
	return cdp.New(l.url).Header(header)
}

// Proxy to help launch browser remotely.
// Any http request will return a default Launcher based on remote OS environment.
// Any websocket request will start a new browser and the request will be proxied to the browser.
// The websocket header "Rod-Launcher" holds the options to launch browser.
// If the websocket is closed, the browser will be killed.
type Proxy struct {
	Log func(string)
}

var _ http.Handler = &Proxy{}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		p.launch(w, r)
		return
	}

	p.defaults(w, r)
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
	defer func() {
		proc, err := os.FindProcess(l.PID())
		l.kill()
		if err == nil {
			_, _ = proc.Wait()
		}

		// TODO: Seems like a delay bug for windows chrome?
		if runtime.GOOS == "windows" {
			kit.Sleep(0.1)
		}

		if _, has := l.Get("keep-user-data-dir"); !has {
			dir, _ := l.Get("user-data-dir")
			if p.Log != nil {
				p.Log(fmt.Sprintln(kit.C("Remove", "cyan"), dir))
			}

			_ = os.RemoveAll(dir)
		}
	}()

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
