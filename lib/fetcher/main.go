package fetcher

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/cheggaaa/pb/v3"
	"github.com/mholt/archiver"
	"github.com/ysmood/kit"
)

// Revision is the default revision of chromium to use
const Revision = 722234

// Chrome is a smart helper to get the executable chrome binary.
// It will first try to find the chrome from local disk, if not exists
// it will try to download the chromium to Dir.
type Chrome struct {
	// Host default is https://storage.googleapis.com
	Host string

	// Revision default is DefaultRevision
	Revision int

	// Dir default is the filepath.Join(os.TempDir(), "cdp")
	Dir string
}

func (c *Chrome) dir() string {
	if c.Dir == "" {
		return filepath.Join(os.TempDir(), "cdp")
	}
	return c.Dir
}

func (c *Chrome) revision() int {
	if c.Revision == 0 {
		return Revision
	}
	return c.Revision
}

func (c *Chrome) host() string {
	if c.Host == "" {
		return "https://storage.googleapis.com"
	}
	return c.Host
}

// ExecPath of the chromium executable
func (c *Chrome) ExecPath() string {
	bin := map[string]string{
		"darwin":  fmt.Sprintf("chromium-%d/chrome-mac/Chromium.app/Contents/MacOS/Chromium", c.revision()),
		"linux":   fmt.Sprintf("chromium-%d/chrome-linux/chrome", c.revision()),
		"windows": fmt.Sprintf("chromium-%d/chrome-win/chrome.exe", c.revision()),
	}[runtime.GOOS]

	return filepath.Join(c.dir(), bin)
}

// Download chromium
func (c *Chrome) Download() error {
	host := c.host()
	revision := c.revision()
	dir := c.dir()

	conf := map[string]struct {
		zipName   string
		urlPrefix string
	}{
		"darwin":  {"chrome-mac.zip", "Mac"},
		"linux":   {"chrome-linux.zip", "Linux_x64"},
		"windows": {"chrome-win.zip", "Win"},
	}[runtime.GOOS]

	u := fmt.Sprintf("%s/chromium-browser-snapshots/%s/%d/%s", host, conf.urlPrefix, revision, conf.zipName)
	kit.Log("Download chromium from:", u)

	zipPath := filepath.Join(dir, fmt.Sprintf("chromium-%d.zip", revision))

	err := kit.Mkdir(dir, nil)
	if err != nil {
		return err
	}

	zipFile, err := os.OpenFile(zipPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
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

	bar := pb.StartNew(int(size))

	_, err = io.Copy(zipFile, bar.NewProxyReader(res.Body))
	if err != nil {
		return err
	}

	bar.Finish()

	err = zipFile.Close()
	if err != nil {
		return err
	}

	unzipPath := filepath.Join(dir, fmt.Sprintf("chromium-%d", revision))
	_ = os.RemoveAll(unzipPath)
	return archiver.Unarchive(zipPath, unzipPath)
}

// Get tries to find chrome binary depends the OS
func (c *Chrome) Get() (string, error) {
	execPath := c.ExecPath()

	list := append(downloadMap[runtime.GOOS], execPath)

	for _, path := range list {
		found, err := exec.LookPath(path)
		if err == nil {
			return found, nil
		}
	}
	return execPath, c.Download()
}

var downloadMap = map[string][]string{
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
