package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/kit"
	"github.com/ysmood/leakless"
)

// Launcher is a helper to launch browser binary smartly
type Launcher struct {
	ctx       context.Context
	ctxCancel func()
	bin       string
	url       string
	log       func(string)
	Flags     map[string][]string `json:"flags"`
	output    chan string
	pid       int
	exit      chan kit.Nil
	remote    bool // remote mode or not
	reap      bool
}

// New returns the default arguments to start browser.
// "--" is optional, with or without it won't affect the result.
// List of switches: https://peter.sh/experiments/chromium-command-line-switches/
func New() *Launcher {
	dir := ""
	if defaults.Dir == "" {
		dir = filepath.Join(os.TempDir(), "rod", "user-data", kit.RandString(8))
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
		output:    make(chan string),
		exit:      make(chan kit.Nil),
		bin:       defaults.Bin,
		reap:      true,
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
		output: make(chan string),
		exit:   make(chan kit.Nil),
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
	l.Set(name, append(flags, values...)...)
	return l
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

// Headless switch
func (l *Launcher) Headless(enable bool) *Launcher {
	if enable {
		l.Set("headless")
	} else {
		l.Delete("headless")
	}
	return l
}

// Devtools switch to auto open devtools for each tab
func (l *Launcher) Devtools(autoOpenForTabs bool) *Launcher {
	if autoOpenForTabs {
		l.Set("auto-open-devtools-for-tabs")
	} else {
		l.Delete("auto-open-devtools-for-tabs")
	}
	return l
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
	l.Set("remote-debugging-port", strconv.FormatInt(int64(port), 10))
	return l
}

// FormatArgs returns the formated arg list for cli
func (l *Launcher) FormatArgs() []string {
	execArgs := []string{}
	for k, v := range l.Flags {
		if k == "" {
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

// Log function to handle stdout and stderr from browser
func (l *Launcher) Log(log func(string)) *Launcher {
	l.log = log
	return l
}

// Reap enable/disable a guard to cleanup zombie processes
func (l *Launcher) Reap(enable bool) *Launcher {
	l.reap = enable
	return l
}

// Launch a standalone temp browser instance and returns the debug url.
// bin and profileDir are optional, set them to empty to use the default values.
// If you want to reuse sessions, such as cookies, set the userDataDir to the same location.
func (l *Launcher) Launch() string {
	u, err := l.LaunchE()
	utils.E(err)
	return u
}

// LaunchE doc is similar to the method Launch
func (l *Launcher) LaunchE() (wsURL string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	if l.reap {
		runReaper()
	}

	defer l.ctxCancel()

	bin := l.bin
	if bin == "" {
		var err error
		b := NewBrowser()
		b.Context = l.ctx
		bin, err = b.Get()
		utils.E(err)
	}

	var ll *leakless.Launcher
	var cmd *exec.Cmd

	_, headless := l.Get("headless")

	if headless && leakless.Support() {
		ll = leakless.New()
		cmd = ll.Command(bin, l.FormatArgs()...)
	} else {
		port, _ := l.Get("remote-debugging-port")
		u, err := GetWebSocketDebuggerURL(l.ctx, "http://127.0.0.1:"+port)
		if err == nil {
			return u, nil
		}
		cmd = exec.Command(bin, l.FormatArgs()...)
	}

	stdout, err := cmd.StdoutPipe()
	utils.E(err)
	defer func() {
		if l.log == nil {
			_ = stdout.Close()
		}
	}()

	stderr, err := cmd.StderrPipe()
	utils.E(err)
	defer func() {
		if l.log == nil {
			_ = stderr.Close()
		}
	}()

	err = cmd.Start()
	utils.E(err)

	if headless {
		select {
		case <-l.exit:
		case pid := <-ll.Pid():
			l.pid = pid
			if ll.Err() != "" {
				return "", errors.New(ll.Err())
			}
		}
	} else {
		l.pid = cmd.Process.Pid
	}

	go l.read(stdout)
	go l.read(stderr)

	go func() {
		_ = cmd.Wait()
		close(l.exit)
	}()

	u, err := l.getURL()
	if err != nil {
		go l.kill()
		return "", err
	}

	return GetWebSocketDebuggerURL(l.ctx, u)
}

// PID returns the browser process pid
func (l *Launcher) PID() int {
	return l.pid
}

// Cleanup wait until the Browser exits and release related resources
func (l *Launcher) Cleanup() {
	<-l.exit
	if _, has := l.Get("keep-user-data-dir"); !has {
		dir, _ := l.Get("user-data-dir")
		if l.log != nil {
			l.log(fmt.Sprintln(utils.C("Remove", "cyan"), dir))
		}

		_ = os.RemoveAll(dir)
	}
}

func (l *Launcher) kill() {
	p, err := os.FindProcess(l.pid)
	if err == nil {
		_ = p.Kill()
	}
}

func (l *Launcher) read(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if l.log != nil {
			l.log(scanner.Text() + "\n")
		}
		select {
		case <-l.ctx.Done():
			if l.log == nil {
				return
			}
		case l.output <- scanner.Text() + "\n":
		}
	}
}

// ReadURL from browser stderr
func (l *Launcher) getURL() (u string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	out := ""

	for {
		select {
		case <-l.ctx.Done():
			utils.E(l.ctx.Err())
		case e := <-l.output:
			out += e

			if strings.Contains(out, "Opening in existing browser session") {
				utils.E(errors.New("[launcher] Quit the current running browser first"))
			}

			str := regexp.MustCompile(`ws://.+/`).FindString(out)
			if str != "" {
				u, err := url.Parse(strings.TrimSpace(str))
				utils.E(err)
				return "http://" + u.Host, nil
			}
		case <-l.exit:
			utils.E(errors.New("[launcher] Failed to get the debug url: " + out))
		}
	}
}

// KeepUserDataDir after browser is closed. By default user-data-dir will be removed.
func (l *Launcher) KeepUserDataDir() *Launcher {
	l.Set("keep-user-data-dir")
	return l
}

// GetWebSocketDebuggerURL from browser remote url
func GetWebSocketDebuggerURL(ctx context.Context, u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	toHTTP(parsed)

	parsed.Path = "/json/version"

	obj, err := kit.Req(parsed.String()).Context(ctx).JSON()
	if err != nil {
		return "", err
	}
	return obj.Get("webSocketDebuggerUrl").String(), nil
}
