package cdp

import (
	"context"
	"errors"
	"io"
	"net/url"
	nurl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/fetcher"
)

// ChromeArgs returns the default arguments to start chrome.
func ChromeArgs() map[string][]string {
	return map[string][]string{
		"--disable-background-networking":                      nil,
		"--enable-features":                                    {"NetworkService", "NetworkServiceInProcess"},
		"--disable-background-timer-throttling":                nil,
		"--disable-backgrounding-occluded-windows":             nil,
		"--disable-breakpad":                                   nil,
		"--disable-client-side-phishing-detection":             nil,
		"--disable-component-extensions-with-background-pages": nil,
		"--disable-default-apps":                               nil,
		"--disable-dev-shm-usage":                              nil,
		"--disable-extensions":                                 nil,
		"--disable-hang-monitor":                               nil,
		"--disable-ipc-flooding-protection":                    nil,
		"--disable-popup-blocking":                             nil,
		"--disable-prompt-on-repost":                           nil,
		"--disable-renderer-backgrounding":                     nil,
		"--disable-sync":                                       nil,
		"--force-color-profile":                                {"srgb"},
		"--metrics-recording-only":                             nil,
		"--no-first-run":                                       nil,
		"--enable-automation":                                  nil,
		"--password-store":                                     {"basic"},
		"--use-mock-keychain":                                  nil,
		"--remote-debugging-port":                              {"0"},
		"--headless":                                           nil,
		"about:blank":                                          nil,

		// disable site-per-process to make sure iframes are not detached automatically
		"--disable-features": {"site-per-process", "TranslateUI"},
	}
}

// LaunchBrowser a standalone temp browser instance and returns the debug url
func LaunchBrowser(bin string, args map[string][]string) (string, error) {
	if bin == "" {
		var err error
		bin, err = new(fetcher.Chrome).Get()
		if err != nil {
			return "", err
		}
	}

	if args == nil {
		args = ChromeArgs()
	}

	if _, has := args["--user-data-dir"]; !has {
		tmp := filepath.Join(os.TempDir(), "cdp", kit.RandString(8))
		err := os.MkdirAll(tmp, 0700)
		if err != nil {
			return "", err
		}
		args["--user-data-dir"] = []string{tmp}
	}

	execArgs := []string{}
	for k, v := range args {
		str := k
		if v != nil {
			str += "=" + strings.Join(v, ",")
		}
		execArgs = append(execArgs, str)
	}

	cmd := exec.Command(
		bin,
		execArgs...,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	u, err := getURL(stderr)
	if err != nil {
		_ = cmd.Process.Kill()
		return "", err
	}

	return u, nil
}

func getURL(stderr io.ReadCloser) (string, error) {
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
		return "", errors.New("[cdp] launch chrome timeout: " + out)
	case <-wait:
	}

	u, err := url.Parse(strings.TrimSpace(str))
	if err != nil {
		return "", errors.New("[cdp] failed to get control url: " + out + " " + err.Error())
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

func checkPanic(err error) {
	if err == nil {
		return
	}
	panic(kit.Sdump(err))
}

var isDebug = os.Getenv("debug_cdp") == "true"

func debug(obj interface{}) {
	if !isDebug {
		return
	}

	kit.Log(kit.Sdump(obj))
}
