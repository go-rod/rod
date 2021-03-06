// Package launcher for launching browser utils.
package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/leakless"
)

const (
	flagWorkingDir = "rod-working-dir"
	flagEnv        = "rod-env"
	flagXVFB       = "rod-xvfb"
)

// Launcher is a helper to launch browser binary smartly
type Launcher struct {
	logger    io.Writer
	ctx       context.Context
	ctxCancel func()
	browser   *Browser
	bin       string
	url       string
	parser    *URLParser
	Flags     map[string][]string `json:"flags"`
	pid       int
	exit      chan struct{}
	remote    bool // remote mode or not
	leakless  bool
}

// New returns the default arguments to start browser.
// Headless will be enabled by default.
// Leakless will be enabled by default.
// UserDataDir will use OS tmp dir by default.
func New() *Launcher {
	dir := defaults.Dir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "rod", "user-data", utils.RandString(8))
	}

	defaultFlags := map[string][]string{
		"user-data-dir": {dir},

		// use random port by default
		"remote-debugging-port": {defaults.Port},

		// enable headless by default
		"headless": nil,

		// to disable the init blank window
		"no-first-run":      nil,
		"no-startup-window": nil,

		// TODO: about the "site-per-process" see https://github.com/puppeteer/puppeteer/issues/2548
		"disable-features": {"site-per-process", "TranslateUI"},

		// hide the navigator.webdriver
		"disable-blink-features": {"AutomationControlled"},

		"disable-background-networking":                      nil,
		"disable-background-timer-throttling":                nil,
		"disable-backgrounding-occluded-windows":             nil,
		"disable-breakpad":                                   nil,
		"disable-client-side-phishing-detection":             nil,
		"disable-component-extensions-with-background-pages": nil,
		"disable-default-apps":                               nil,
		"disable-dev-shm-usage":                              nil,
		"disable-hang-monitor":                               nil,
		"disable-ipc-flooding-protection":                    nil,
		"disable-popup-blocking":                             nil,
		"disable-prompt-on-repost":                           nil,
		"disable-renderer-backgrounding":                     nil,
		"disable-sync":                                       nil,
		"enable-automation":                                  nil,
		"enable-features":                                    {"NetworkService", "NetworkServiceInProcess"},
		"force-color-profile":                                {"srgb"},
		"metrics-recording-only":                             nil,
		"use-mock-keychain":                                  nil,
	}

	if defaults.Show {
		delete(defaultFlags, "headless")
	}
	if defaults.Devtools {
		defaultFlags["auto-open-devtools-for-tabs"] = nil
	}
	if inContainer {
		defaultFlags["no-sandbox"] = nil
	}
	if defaults.Proxy != "" {
		defaultFlags["proxy-server"] = []string{defaults.Proxy}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Launcher{
		ctx:       ctx,
		ctxCancel: cancel,
		Flags:     defaultFlags,
		exit:      make(chan struct{}),
		browser:   NewBrowser(),
		bin:       defaults.Bin,
		parser:    NewURLParser(),
		leakless:  true,
		logger:    ioutil.Discard,
	}
}

// NewUserMode is a preset to enable reusing current user data. Useful for automation of personal browser.
// If you see any error, it may because you can't launch debug port for existing browser, the solution is to
// completely close the running browser. Unfortunately, there's no API for rod to tell it automatically yet.
func NewUserMode() *Launcher {
	ctx, cancel := context.WithCancel(context.Background())
	bin, _ := LookPath()

	return &Launcher{
		ctx:       ctx,
		ctxCancel: cancel,
		Flags: map[string][]string{
			"remote-debugging-port": {"37712"},
			"no-startup-window":     nil,
		},
		browser: NewBrowser(),
		exit:    make(chan struct{}),
		bin:     bin,
		parser:  NewURLParser(),
		logger:  ioutil.Discard,
	}
}

// Context set the context
func (l *Launcher) Context(ctx context.Context) *Launcher {
	ctx, cancel := context.WithCancel(ctx)
	l.ctx = ctx
	l.ctxCancel = cancel
	return l
}

// Get flag's first value
func (l *Launcher) Get(name string) (string, bool) {
	list, has := l.GetFlags(name)

	if has {
		if len(list) == 0 {
			return "", true
		}
		return list[0], true
	}
	return "", false
}

// GetFlags from settings
func (l *Launcher) GetFlags(name string) ([]string, bool) {
	flag, has := l.Flags[l.normalizeFlag(name)]
	return flag, has
}

// Set a command line argument to launch the browser. People also call it command line flag or switch.
// List of available flags: https://peter.sh/experiments/chromium-command-line-switches/
func (l *Launcher) Set(name string, values ...string) *Launcher {
	l.Flags[l.normalizeFlag(name)] = values
	return l
}

// Append values to the flag
func (l *Launcher) Append(name string, values ...string) *Launcher {
	flags, has := l.GetFlags(name)
	if !has {
		flags = []string{}
	}
	return l.Set(name, append(flags, values...)...)
}

// Delete a flag
func (l *Launcher) Delete(name string) *Launcher {
	delete(l.Flags, l.normalizeFlag(name))
	return l
}

// Bin set browser executable file path. If it's empty, launcher will automatically search or download the bin.
func (l *Launcher) Bin(path string) *Launcher {
	l.bin = path
	return l
}

// Headless switch. Whether to run browser in headless mode. A mode without visible UI.
func (l *Launcher) Headless(enable bool) *Launcher {
	if enable {
		return l.Set("headless")
	}
	return l.Delete("headless")
}

// NoSandbox switch. Whether to run browser in no-sandbox mode.
// Linux users may face "running as root without --no-sandbox is not supported" in some Linux/Chrome combinations. This function helps switch mode easily.
// Be aware disabling sandbox is not trivial. Use at your own risk.
// Related doc: https://bugs.chromium.org/p/chromium/issues/detail?id=638180
func (l *Launcher) NoSandbox(enable bool) *Launcher {
	if enable {
		return l.Set("no-sandbox")
	}
	return l.Delete("no-sandbox")
}

// XVFB enables to run browser in by XVFB. Useful when you want to run headful mode on linux.
func (l *Launcher) XVFB() *Launcher {
	return l.Set(flagXVFB)
}

// Leakless switch. If enabled, the browser will be force killed after the Go process exits.
// The doc of leakless: https://github.com/ysmood/leakless.
func (l *Launcher) Leakless(enable bool) *Launcher {
	l.leakless = enable
	return l
}

// Devtools switch to auto open devtools for each tab
func (l *Launcher) Devtools(autoOpenForTabs bool) *Launcher {
	if autoOpenForTabs {
		return l.Set("auto-open-devtools-for-tabs")
	}
	return l.Delete("auto-open-devtools-for-tabs")
}

// UserDataDir is where the browser will look for all of its state, such as cookie and cache.
// When set to empty, browser will use current OS home dir.
// Related doc: https://chromium.googlesource.com/chromium/src/+/master/docs/user_data_dir.md
func (l *Launcher) UserDataDir(dir string) *Launcher {
	if dir == "" {
		l.Delete("user-data-dir")
	} else {
		l.Set("user-data-dir", dir)
	}
	return l
}

// ProfileDir is the browser profile the browser will use.
// When set to empty, the profile 'Default' is used.
// Related article: https://superuser.com/a/377195
func (l *Launcher) ProfileDir(dir string) *Launcher {
	if dir == "" {
		l.Delete("profile-directory")
	} else {
		l.Set("profile-directory", dir)
	}
	return l
}

// RemoteDebuggingPort to launch the browser. Zero for a random port. Zero is the default value.
// If it's not zero, the launcher will try to connect to it before starting a new browser process.
// For example, to reuse the same browser process for between 2 runs of a Go program, you can
// do something like:
//     launcher.New().RemoteDebuggingPort(9222).MustLaunch()
//
// Related doc: https://chromedevtools.github.io/devtools-protocol/
func (l *Launcher) RemoteDebuggingPort(port int) *Launcher {
	return l.Set("remote-debugging-port", fmt.Sprintf("%d", port))
}

// Proxy switch. When disabled leakless will be disabled.
func (l *Launcher) Proxy(host string) *Launcher {
	return l.Set("proxy-server", host)
}

// WorkingDir to launch the browser process.
func (l *Launcher) WorkingDir(path string) *Launcher {
	return l.Set(flagWorkingDir, path)
}

// Env to launch the browser process. The default value is os.Environ().
// Usually you use it to set the timezone env. Such as Env("TZ=America/New_York").
// Timezone list: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
func (l *Launcher) Env(env ...string) *Launcher {
	return l.Set(flagEnv, env...)
}

// StartURL to launch
func (l *Launcher) StartURL(u string) *Launcher {
	return l.Set("", u)
}

// FormatArgs returns the formated arg list for cli
func (l *Launcher) FormatArgs() []string {
	execArgs := []string{}
	for k, v := range l.Flags {
		if k == "" {
			continue
		}

		if strings.HasPrefix(k, "rod-") {
			continue
		}

		// fix a bug of chrome, if path is not absolute chrome will hang
		if k == "user-data-dir" {
			abs, err := filepath.Abs(v[0])
			utils.E(err)
			v[0] = abs
		}

		str := "--" + k
		if v != nil {
			str += "=" + strings.Join(v, ",")
		}
		execArgs = append(execArgs, str)
	}
	return append(execArgs, l.Flags[""]...)
}

// Logger to handle stdout and stderr from browser.
// For example, pipe all browser output to stdout: launcher.New().Logger(os.Stdout)
func (l *Launcher) Logger(w io.Writer) *Launcher {
	l.logger = w
	return l
}

// MustLaunch is similar to Launch
func (l *Launcher) MustLaunch() string {
	u, err := l.Launch()
	utils.E(err)
	return u
}

// Launch a standalone temp browser instance and returns the debug url.
// bin and profileDir are optional, set them to empty to use the default values.
// If you want to reuse sessions, such as cookies, set the UserDataDir to the same location.
func (l *Launcher) Launch() (string, error) {
	defer l.ctxCancel()

	bin, err := l.getBin()
	if err != nil {
		return "", err
	}

	var ll *leakless.Launcher
	var cmd *exec.Cmd

	if l.leakless && leakless.Support() {
		ll = leakless.New()
		cmd = ll.Command(bin, l.FormatArgs()...)
	} else {
		port, _ := l.Get("remote-debugging-port")
		u, err := ResolveURL(port)
		if err == nil {
			return u, nil
		}
		cmd = exec.Command(bin, l.FormatArgs()...)
	}

	l.setupCmd(cmd)

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	if ll == nil {
		l.pid = cmd.Process.Pid
	} else {
		l.pid = <-ll.Pid()
		if ll.Err() != "" {
			return "", errors.New(ll.Err())
		}
	}

	go func() {
		_ = cmd.Wait()
		close(l.exit)
	}()

	u, err := l.getURL()
	if err != nil {
		l.Kill()
		return "", err
	}

	return ResolveURL(u)
}

func (l *Launcher) setupCmd(cmd *exec.Cmd) {
	l.osSetupCmd(cmd)

	dir, _ := l.Get(flagWorkingDir)
	env, _ := l.GetFlags(flagEnv)
	cmd.Dir = dir
	cmd.Env = env

	cmd.Stdout = io.MultiWriter(l.logger, l.parser)
	cmd.Stderr = io.MultiWriter(l.logger, l.parser)
}

func (l *Launcher) getBin() (string, error) {
	if l.bin == "" {
		l.browser.Context = l.ctx
		return l.browser.Get()
	}
	return l.bin, nil
}

func (l *Launcher) getURL() (u string, err error) {
	select {
	case <-l.ctx.Done():
		err = l.ctx.Err()
	case u = <-l.parser.URL:
	case <-l.exit:
		l.parser.Lock()
		defer l.parser.Unlock()
		err = l.parser.Err()
	}
	return
}

// PID returns the browser process pid
func (l *Launcher) PID() int {
	return l.pid
}

// Kill the browser process
func (l *Launcher) Kill() {
	// TODO: If kill too fast, the browser's children processes may not be ready.
	// Browser don't have an API to tell if the children processes are ready.
	utils.Sleep(1)

	if l.PID() == 0 { // avoid killing the current process
		return
	}

	killGroup(l.PID())
	p, err := os.FindProcess(l.PID())
	if err == nil {
		_ = p.Kill()
	}
}

// Cleanup wait until the Browser exits and remove UserDataDir
func (l *Launcher) Cleanup() {
	<-l.exit

	dir, _ := l.Get("user-data-dir")
	_ = os.RemoveAll(dir)
}

func (l *Launcher) normalizeFlag(name string) string {
	return strings.TrimLeft(name, "-")
}
