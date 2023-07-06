// Package launcher for launching browser utils.
package launcher

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/leakless"
)

// DefaultUserDataDirPrefix ...
var DefaultUserDataDirPrefix = filepath.Join(os.TempDir(), "rod", "user-data")

// Launcher is a helper to launch browser binary smartly
type Launcher struct {
	Flags map[flags.Flag][]string `json:"flags"`

	ctx       context.Context
	ctxCancel func()

	logger io.Writer

	browser *Browser
	parser  *URLParser
	pid     int
	exit    chan struct{}

	managed    bool
	serviceURL string
}

// New returns the default arguments to start browser.
// Headless will be enabled by default.
// Leakless will be enabled by default.
// UserDataDir will use OS tmp dir by default, this folder will usually be cleaned up by the OS after reboot.
func New() *Launcher {
	dir := defaults.Dir
	if dir == "" {
		dir = filepath.Join(DefaultUserDataDirPrefix, utils.RandString(8))
	}

	defaultFlags := map[flags.Flag][]string{
		flags.Bin:      {defaults.Bin},
		flags.Leakless: nil,

		flags.UserDataDir: {dir},

		// use random port by default
		flags.RemoteDebuggingPort: {defaults.Port},

		// enable headless by default
		flags.Headless: nil,

		// to disable the init blank window
		"no-first-run":      nil,
		"no-startup-window": nil,

		// TODO: about the "site-per-process" see https://github.com/puppeteer/puppeteer/issues/2548
		"disable-features": {"site-per-process", "TranslateUI"},

		"disable-dev-shm-usage":                              nil,
		"disable-background-networking":                      nil,
		"disable-background-timer-throttling":                nil,
		"disable-backgrounding-occluded-windows":             nil,
		"disable-breakpad":                                   nil,
		"disable-client-side-phishing-detection":             nil,
		"disable-component-extensions-with-background-pages": nil,
		"disable-default-apps":                               nil,
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
		delete(defaultFlags, flags.Headless)
	}
	if defaults.Devtools {
		defaultFlags["auto-open-devtools-for-tabs"] = nil
	}
	if inContainer {
		defaultFlags[flags.NoSandbox] = nil
	}
	if defaults.Proxy != "" {
		defaultFlags[flags.ProxyServer] = []string{defaults.Proxy}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Launcher{
		ctx:       ctx,
		ctxCancel: cancel,
		Flags:     defaultFlags,
		exit:      make(chan struct{}),
		browser:   NewBrowser(),
		parser:    NewURLParser(),
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
		Flags: map[flags.Flag][]string{
			flags.RemoteDebuggingPort: {"37712"},
			"no-startup-window":       nil,
			flags.Bin:                 {bin},
		},
		browser: NewBrowser(),
		exit:    make(chan struct{}),
		parser:  NewURLParser(),
		logger:  ioutil.Discard,
	}
}

// NewAppMode is a preset to run the browser like a native application.
func NewAppMode(u string) *Launcher {
	l := New()
	l.Set(flags.App, u).
		Set(flags.Env, "GOOGLE_API_KEY=no").
		Headless(false).
		Delete("no-startup-window").
		Delete("enable-automation")
	return l
}

// Context sets the context
func (l *Launcher) Context(ctx context.Context) *Launcher {
	ctx, cancel := context.WithCancel(ctx)
	l.ctx = ctx
	l.parser.Context(ctx)
	l.ctxCancel = cancel
	return l
}

// Set a command line argument when launching the browser. Be careful the first argument is a flag name,
// it shouldn't contain values. The values the will be joined with comma.
// You can use the [Launcher.FormatArgs] to debug the final CLI arguments.
func (l *Launcher) Set(name flags.Flag, values ...string) *Launcher {
	name.Check()
	l.Flags[name.NormalizeFlag()] = values
	return l
}

// Get flag's first value
func (l *Launcher) Get(name flags.Flag) string {
	if list, has := l.GetFlags(name); has {
		return list[0]
	}
	return ""
}

// Has flag or not
func (l *Launcher) Has(name flags.Flag) bool {
	_, has := l.GetFlags(name)
	return has
}

// GetFlags from settings
func (l *Launcher) GetFlags(name flags.Flag) ([]string, bool) {
	flag, has := l.Flags[name.NormalizeFlag()]
	return flag, has
}

// Append values to the flag
func (l *Launcher) Append(name flags.Flag, values ...string) *Launcher {
	flags, has := l.GetFlags(name)
	if !has {
		flags = []string{}
	}
	return l.Set(name, append(flags, values...)...)
}

// Delete a flag
func (l *Launcher) Delete(name flags.Flag) *Launcher {
	delete(l.Flags, name.NormalizeFlag())
	return l
}

// Bin of the browser binary path to launch, if the path is not empty the auto download will be disabled
func (l *Launcher) Bin(path string) *Launcher {
	return l.Set(flags.Bin, path)
}

// Revision of the browser to auto download
func (l *Launcher) Revision(rev int) *Launcher {
	l.browser.Revision = rev
	return l
}

// Headless switch. Whether to run browser in headless mode. A mode without visible UI.
func (l *Launcher) Headless(enable bool) *Launcher {
	if enable {
		return l.Set(flags.Headless)
	}
	return l.Delete(flags.Headless)
}

// NoSandbox switch. Whether to run browser in no-sandbox mode.
// Linux users may face "running as root without --no-sandbox is not supported" in some Linux/Chrome combinations. This function helps switch mode easily.
// Be aware disabling sandbox is not trivial. Use at your own risk.
// Related doc: https://bugs.chromium.org/p/chromium/issues/detail?id=638180
func (l *Launcher) NoSandbox(enable bool) *Launcher {
	if enable {
		return l.Set(flags.NoSandbox)
	}
	return l.Delete(flags.NoSandbox)
}

// XVFB enables to run browser in by XVFB. Useful when you want to run headful mode on linux.
func (l *Launcher) XVFB(args ...string) *Launcher {
	return l.Set(flags.XVFB, args...)
}

// Leakless switch. If enabled, the browser will be force killed after the Go process exits.
// The doc of leakless: https://github.com/ysmood/leakless.
func (l *Launcher) Leakless(enable bool) *Launcher {
	if enable {
		return l.Set(flags.Leakless)
	}
	return l.Delete(flags.Leakless)
}

// Devtools switch to auto open devtools for each tab
func (l *Launcher) Devtools(autoOpenForTabs bool) *Launcher {
	if autoOpenForTabs {
		return l.Set("auto-open-devtools-for-tabs")
	}
	return l.Delete("auto-open-devtools-for-tabs")
}

// IgnoreCerts configure the Chrome's ignore-certificate-errors-spki-list argument with the public keys.
func (l *Launcher) IgnoreCerts(pks []crypto.PublicKey) error {
	spkis := make([]string, 0, len(pks))

	for _, pk := range pks {
		spki, err := certSPKI(pk)
		if err != nil {
			return fmt.Errorf("certSPKI: %w", err)
		}
		spkis = append(spkis, string(spki))
	}

	l.Set("ignore-certificate-errors-spki-list", spkis...)

	return nil
}

// UserDataDir is where the browser will look for all of its state, such as cookie and cache.
// When set to empty, browser will use current OS home dir.
// Related doc: https://chromium.googlesource.com/chromium/src/+/master/docs/user_data_dir.md
func (l *Launcher) UserDataDir(dir string) *Launcher {
	if dir == "" {
		l.Delete(flags.UserDataDir)
	} else {
		l.Set(flags.UserDataDir, dir)
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
// If it's not zero and the Launcher.Leakless is disabled, the launcher will try to reconnect to it first,
// if the reconnection fails it will launch a new browser.
func (l *Launcher) RemoteDebuggingPort(port int) *Launcher {
	return l.Set(flags.RemoteDebuggingPort, fmt.Sprintf("%d", port))
}

// Proxy for the browser
func (l *Launcher) Proxy(host string) *Launcher {
	return l.Set(flags.ProxyServer, host)
}

// WorkingDir to launch the browser process.
func (l *Launcher) WorkingDir(path string) *Launcher {
	return l.Set(flags.WorkingDir, path)
}

// Env to launch the browser process. The default value is os.Environ().
// Usually you use it to set the timezone env. Such as:
//
//	Env(append(os.Environ(), "TZ=Asia/Tokyo")...)
func (l *Launcher) Env(env ...string) *Launcher {
	return l.Set(flags.Env, env...)
}

// StartURL to launch
func (l *Launcher) StartURL(u string) *Launcher {
	return l.Set("", u)
}

// FormatArgs returns the formatted arg list for cli
func (l *Launcher) FormatArgs() []string {
	execArgs := []string{}
	for k, v := range l.Flags {
		if k == flags.Arguments {
			continue
		}

		if strings.HasPrefix(string(k), "rod-") {
			continue
		}

		// fix a bug of chrome, if path is not absolute chrome will hang
		if k == flags.UserDataDir {
			abs, err := filepath.Abs(v[0])
			utils.E(err)
			v[0] = abs
		}

		str := "--" + string(k)
		if v != nil {
			str += "=" + strings.Join(v, ",")
		}
		execArgs = append(execArgs, str)
	}

	execArgs = append(execArgs, l.Flags[flags.Arguments]...)
	sort.Strings(execArgs)
	return execArgs
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

	if l.Has(flags.Leakless) && leakless.Support() {
		ll = leakless.New()
		cmd = ll.Command(bin, l.FormatArgs()...)
	} else {
		port := l.Get(flags.RemoteDebuggingPort)
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

	dir := l.Get(flags.WorkingDir)
	env, _ := l.GetFlags(flags.Env)
	cmd.Dir = dir
	cmd.Env = env

	cmd.Stdout = io.MultiWriter(l.logger, l.parser)
	cmd.Stderr = io.MultiWriter(l.logger, l.parser)
}

func (l *Launcher) getBin() (string, error) {
	bin := l.Get(flags.Bin)
	if bin == "" {
		l.browser.Context = l.ctx
		return l.browser.Get()
	}
	return bin, nil
}

func (l *Launcher) getURL() (u string, err error) {
	select {
	case <-l.ctx.Done():
		err = l.ctx.Err()
	case u = <-l.parser.URL:
	case <-l.exit:
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

	dir := l.Get(flags.UserDataDir)
	_ = os.RemoveAll(dir)
}
