package launcher

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/leakless"
)

const (
	flagWorkingDir = "rod-working-dir"
	flagEnv        = "rod-env"
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
	reap      bool
	leakless  bool
}

// New returns the default arguments to start browser.
// "--" is optional, with or without it won't affect the result.
// Leakless will be enabled by default.
// List of switches: https://peter.sh/experiments/chromium-command-line-switches/
func New() *Launcher {
	dir := ""
	if defaults.Dir == "" {
		dir = filepath.Join(os.TempDir(), "rod", "user-data", utils.RandString(8))
	}

	defaultFlags := map[string][]string{
		"user-data-dir": {dir},

		// use random port by default
		"remote-debugging-port": {defaults.Port},

		// enable headless by default
		"headless": nil,

		// to prevent welcome page
		"": {"about:blank"},

		"disable-background-networking":                      nil,
		"disable-background-timer-throttling":                nil,
		"disable-backgrounding-occluded-windows":             nil,
		"disable-breakpad":                                   nil,
		"disable-client-side-phishing-detection":             nil,
		"disable-component-extensions-with-background-pages": nil,
		"disable-default-apps":                               nil,
		"disable-dev-shm-usage":                              nil,
		"disable-extensions":                                 nil,
		"disable-features":                                   {"site-per-process", "TranslateUI"},
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
		"no-first-run":                                       nil,
		"use-mock-keychain":                                  nil,
	}

	if defaults.Show {
		delete(defaultFlags, "headless")
	}

	if isInDocker {
		defaultFlags["no-sandbox"] = nil
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
		reap:      true,
		leakless:  true,
		logger:    ioutil.Discard,
	}
}

// NewUserMode is a preset to enable reusing current user data. Useful for automation of personal browser.
// If you see any error, it may because you can't launch debug port for existing browser, the solution is to
// completely close the running browser. Unfortunately, there's no API for rod to tell it automatically yet.
func NewUserMode() *Launcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Launcher{
		ctx:       ctx,
		ctxCancel: cancel,
		Flags: map[string][]string{
			"remote-debugging-port":  {"37712"},
			"disable-blink-features": {"AutomationControlled"},
		},
		exit:    make(chan struct{}),
		browser: NewBrowser(),
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
	flag, has := l.Flags[name]
	return flag, has
}

// Set flag
func (l *Launcher) Set(name string, values ...string) *Launcher {
	l.Flags[strings.TrimLeft(name, "-")] = values
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

// Delete flag
func (l *Launcher) Delete(name string) *Launcher {
	delete(l.Flags, strings.TrimLeft(name, "-"))
	return l
}

// Bin set browser executable file path
func (l *Launcher) Bin(path string) *Launcher {
	l.bin = path
	return l
}

// Headless switch. When disabled leakless will be disabled.
func (l *Launcher) Headless(enable bool) *Launcher {
	if enable {
		return l.Set("headless")
	}
	return l.Delete("headless")
}

// Leakless switch.
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
// When set to empty, system user's default dir will be used.
func (l *Launcher) UserDataDir(dir string) *Launcher {
	if dir == "" {
		l.Delete("user-data-dir")
	} else {
		l.Set("user-data-dir", dir)
	}
	return l
}

// RemoteDebuggingPort arg
func (l *Launcher) RemoteDebuggingPort(port int) *Launcher {
	return l.Set("remote-debugging-port", strconv.FormatInt(int64(port), 10))
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

// Logger to handle stdout and stderr from browser
func (l *Launcher) Logger(w io.Writer) *Launcher {
	l.logger = w
	return l
}

// Reap enable/disable a guard to cleanup zombie processes
func (l *Launcher) Reap(enable bool) *Launcher {
	l.reap = enable
	return l
}

// MustLaunch a standalone temp browser instance and returns the debug url.
// bin and profileDir are optional, set them to empty to use the default values.
// If you want to reuse sessions, such as cookies, set the userDataDir to the same location.
func (l *Launcher) MustLaunch() string {
	u, err := l.Launch()
	utils.E(err)
	return u
}

// Launch doc is similar to the method MustLaunch
func (l *Launcher) Launch() (string, error) {
	if l.reap {
		runReaper()
	}

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
		u, err := GetWebSocketDebuggerURL("http://127.0.0.1:" + port)
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
		go l.kill()
		return "", err
	}

	return GetWebSocketDebuggerURL(u)
}

func (l *Launcher) setupCmd(cmd *exec.Cmd) {
	dir, _ := l.Get(flagWorkingDir)
	env, _ := l.GetFlags(flagEnv)
	cmd.Dir = dir
	cmd.Env = env

	r, w := io.Pipe()
	cmd.Stdout = w
	cmd.Stderr = w
	go func() { _, _ = io.Copy(io.MultiWriter(l.logger, l.parser), r) }()
	go func() {
		<-l.ctx.Done()
		_ = r.CloseWithError(io.EOF)
		_ = w.CloseWithError(io.EOF)
	}()
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
		err = errors.New("[launcher] Failed to get the debug url")
	}
	return
}

// PID returns the browser process pid
func (l *Launcher) PID() int {
	return l.pid
}

// Cleanup wait until the Browser exits and release related resources
func (l *Launcher) Cleanup() {
	<-l.exit

	dir, _ := l.Get("user-data-dir")
	_ = os.RemoveAll(dir)
}

func (l *Launcher) kill() {
	p, err := os.FindProcess(l.pid)
	if err == nil {
		_ = p.Kill()
	}
}
