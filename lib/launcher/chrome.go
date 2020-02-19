package launcher

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/mholt/archiver"
	"github.com/ysmood/kit"
)

// Chrome to smartly launch chrome
type Chrome struct {
	// Host default is https://storage.googleapis.com
	Host string

	// Revision of the chrome to use
	Revision int

	// Dir default is the filepath.Join(os.TempDir(), "cdp")
	Dir string

	// Log to print output
	Log func(string)

	// ErrInjector for testing
	ErrInjector *kit.ErrInjector
}

// NewChrome with default values
func NewChrome() *Chrome {
	return &Chrome{
		Revision: 722234,
		Host:     "https://storage.googleapis.com",
		Dir:      filepath.Join(os.TempDir(), "cdp"),
		Log: func(str string) {
			fmt.Print(str)
		},
		ErrInjector: &kit.ErrInjector{},
	}
}

// ExecPath of the chromium executable
func (lc *Chrome) ExecPath() string {
	bin := map[string]string{
		"darwin":  fmt.Sprintf("chromium-%d/chrome-mac/Chromium.app/Contents/MacOS/Chromium", lc.Revision),
		"linux":   fmt.Sprintf("chromium-%d/chrome-linux/chrome", lc.Revision),
		"windows": fmt.Sprintf("chromium-%d/chrome-win/chrome.exe", lc.Revision),
	}[runtime.GOOS]

	return filepath.Join(lc.Dir, bin)
}

// Download chromium
func (lc *Chrome) Download() error {
	conf := map[string]struct {
		zipName   string
		urlPrefix string
	}{
		"darwin":  {"chrome-mac.zip", "Mac"},
		"linux":   {"chrome-linux.zip", "Linux_x64"},
		"windows": {"chrome-win.zip", "Win"},
	}[runtime.GOOS]

	u := fmt.Sprintf("%s/chromium-browser-snapshots/%s/%d/%s", lc.Host, conf.urlPrefix, lc.Revision, conf.zipName)
	lc.Log("[rod/lib/launcher] Download chromium from: " + u + "\n[rod/lib/launcher] ")

	zipPath := filepath.Join(lc.Dir, fmt.Sprintf("chromium-%d.zip", lc.Revision))

	err := kit.Mkdir(lc.Dir, nil)
	err = lc.ErrInjector.E(err)
	if err != nil {
		return err
	}

	zipFile, err := os.OpenFile(zipPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	err = lc.ErrInjector.E(err)
	if err != nil {
		return err
	}

	res, err := kit.Req(u).Response()
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

// Get is a smart helper to get the executable chrome binary.
// It will first try to find the chrome from local disk, if not exists
// it will try to download the chromium to Dir.
func (lc *Chrome) Get() (string, error) {
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
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	},
}
