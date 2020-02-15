package launcher

import (
	"context"
	"errors"
	"io"
	nurl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/leakless"
)

// Launcher chrome cli flags helper
type Launcher struct {
	bin   string
	flags map[string][]string
}

// New returns the default arguments to start chrome.
// "--" is optional, with or without it won't affect the result.
// All available switches: https://chromium.googlesource.com/chromium/src/+/32352ad08ee673a4d43e8593ce988b224f6482d3/chrome/common/chrome_switches.cc
func New() *Launcher {
	defaultFlags := map[string][]string{
		// use random port by default
		"remote-debugging-port": {"0"},

		// enable headless by default
		"headless": nil,

		// disable site-per-process to make sure iframes are not detached automatically
		"disable-features": {"site-per-process", "TranslateUI"},

		"disable-background-networking":                      nil,
		"enable-features":                                    {"NetworkService", "NetworkServiceInProcess"},
		"disable-background-timer-throttling":                nil,
		"disable-backgrounding-occluded-windows":             nil,
		"disable-breakpad":                                   nil,
		"disable-client-side-phishing-detection":             nil,
		"disable-component-extensions-with-background-pages": nil,
		"disable-default-apps":                               nil,
		"disable-dev-shm-usage":                              nil,
		"disable-extensions":                                 nil,
		"disable-hang-monitor":                               nil,
		"disable-ipc-flooding-protection":                    nil,
		"disable-popup-blocking":                             nil,
		"disable-prompt-on-repost":                           nil,
		"disable-renderer-backgrounding":                     nil,
		"disable-sync":                                       nil,
		"force-color-profile":                                {"srgb"},
		"metrics-recording-only":                             nil,
		"no-first-run":                                       nil,
		"enable-automation":                                  nil,
		"password-store":                                     {"basic"},
		"use-mock-keychain":                                  nil,

		// to prevent welcome page
		"about:blank": nil,
	}

	return &Launcher{
		flags: defaultFlags,
	}
}

// Has flag
func (l *Launcher) Has(name string) bool {
	_, has := l.flags[name]
	return has
}

// Set flag
func (l *Launcher) Set(name string, values ...string) *Launcher {
	l.flags[strings.TrimLeft(name, "-")] = values
	return l
}

// Delete flag
func (l *Launcher) Delete(name string) *Launcher {
	delete(l.flags, strings.TrimLeft(name, "-"))
	return l
}

// Bin set chrome executable file path
func (l *Launcher) Bin(path string) *Launcher {
	l.bin = path
	return l
}

// Headless switch
func (l *Launcher) Headless(enable bool) *Launcher {
	if enable {
		l.Set("headless")
	} else {
		l.Delete("headless")
	}
	return l
}

// UserDataDir arg
func (l *Launcher) UserDataDir(dir string) *Launcher {
	l.Set("user-data-dir", dir)
	return l
}

// RemoteDebuggingPort arg
func (l *Launcher) RemoteDebuggingPort(port string) *Launcher {
	l.Set("remote-debugging-port", port)
	return l
}

// ExecFormat returns the formated arg list for cli
func (l *Launcher) ExecFormat() []string {
	execArgs := []string{}
	for k, v := range l.flags {
		str := "--" + k
		if v != nil {
			str += "=" + strings.Join(v, ",")
		}
		execArgs = append(execArgs, str)
	}
	return execArgs
}

// Launch a standalone temp browser instance and returns the debug url.
// bin and profileDir are optional, set them to empty to use the default values.
// If you want to reuse sessions, such as cookies, set the userDataDir to the same location.
func (l *Launcher) Launch() string {
	u, err := l.LaunchE()
	kit.E(err)
	return u
}

// LaunchE ...
func (l *Launcher) LaunchE() (string, error) {
	bin := l.bin
	if bin == "" {
		var err error
		chrome := NewChrome()
		bin, err = chrome.Get()
		if err != nil {
			return "", err
		}
	}

	if !l.Has("user-data-dir") {
		tmp := filepath.Join(os.TempDir(), "cdp", kit.RandString(8))
		err := os.MkdirAll(tmp, 0700)
		if err != nil {
			return "", err
		}
		l.UserDataDir(tmp)
	}

	cmd := leakless.New().Command(
		bin,
		l.ExecFormat()...,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	u, err := readURL(stderr)
	if err != nil {
		_ = cmd.Process.Kill()
		return "", err
	}

	return u, nil
}

func readURL(stderr io.ReadCloser) (string, error) {
	buf := make([]byte, 100)
	str := ""
	out := ""
	wait := make(chan kit.Nil)

	read := func() {
		defer close(wait)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				break
			}
			out += string(buf[:n])

			str = regexp.MustCompile(`ws://.+/`).FindString(out)
			if str != "" {
				break
			}
		}
	}

	timeout, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	go read()

	select {
	case <-timeout.Done():
		return "", errors.New("[rod/lib/launcher] launch chrome timeout: " + out)
	case <-wait:
	}

	u, err := nurl.Parse(strings.TrimSpace(str))
	if err != nil {
		return "", errors.New("[rod/lib/launcher] failed to get control url: " + out + " " + err.Error())
	}

	return "http://" + u.Host, nil
}

// GetWebSocketDebuggerURL ...
func GetWebSocketDebuggerURL(url string) (string, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return "", err
	}

	if u.Scheme == "ws" {
		u.Scheme = "http"
	}
	if u.Scheme == "wss" {
		u.Scheme = "https"
	}

	u.Path = "/json/version"

	obj, err := kit.Req(u.String()).JSON()
	if err != nil {
		return "", err
	}
	return obj.Get("webSocketDebuggerUrl").String(), nil
}
