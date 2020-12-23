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

// MustNewRemote is similar to NewRemote
func MustNewRemote(remoteURL string) *Launcher {
	l, err := NewRemote(remoteURL)
	utils.E(err)
	return l
}

// NewRemote creates a Launcher instance from remote defaults.
// For more info check the doc of RemoteLauncher.
func NewRemote(remoteURL string) (*Launcher, error) {
	u, err := url.Parse(remoteURL)
	if err != nil {
		return nil, err
	}

	l := New()
	l.remote = true
	l.url = toWS(*u).String()
	l.Flags = nil

	res, err := http.Get(toHTTP(*u).String())
	if err != nil {
		return nil, err
	}

	return l, json.NewDecoder(res.Body).Decode(l)
}

// KeepUserDataDir after remote browser is closed. By default user-data-dir will be removed.
func (l *Launcher) KeepUserDataDir() *Launcher {
	l.mustRemote()
	l.Set(flagKeepUserDataDir)
	return l
}

// JSON serialization
func (l *Launcher) JSON() []byte {
	return utils.MustToJSONBytes(l)
}

// Client for launching browser remotely, such as browser from a docker container.
func (l *Launcher) Client() *cdp.Client {
	l.mustRemote()
	header := http.Header{}
	header.Add(HeaderName, utils.MustToJSON(l))
	return cdp.New(l.url).Header(header)
}

func (l *Launcher) mustRemote() {
	if !l.remote {
		panic("Must be used with launcher.NewRemote")
	}
}

var _ http.Handler = &RemoteLauncher{}

// RemoteLauncher is used to launch browsers via http server on another machine.
// For example, the work flow looks like:
//
// 	|     Machine A      |                           Machine B                                  |
// 	 NewRemote("a.com") --> http.ListenAndServe("a.com", NewRemoteLauncher()) --> launch browser
//
// Any http request will return a default Launcher based on remote OS environment.
// Any websocket request will start a new browser and the request will be proxied to the browser.
// The websocket header "Rod-Launcher" holds the options to launch browser.
// If the websocket is closed, the browser will be killed.
type RemoteLauncher struct {
	Logger utils.Logger
}

// NewRemoteLauncher instance
func NewRemoteLauncher() *RemoteLauncher {
	return &RemoteLauncher{
		Logger: utils.LoggerQuiet,
	}
}

func (p *RemoteLauncher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		p.launch(w, r)
		return
	}

	p.defaults(w, r)
}

func (p *RemoteLauncher) defaults(w http.ResponseWriter, _ *http.Request) {
	l := New()
	utils.E(w.Write(l.JSON()))
}

func (p *RemoteLauncher) launch(w http.ResponseWriter, r *http.Request) {
	l := New()

	options := r.Header.Get(HeaderName)
	if options != "" {
		l.Flags = nil
		utils.E(json.Unmarshal([]byte(options), l))
	}

	u := l.Leakless(false).MustLaunch()
	defer func() {
		l.Kill()
		p.Logger.Println("Killed PID:", l.PID())

		if _, has := l.Get(flagKeepUserDataDir); !has {
			l.Cleanup()
			dir, _ := l.Get("user-data-dir")
			p.Logger.Println("Removed", dir)
		}
	}()

	parsedURL, err := url.Parse(u)
	utils.E(err)

	p.Logger.Println("Launch", u, options)
	defer p.Logger.Println("Close", u)

	parsedWS, err := url.Parse(u)
	utils.E(err)
	parsedURL.Path = parsedWS.Path

	httputil.NewSingleHostReverseProxy(toHTTP(*parsedURL)).ServeHTTP(w, r)
}
