package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
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

	// Dir to download browser.
	Dir string

	// Log to print output
	Logger utils.Logger

	// LockPort a tcp port to prevent race downloading. Default is 2968 .
	LockPort int
}

// NewBrowser with default values
func NewBrowser() *Browser {
	return &Browser{
		Context:  context.Background(),
		Revision: RevisionDefault,
		Hosts:    []Host{HostGoogle, HostNPM, HostPlaywright},
		Dir:      DefaultBrowserDir,
		Logger:   log.New(os.Stdout, "[launcher.Browser]", log.LstdFlags),
		LockPort: defaults.LockPort,
	}
}

// Destination of the downloaded browser executable
func (lc *Browser) Destination() string {
	bin := map[string]string{
		"darwin":  fmt.Sprintf("chromium-%d/chrome-mac/Chromium.app/Contents/MacOS/Chromium", lc.Revision),
		"linux":   fmt.Sprintf("chromium-%d/chrome-linux/chrome", lc.Revision),
		"windows": fmt.Sprintf("chromium-%d/chrome-win/chrome.exe", lc.Revision),
	}[runtime.GOOS]

	return filepath.Join(lc.Dir, bin)
}

// Download browser from the fastest host. It will race downloading a TCP packet from each host and use the fastest host.
func (lc *Browser) Download() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	u, err := lc.fastestHost()
	utils.E(err)

	if u == "" {
		panic(fmt.Errorf("Can't find a browser binary for your OS, the doc might help https://go-rod.github.io/#/compatibility?id=os"))
	}

	return lc.download(lc.Context, u)
}

func (lc *Browser) fastestHost() (fastest string, err error) {
	lc.Logger.Println("try to find the fastest host to download the browser binary")

	setURL := sync.Once{}
	ctx, cancel := context.WithCancel(lc.Context)
	defer cancel()

	wg := sync.WaitGroup{}
	for _, host := range lc.Hosts {
		u := host(lc.Revision)

		lc.Logger.Println("check", u)
		wg.Add(1)

		go func() {
			defer func() {
				err := recover()
				if err != nil {
					lc.Logger.Println("check result:", err)
				}
				wg.Done()
			}()

			q, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			utils.E(err)

			res, err := lc.httpClient().Do(q)
			utils.E(err)
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode == http.StatusOK {
				buf := make([]byte, 64*1024) // a TCP packet won't be larger than 64KB
				_, err = res.Body.Read(buf)
				utils.E(err)

				setURL.Do(func() {
					fastest = u
					cancel()
				})
			}
		}()
	}
	wg.Wait()

	return
}

func (lc *Browser) download(ctx context.Context, u string) error {
	lc.Logger.Println("Download:", u)

	zipPath := filepath.Join(lc.Dir, fmt.Sprintf("chromium-%d.zip", lc.Revision))

	err := utils.Mkdir(lc.Dir)
	utils.E(err)

	zipFile, err := os.Create(zipPath)
	utils.E(err)

	q, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	utils.E(err)

	res, err := lc.httpClient().Do(q)
	utils.E(err)
	defer func() { _ = res.Body.Close() }()

	size, _ := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)

	if res.StatusCode >= 400 || size < 1024*1024 {
		b, err := ioutil.ReadAll(res.Body)
		utils.E(err)
		err = errors.New("failed to download the browser")
		return fmt.Errorf("%w: %d %s", err, res.StatusCode, string(b))
	}

	progress := &progresser{
		size:   int(size),
		logger: lc.Logger,
	}

	_, err = io.Copy(io.MultiWriter(progress, zipFile), res.Body)
	utils.E(err)

	err = zipFile.Close()
	utils.E(err)

	unzipPath := filepath.Join(lc.Dir, fmt.Sprintf("chromium-%d", lc.Revision))
	_ = os.RemoveAll(unzipPath)
	utils.E(unzip(lc.Logger, zipPath, unzipPath))
	return os.Remove(zipPath)
}

func (lc *Browser) httpClient() *http.Client {
	return &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
}

// Get is a smart helper to get the browser executable path.
// If Destination is not valid it will auto download the browser to Destination.
func (lc *Browser) Get() (string, error) {
	defer leakless.LockPort(lc.LockPort)()

	if lc.Validate() == nil {
		return lc.Destination(), nil
	}

	return lc.Destination(), lc.Download()
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
	_, err := os.Stat(lc.Destination())
	if err != nil {
		return err
	}

	cmd := exec.Command(lc.Destination(), "--headless", "--no-sandbox",
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
