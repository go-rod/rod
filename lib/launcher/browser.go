package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	zipName   string
	urlPrefix string
}{
	"darwin":  {"chrome-mac.zip", "Mac"},
	"linux":   {"chrome-linux.zip", "Linux_x64"},
	"windows": {"chrome-win.zip", "Win"},
}[runtime.GOOS]

// HostGoogle to download browser
func HostGoogle(revision int) string {
	return fmt.Sprintf(
		"https://storage.googleapis.com/chromium-browser-snapshots/%s/%d/%s",
		hostConf.urlPrefix,
		revision,
		hostConf.zipName,
	)
}

// HostTaobao to download browser
func HostTaobao(revision int) string {
	return fmt.Sprintf(
		"https://npm.taobao.org/mirrors/chromium-browser-snapshots/%s/%d/%s",
		hostConf.urlPrefix,
		revision,
		hostConf.zipName,
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

	// Dir to download broweser.
	Dir string

	// Log to print output
	Logger io.Writer

	// LockPort a tcp port to prevent race downloading. Default is 2968 .
	LockPort int
}

// NewBrowser with default values
func NewBrowser() *Browser {
	return &Browser{
		Context:  context.Background(),
		Revision: DefaultRevision,
		Hosts:    []Host{HostGoogle, HostTaobao},
		Dir:      DefaultBrowserDir,
		Logger:   os.Stdout,
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
	return lc.download(lc.Context, u)
}

func (lc *Browser) fastestHost() (fastest string, err error) {
	setURL := sync.Once{}
	ctx, cancel := context.WithCancel(lc.Context)
	defer cancel()

	for _, host := range lc.Hosts {
		u := host(lc.Revision)

		go func() {
			defer func() {
				_ = recover()
				cancel()
			}()

			q, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			utils.E(err)

			res, err := lc.httpClient().Do(q)
			utils.E(err)
			defer func() { _ = res.Body.Close() }()

			buf := make([]byte, 64*1024) // a TCP packet won't be larger than 64KB
			_, err = res.Body.Read(buf)
			utils.E(err)

			setURL.Do(func() {
				fastest = u
			})
		}()
	}

	<-ctx.Done()

	return
}

func (lc *Browser) download(ctx context.Context, u string) error {
	_, _ = fmt.Fprintln(lc.Logger, "Download:", u)

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
// If Destination doesn't exists it will download the browser to Destination.
func (lc *Browser) Get() (string, error) {
	defer leakless.LockPort(lc.LockPort)()

	if lc.Exists() {
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

// Exists returns true if the browser executable path exists.
func (lc *Browser) Exists() bool {
	_, err := os.Stat(lc.Destination())
	return err == nil
}

// LookPath searches for the browser executable from often used paths on current operating system.
func LookPath() (found string, has bool) {
	list := map[string][]string{
		"darwin": {
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		},
		"linux": {
			"chrome",
			"google-chrome",
			"/usr/bin/google-chrome",
			"microsoft-edge",
			"/usr/bin/microsoft-edge",
			"chromium",
			"chromium-browser",
		},
		"windows": append([]string{"chrome", "edge"}, expandWindowsExePaths(
			`Google\Chrome\Application\chrome.exe`,
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
