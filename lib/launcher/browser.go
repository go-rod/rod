package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/fetchup"
	"github.com/ysmood/leakless"
)

// Host to download browser
type Host func(revision int) string

var hostConf = map[string]struct {
	urlPrefix string
	zipName   string
}{
	"darwin_amd64":  {"Mac", "chrome-mac.zip"},
	"darwin_arm64":  {"Mac_Arm", "chrome-mac.zip"},
	"linux_amd64":   {"Linux_x64", "chrome-linux.zip"},
	"windows_386":   {"Win", "chrome-win.zip"},
	"windows_amd64": {"Win_x64", "chrome-win.zip"},
}[runtime.GOOS+"_"+runtime.GOARCH]

// HostGoogle to download browser
func HostGoogle(revision int) string {
	return fmt.Sprintf(
		"https://storage.googleapis.com/chromium-browser-snapshots/%s/%d/%s",
		hostConf.urlPrefix,
		revision,
		hostConf.zipName,
	)
}

// HostNPM to download browser
func HostNPM(revision int) string {
	return fmt.Sprintf(
		"https://registry.npmmirror.com/-/binary/chromium-browser-snapshots/%s/%d/%s",
		hostConf.urlPrefix,
		revision,
		hostConf.zipName,
	)
}

// HostPlaywright to download browser
func HostPlaywright(revision int) string {
	rev := RevisionPlaywright
	if !(runtime.GOOS == "linux" && runtime.GOARCH == "arm64") {
		rev = revision
	}
	return fmt.Sprintf(
		"https://playwright.azureedge.net/builds/chromium/%d/chromium-linux-arm64.zip",
		rev,
	)
}

// DefaultBrowserDir for downloaded browser. For unix is "$HOME/.cache/rod/browser",
// for Windows it's "%APPDATA%\rod\browser"
var DefaultBrowserDir = filepath.Join(map[string]string{
	"windows": filepath.Join(os.Getenv("APPDATA")),
	"darwin":  filepath.Join(os.Getenv("HOME"), ".cache"),
	"linux":   filepath.Join(os.Getenv("HOME"), ".cache"),
}[runtime.GOOS], "rod", "browser")

// Browser is a helper to download browser smartly
type Browser struct {
	Context context.Context

	// Hosts are the candidates to download the browser.
	Hosts []Host

	// Revision of the browser to use
	Revision int

	// RootDir to download different browser versions.
	RootDir string

	// Log to print output
	Logger utils.Logger

	// LockPort a tcp port to prevent race downloading. Default is 2968 .
	LockPort int

	// HTTPClient to download the browser
	HTTPClient *http.Client
}

// NewBrowser with default values
func NewBrowser() *Browser {
	return &Browser{
		Context:  context.Background(),
		Revision: RevisionDefault,
		Hosts:    []Host{HostGoogle, HostNPM, HostPlaywright},
		RootDir:  DefaultBrowserDir,
		Logger:   log.New(os.Stdout, "[launcher.Browser]", log.LstdFlags),
		LockPort: defaults.LockPort,
	}
}

// Dir to download the browser
func (lc *Browser) Dir() string {
	return filepath.Join(lc.RootDir, fmt.Sprintf("chromium-%d", lc.Revision))
}

// BinPath to download the browser executable
func (lc *Browser) BinPath() string {
	bin := map[string]string{
		"darwin":  "Chromium.app/Contents/MacOS/Chromium",
		"linux":   "chrome",
		"windows": "chrome.exe",
	}[runtime.GOOS]

	return filepath.Join(lc.Dir(), filepath.FromSlash(bin))
}

// Download browser from the fastest host. It will race downloading a TCP packet from each host and use the fastest host.
func (lc *Browser) Download() error {
	us := []string{}
	for _, host := range lc.Hosts {
		us = append(us, host(lc.Revision))
	}

	dir := lc.Dir()

	fu := fetchup.New(dir, us...)
	fu.Ctx = lc.Context
	fu.Logger = lc.Logger
	if lc.HTTPClient != nil {
		fu.HttpClient = lc.HTTPClient
	}

	err := fu.Fetch()
	if err != nil {
		return fmt.Errorf("Can't find a browser binary for your OS, the doc might help https://go-rod.github.io/#/compatibility?id=os : %w", err)
	}

	return fetchup.StripFirstDir(dir)
}

// Get is a smart helper to get the browser executable path.
// If Destination is not valid it will auto download the browser to Destination.
func (lc *Browser) Get() (string, error) {
	defer leakless.LockPort(lc.LockPort)()

	if lc.Validate() == nil {
		return lc.BinPath(), nil
	}

	// Try to cleanup before downloading
	_ = os.RemoveAll(lc.Dir())

	return lc.BinPath(), lc.Download()
}

// MustGet is similar with Get
func (lc *Browser) MustGet() string {
	p, err := lc.Get()
	utils.E(err)
	return p
}

// Validate returns nil if the browser executable valid.
// If the executable is malformed it will return error.
func (lc *Browser) Validate() error {
	_, err := os.Stat(lc.BinPath())
	if err != nil {
		return err
	}

	cmd := exec.Command(lc.BinPath(), "--headless", "--no-sandbox",
		"--disable-gpu", "--dump-dom", "about:blank")
	b, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(b), "error while loading shared libraries") {
			// When the os is missing some dependencies for chromium we treat it as valid binary.
			return nil
		}

		return fmt.Errorf("failed to run the browser: %w\n%s", err, b)
	}
	if !bytes.Contains(b, []byte(`<html><head></head><body></body></html>`)) {
		return errors.New("the browser executable doesn't support headless mode")
	}

	return nil
}

// LookPath searches for the browser executable from often used paths on current operating system.
func LookPath() (found string, has bool) {
	list := map[string][]string{
		"darwin": {
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		},
		"linux": {
			"chrome",
			"google-chrome",
			"/usr/bin/google-chrome",
			"microsoft-edge",
			"/usr/bin/microsoft-edge",
			"chromium",
			"chromium-browser",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
			"/data/data/com.termux/files/usr/bin/chromium-browser",
		},
		"openbsd": {
			"chrome",
			"chromium",
		},
		"windows": append([]string{"chrome", "edge"}, expandWindowsExePaths(
			`Google\Chrome\Application\chrome.exe`,
			`Chromium\Application\chrome.exe`,
			`Microsoft\Edge\Application\msedge.exe`,
		)...),
	}[runtime.GOOS]

	for _, path := range list {
		var err error
		found, err = exec.LookPath(path)
		has = err == nil
		if has {
			break
		}
	}

	return
}

// interface for testing
var openExec = exec.Command

// Open tries to open the url via system's default browser.
func Open(url string) {
	// Windows doesn't support format [::]
	url = strings.Replace(url, "[::]", "[::1]", 1)

	if bin, has := LookPath(); has {
		p := openExec(bin, url)
		_ = p.Start()
		_ = p.Process.Release()
	}
}

func expandWindowsExePaths(list ...string) []string {
	newList := []string{}
	for _, p := range list {
		newList = append(
			newList,
			filepath.Join(os.Getenv("ProgramFiles"), p),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), p),
			filepath.Join(os.Getenv("LocalAppData"), p),
		)
	}

	return newList
}
