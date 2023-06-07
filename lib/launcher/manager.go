package launcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/utils"
)

const (
	// HeaderName for remote launch
	HeaderName = "Rod-Launcher"
)

// MustNewManaged is similar to NewManaged
func MustNewManaged(serviceURL string) *Launcher {
	l, err := NewManaged(serviceURL)
	utils.E(err)

	// TODO: remove this after we have a better way to handle this
	// The latest chromium in docker will crash pages that use http2
	l.Set("disable-http2")

	return l
}

// NewManaged creates a default Launcher instance from launcher.Manager.
// The serviceURL must point to a launcher.Manager. It will send a http request to the serviceURL
// to get the default settings of the Launcher instance. For example if the launcher.Manager running on a
// Linux machine will return different default settings from the one on Mac.
// If Launcher.Leakless is enabled, the remote browser will be killed after the websocket is closed.
func NewManaged(serviceURL string) (*Launcher, error) {
	if serviceURL == "" {
		serviceURL = "ws://127.0.0.1:7317"
	}

	u, err := url.Parse(serviceURL)
	if err != nil {
		return nil, err
	}

	l := New()
	l.managed = true
	l.serviceURL = toWS(*u).String()
	l.Flags = nil

	res, err := http.Get(toHTTP(*u).String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	return l, json.NewDecoder(res.Body).Decode(l)
}

// KeepUserDataDir after remote browser is closed. By default launcher.FlagUserDataDir will be removed.
func (l *Launcher) KeepUserDataDir() *Launcher {
	l.mustManaged()
	l.Set(flags.KeepUserDataDir)
	return l
}

// JSON serialization
func (l *Launcher) JSON() []byte {
	return utils.MustToJSONBytes(l)
}

// MustClient similar to Launcher.Client
func (l *Launcher) MustClient() *cdp.Client {
	u, h := l.ClientHeader()
	return cdp.MustStartWithURL(l.ctx, u, h)
}

// Client for launching browser remotely via the launcher.Manager.
func (l *Launcher) Client() (*cdp.Client, error) {
	u, h := l.ClientHeader()
	return cdp.StartWithURL(l.ctx, u, h)
}

// ClientHeader for launching browser remotely via the launcher.Manager.
func (l *Launcher) ClientHeader() (string, http.Header) {
	l.mustManaged()
	header := http.Header{}
	header.Add(string(HeaderName), utils.MustToJSON(l))
	return l.serviceURL, header
}

func (l *Launcher) mustManaged() {
	if !l.managed {
		panic("Must be used with launcher.NewManaged")
	}
}

var _ http.Handler = &Manager{}

// Manager is used to launch browsers via http server on another machine.
// The reason why we have Manager is after we launcher a browser, we can't dynamically change its
// CLI arguments, such as "--headless". The Manager allows us to decide what CLI arguments to
// pass to the browser when launch it remotely.
// The work flow looks like:
//
//	|      Machine X       |                             Machine Y                                    |
//	| NewManaged("a.com") -|-> http.ListenAndServe("a.com", launcher.NewManager()) --> launch browser |
//
//	1. X send a http request to Y, Y respond default Launcher settings based the OS of Y.
//	2. X start a websocket connect to Y with the Launcher settings
//	3. Y launches a browser with the Launcher settings X
//	4. Y transparently proxy the websocket connect between X and the launched browser
type Manager struct {
	// Logger for key events
	Logger utils.Logger

	// Defaults should return the default Launcher settings
	Defaults func(http.ResponseWriter, *http.Request) *Launcher

	// BeforeLaunch hook is called right before the launching with the Launcher instance that will be used
	// to launch the browser.
	// Such as use it to filter malicious values of Launcher.UserDataDir, Launcher.Bin, or Launcher.WorkingDir.
	BeforeLaunch func(*Launcher, http.ResponseWriter, *http.Request)
}

// NewManager instance
func NewManager() *Manager {
	allowedPath := map[flags.Flag]string{
		flags.Bin: DefaultBrowserDir,
		flags.WorkingDir: func() string {
			p, _ := os.Getwd()
			return p
		}(),
		flags.UserDataDir: DefaultUserDataDirPrefix,
	}

	return &Manager{
		Logger:   utils.LoggerQuiet,
		Defaults: func(_ http.ResponseWriter, _ *http.Request) *Launcher { return New() },
		BeforeLaunch: func(l *Launcher, w http.ResponseWriter, r *http.Request) {
			for f, allowed := range allowedPath {
				p := l.Get(f)
				if p != "" && !strings.HasPrefix(p, allowed) {
					b := []byte(fmt.Sprintf("[rod-manager] not allowed %s path: %s (use --allow-all to disable the protection)", f, p))
					w.Header().Add("Content-Length", fmt.Sprintf("%d", len(b)))
					w.WriteHeader(http.StatusBadRequest)
					utils.E(w.Write(b))
					w.(http.Flusher).Flush()
					panic(http.ErrAbortHandler)
				}
			}
		},
	}
}

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		m.launch(w, r)
		return
	}

	l := m.Defaults(w, r)
	utils.E(w.Write(l.JSON()))
}

func (m *Manager) launch(w http.ResponseWriter, r *http.Request) {
	l := New()

	options := r.Header.Get(string(HeaderName))
	if options != "" {
		l.Flags = nil
		utils.E(json.Unmarshal([]byte(options), l))
	}

	m.BeforeLaunch(l, w, r)

	kill := l.Has(flags.Leakless)

	// Always enable leakless so that if the Manager process crashes
	// all the managed browsers will be killed.
	u := l.Leakless(true).MustLaunch()
	defer m.cleanup(l, kill)

	parsedURL, err := url.Parse(u)
	utils.E(err)

	m.Logger.Println("Launch", u, options)
	defer m.Logger.Println("Close", u)

	parsedWS, err := url.Parse(u)
	utils.E(err)
	parsedURL.Path = parsedWS.Path

	httputil.NewSingleHostReverseProxy(toHTTP(*parsedURL)).ServeHTTP(w, r)
}

func (m *Manager) cleanup(l *Launcher, kill bool) {
	if kill {
		l.Kill()
		m.Logger.Println("Killed PID:", l.PID())
	}

	if !l.Has(flags.KeepUserDataDir) {
		l.Cleanup()
		dir := l.Get(flags.UserDataDir)
		m.Logger.Println("Removed", dir)
	}
}
