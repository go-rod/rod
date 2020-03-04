package launcher

import (
	"context"
	"errors"
	"io"
	nurl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ysmood/kit"
	"github.com/ysmood/leakless"
)

// Launcher is a helper to launch chrome binary smartly
type Launcher struct {
	ctx      context.Context
	bin      string
	leakless bool
	log      func(string)
	flags    map[string][]string
	output   chan string
	exit     chan kit.Nil
}

// New returns the default arguments to start chrome.
// "--" is optional, with or without it won't affect the result.
// All available switches: https://chromium.googlesource.com/chromium/src/+/32352ad08ee673a4d43e8593ce988b224f6482d3/chrome/common/chrome_switches.cc
func New() *Launcher {
	tmp := filepath.Join(os.TempDir(), "cdp", kit.RandString(8))
	kit.E(os.MkdirAll(tmp, 0700))

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
		"user-data-dir":                                      {tmp},

		// to prevent welcome page
		"": {"about:blank"},
	}

	return &Launcher{
		ctx:      context.Background(),
		leakless: true,
		flags:    defaultFlags,
		output:   make(chan string),
		exit:     make(chan kit.Nil),
	}
}

// Context set the context
func (l *Launcher) Context(ctx context.Context) *Launcher {
	l.ctx = ctx
	return l
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

// KillAfterExit switch. Whether to kill chrome or not after main process exits. Default value is true.
func (l *Launcher) KillAfterExit(enable bool) *Launcher {
	l.leakless = enable
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

// ExecFormat returns the formated arg list for cli
func (l *Launcher) ExecFormat() []string {
	execArgs := []string{}
	for k, v := range l.flags {
		if k == "" {
			continue
		}

		str := "--" + k
		if v != nil {
			str += "=" + strings.Join(v, ",")
		}
		execArgs = append(execArgs, str)
	}
	return append(execArgs, l.flags[""]...)
}

// Log function to handle stdout and stderr from chrome
func (l *Launcher) Log(log func(string)) *Launcher {
	l.log = log
	return l
}

// UserModeLaunch is a preset to enable reusing current user data. Useful for automation of personal browser.
func (l *Launcher) UserModeLaunch() string {
	port := l.flags["remote-debugging-port"][0]
	if port == "0" {
		port = "37712"
		l.Set("remote-debugging-port", port)
	}
	url, err := GetWebSocketDebuggerURL(context.Background(), "http://127.0.0.1:"+port)
	if err != nil {
		url = l.Headless(false).KillAfterExit(false).UserDataDir("").Launch()
	}
	return url
}

// Launch a standalone temp browser instance and returns the debug url.
// bin and profileDir are optional, set them to empty to use the default values.
// If you want to reuse sessions, such as cookies, set the userDataDir to the same location.
func (l *Launcher) Launch() string {
	u, err := l.LaunchE()
	kit.E(err)
	return u
}

// LaunchE doc is the same as the method Launch
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

	ll := leakless.New()

	var cmd *exec.Cmd
	if l.leakless {
		cmd = ll.Command(bin, l.ExecFormat()...)
	} else {
		cmd = exec.Command(bin, l.ExecFormat()...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	go l.read(stdout)
	go l.read(stderr)

	go func() {
		_ = cmd.Wait()
		close(l.exit)
	}()

	u, err := l.getURL()
	if err != nil {
		go func() {
			p, err := os.FindProcess(<-ll.Pid())
			kit.E(err)
			kit.E(p.Kill())
		}()
		return "", err
	}

	return u, nil
}

func (l *Launcher) read(reader io.Reader) {
	buf := make([]byte, 256*1024) // 256KB
	for {
		n, err := reader.Read(buf)
		if err != nil {
			return
		}
		str := string(buf[:n])
		if l.log != nil {
			l.log(str)
		}
		l.output <- str
	}
}

// ReadURL from chrome stderr
func (l *Launcher) getURL() (string, error) {
	out := ""

	for {
		select {
		case e := <-l.output:
			out += e

			if strings.Contains(out, "Opening in existing browser session") {
				return "", errors.New("[launcher] Quit the current running Chrome first")
			}

			str := regexp.MustCompile(`ws://.+/`).FindString(out)
			if str != "" {
				u, err := nurl.Parse(strings.TrimSpace(str))
				if err != nil {
					return "", err
				}
				return "http://" + u.Host, nil
			}
		case <-l.exit:
			return "", errors.New("[launcher] Failed to get the debug url: " + out)
		}
	}
}

// GetWebSocketDebuggerURL from chrome remote url
func GetWebSocketDebuggerURL(ctx context.Context, url string) (string, error) {
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

	obj, err := kit.Req(u.String()).Context(ctx).JSON()
	if err != nil {
		return "", err
	}
	return obj.Get("webSocketDebuggerUrl").String(), nil
}
