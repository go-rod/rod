package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/mholt/archiver"
	"github.com/ysmood/kit"
)

// HostGoogle to download browser
const HostGoogle = "https://storage.googleapis.com"

// HostChina to download browser
const HostChina = "https://npm.taobao.org/mirrors"

// Browser is a helper to download browser smartly
type Browser struct {
	Context context.Context

	// Hosts to download browser, examples:
	// https://storage.googleapis.com/chromium-browser-snapshots/Linux_x64/748030/chrome-linux.zip
	// https://storage.googleapis.com/chromium-browser-snapshots/Mac/748030/chrome-mac.zip
	// https://storage.googleapis.com/chromium-browser-snapshots/Win/748030/chrome-win.zip
	Hosts []string

	// Revision of the browser to use
	Revision int

	// Dir default is the filepath.Join(os.TempDir(), "rod")
	Dir string

	// Log to print output
	Log func(string)
}

// NewBrowser with default values
func NewBrowser() *Browser {
	return &Browser{
		Context:  context.Background(),
		Revision: DefaultRevision,
		Hosts:    []string{HostGoogle, HostChina},
		Dir:      filepath.Join(os.TempDir(), "rod"),
		Log: func(str string) {
			fmt.Print(str)
		},
	}
}

// ExecPath of the chromium executable
func (lc *Browser) ExecPath() string {
	bin := map[string]string{
		"darwin":  fmt.Sprintf("chromium-%d/chrome-mac/Chromium.app/Contents/MacOS/Chromium", lc.Revision),
		"linux":   fmt.Sprintf("chromium-%d/chrome-linux/chrome", lc.Revision),
		"windows": fmt.Sprintf("chromium-%d/chrome-win/chrome.exe", lc.Revision),
	}[runtime.GOOS]

	return filepath.Join(lc.Dir, bin)
}

// Download chromium
func (lc *Browser) Download() error {
	conf := map[string]struct {
		zipName   string
		urlPrefix string
	}{
		"darwin":  {"chrome-mac.zip", "Mac"},
		"linux":   {"chrome-linux.zip", "Linux_x64"},
		"windows": {"chrome-win.zip", "Win"},
	}[runtime.GOOS]

	for _, host := range lc.Hosts {
		u := fmt.Sprintf("%s/chromium-browser-snapshots/%s/%d/%s", host, conf.urlPrefix, lc.Revision, conf.zipName)
		err := lc.download(u)
		if err != nil {
			lc.Log("[rod/lib/launcher] " + err.Error())
			continue
		}
		return nil
	}
	return errors.New("[rod/lib/launcher] failed to download chrome")
}

func (lc *Browser) download(u string) error {
	lc.Log("[rod/lib/launcher] Download chromium from: " + u + "\n[rod/lib/launcher] ")

	zipPath := filepath.Join(lc.Dir, fmt.Sprintf("chromium-%d.zip", lc.Revision))

	err := kit.Mkdir(lc.Dir, nil)
	if err != nil {
		return err
	}

	zipFile, err := os.OpenFile(zipPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	res, err := kit.Req(u).Context(lc.Context).Client(&http.Client{
		Transport: &http.Transport{IdleConnTimeout: 30 * time.Second},
	}).Response()
	if err != nil {
		return err
	}

	size, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}

	progress := &progresser{
		size: int(size),
		r:    res.Body,
		log:  lc.Log,
	}

	_, err = io.Copy(zipFile, progress)
	if err != nil {
		return err
	}

	lc.Log("[rod/lib/launcher] Download chromium complete: " + zipPath + "\n")

	err = zipFile.Close()
	if err != nil {
		return err
	}

	unzipPath := filepath.Join(lc.Dir, fmt.Sprintf("chromium-%d", lc.Revision))
	_ = os.RemoveAll(unzipPath)
	err = archiver.Unarchive(zipPath, unzipPath)
	if err != nil {
		return err
	}
	lc.Log("[rod/lib/launcher] Unzipped chromium bin to: " + lc.ExecPath() + "\n")
	return nil
}

// Get is a smart helper to get the browser executable binary.
// It will first try to find the browser from local disk, if not exists
// it will try to download the chromium to Dir.
func (lc *Browser) Get() (string, error) {
	execPath := lc.ExecPath()

	list := append(execSearchMap[runtime.GOOS], execPath)

	for _, path := range list {
		found, err := exec.LookPath(path)
		if err == nil {
			return found, nil
		}
	}

	return execPath, lc.Download()
}

var execSearchMap = map[string][]string{
	"darwin": {
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	},
	"linux": {
		"chromium",
		"chromium-browser",
		"google-chrome",
		"/usr/bin/google-chrome",
	},
	"windows": {
		"chrome",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	},
}
