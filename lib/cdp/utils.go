package cdp

import (
	"encoding/json"
	"net/url"
	nurl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/fetcher"
)

// LaunchBrowser a standalone temp browser instance and returns the debug url
func LaunchBrowser(bin string, headless bool) (string, error) {
	if bin == "" {
		var err error
		bin, err = new(fetcher.Chrome).Get()
		if err != nil {
			return "", err
		}
	}

	tmp := filepath.Join(os.TempDir(), "cdp", kit.RandString(8))

	err := os.MkdirAll(tmp, 0700)
	if err != nil {
		return "", err
	}

	args := []string{
		// Copied from https://github.com/puppeteer/puppeteer/blob/8b49dc62a62282543ead43541316e23d3450ff3c/lib/Launcher.js#L260
		"--disable-background-networking",
		"--enable-features=NetworkService,NetworkServiceInProcess",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-breakpad",
		"--disable-client-side-phishing-detection",
		"--disable-component-extensions-with-background-pages",
		"--disable-default-apps",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		// disable site-per-process to make sure iframes are not detached automatically
		"--disable-features=site-per-process,TranslateUI",
		"--disable-hang-monitor",
		"--disable-ipc-flooding-protection",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-renderer-backgrounding",
		"--disable-sync",
		"--force-color-profile=srgb",
		"--metrics-recording-only",
		"--no-first-run",
		"--enable-automation",
		"--password-store=basic",
		"--use-mock-keychain",

		"--remote-debugging-port=0",
		"--user-data-dir=" + tmp,
	}

	if headless {
		args = append(args, "--headless")
	}

	args = append(args, "about:blank")

	cmd := exec.Command(
		bin,
		args...,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	buf := make([]byte, 100)
	str := ""
	out := ""
	for {
		n, err := stderr.Read(buf)
		if err != nil {
			return "", err
		}
		out += string(buf[:n])

		str = regexp.MustCompile(`ws://.+`).FindString(out)
		if str != "" {
			break
		}
	}

	u, err := url.Parse(strings.TrimSpace(str))
	if err != nil {
		return "", err
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

func debug(prefix string, data []byte) {
	if !isDebug {
		return
	}

	var obj interface{}
	kit.E(json.Unmarshal(data, &obj))

	kit.Log(kit.C("[cdp]", "cyan"), prefix, kit.Sdump(obj))
}
