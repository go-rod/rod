package launcher

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/utils"
)

// HeaderName for remote launch
const HeaderName = "Rod-Launcher"

const flagKeepUserDataDir = "rod-keep-user-data-dir"

// MustNewManaged is similar to MustNewManaged
func MustNewManaged(serviceURL string) *Launcher {
	l, err := NewManaged(serviceURL)
	utils.E(err)
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

	return l, json.NewDecoder(res.Body).Decode(l)
}

// KeepUserDataDir after remote browser is closed. By default user-data-dir will be removed.
func (l *Launcher) KeepUserDataDir() *Launcher {
	l.mustManaged()
	l.Set(flagKeepUserDataDir)
	return l
}

// JSON serialization
func (l *Launcher) JSON() []byte {
	return utils.MustToJSONBytes(l)
}

// Client for launching browser remotely, such as browser from a docker container.
func (l *Launcher) Client() *cdp.Client {
	l.mustManaged()
	header := http.Header{}
	header.Add(HeaderName, utils.MustToJSON(l))
	return cdp.New(l.serviceURL).Header(header)
}

func (l *Launcher) mustManaged() {
	if !l.managed {
		panic("Must be used with launcher.NewManaged")
	}
}

var _ http.Handler = &Manager{}

// Manager is used to launch browsers via http server on another machine.
// The reason why we have Manager is after we launcher a browser, we can't dynamicall change its
// CLI arguments, such as "--headless". The Manager allows us to decide what CLI arguments to
// pass to the browser when launch it remotely.
// The work flow looks like:
//
//     |      Machine X       |                             Machine Y                                    |
//     | NewManaged("a.com") -|-> http.ListenAndServe("a.com", launcher.NewManager()) --> launch browser |
//
//     1. X send a http request to Y, Y respond default Launcher settings based the OS of Y.
//     2. X start a websocket connect to Y with the Launcher settings
//     3. Y launches a browser with the Launcher settings X
//     4. Y transparently proxy the websocket connect between X and the launched browser
//
type Manager struct {
	Logger   utils.Logger
	Defaults func() *Launcher
}

// NewManager instance
func NewManager() *Manager {
	return &Manager{
		Logger:   utils.LoggerQuiet,
		Defaults: New,
	}
}

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		m.launch(w, r)
		return
	}

	m.defaults(w, r)
}

func (m *Manager) defaults(w http.ResponseWriter, _ *http.Request) {
	l := New()
	utils.E(w.Write(l.JSON()))
}

func (m *Manager) launch(w http.ResponseWriter, r *http.Request) {
	l := m.Defaults()

	options := r.Header.Get(HeaderName)
	if options != "" {
		l.Flags = nil
		utils.E(json.Unmarshal([]byte(options), l))
	}

	kill := l.Has(flagLeakless)

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

	if !l.Has(flagKeepUserDataDir) {
		l.Cleanup()
		dir, _ := l.Get("user-data-dir")
		m.Logger.Println("Removed", dir)
	}
}
