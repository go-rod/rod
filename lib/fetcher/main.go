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

const defaultRevision = 706915

// Chrome is a smart helper to get the executable chrome binary.
// It will first try to find the chrome from local disk, if not exists
// it will try to download the chromium under "./chrome/".
type Chrome struct {
	Host     string
	Revision int
}

func (c *Chrome) dir() string {
	return filepath.Join(os.TempDir(), "cdp")
}

func (c *Chrome) revision() int {
	if c.Revision == 0 {
		return defaultRevision
	}
	return c.Revision
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
	if c.Host == "" {
		c.Host = "https://storage.googleapis.com"
	}

	conf := map[string]struct {
		zipName   string
		urlPrefix string
	}{
		"darwin":  {"chrome-mac.zip", "Mac"},
		"linux":   {"chrome-linux.zip", "Linux_x64"},
		"windows": {"chrome-win.zip", "Win"},
	}[runtime.GOOS]

	u := fmt.Sprintf("%s/chromium-browser-snapshots/%s/%d/%s", c.Host, conf.urlPrefix, c.revision(), conf.zipName)
	kit.Log("Download chromium from:", u)

	zipPath := filepath.Join(c.dir(), fmt.Sprintf("chromium-%d.zip", c.revision()))

	err := kit.OutputFile(zipPath, "", nil)
	if err != nil {
		return err
	}

	zipFile, err := os.OpenFile(zipPath, os.O_WRONLY, os.ModePerm)
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

	unzipPath := fmt.Sprintf("chromium-%d", c.revision())

	err = os.RemoveAll(unzipPath)
	if err != nil {
		return err
	}

	return archiver.Unarchive(zipPath, filepath.Join(c.dir(), unzipPath))
}

// Get tries to find chrome binary depends the OS
func (c *Chrome) Get() (string, error) {
	execPath := c.ExecPath()

	dict := map[string][]string{
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

	list := append([]string{os.Getenv("CHROME_BIN")}, dict[runtime.GOOS]...)
	list = append(list, execPath)

	for _, path := range list {
		found, err := exec.LookPath(path)
		if err == nil {
			return found, nil
		}
	}
	return execPath, c.Download()
}
