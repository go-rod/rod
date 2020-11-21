package launcher

import (
	"archive/zip"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

var isInDocker = utils.FileExists("/.dockerenv") || utils.FileExists("/.containerenv")

type progresser struct {
	size   int
	count  int
	logger io.Writer
	last   time.Time
}

func (p *progresser) Write(b []byte) (n int, err error) {
	n = len(b)

	if p.count == 0 {
		_, _ = fmt.Fprint(p.logger, "Progress:")
	}

	p.count += n

	if p.count == p.size {
		_, _ = fmt.Fprintln(p.logger, " 100%")
		return
	}

	if time.Since(p.last) < time.Second {
		return
	}

	p.last = time.Now()
	_, _ = fmt.Fprintf(p.logger, " %02d%%", p.count*100/p.size)

	return
}

func toHTTP(u url.URL) *url.URL {
	newURL := u
	if newURL.Scheme == "ws" {
		newURL.Scheme = "http"
	} else if newURL.Scheme == "wss" {
		newURL.Scheme = "https"
	}
	return &newURL
}

func toWS(u url.URL) *url.URL {
	newURL := u
	if newURL.Scheme == "http" {
		newURL.Scheme = "ws"
	} else if newURL.Scheme == "https" {
		newURL.Scheme = "wss"
	}
	return &newURL
}

func unzip(logger io.Writer, from, to string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	_, _ = fmt.Fprintln(logger, "Unzip to:", to)
	defer func() { _, _ = fmt.Fprintln(logger) }()

	zr, err := zip.OpenReader(from)
	utils.E(err)

	err = utils.Mkdir(to)
	utils.E(err)

	size := 0
	for _, f := range zr.File {
		size += int(f.FileInfo().Size())
	}

	progress := &progresser{size: size, logger: logger}

	for _, f := range zr.File {
		p := filepath.Join(to, f.Name)

		if f.FileInfo().IsDir() {
			err := os.Mkdir(p, f.Mode())
			utils.E(err)
			continue
		}

		r, err := f.Open()
		utils.E(err)

		dst, err := os.Create(p)
		utils.E(err)

		_, err = io.Copy(io.MultiWriter(dst, progress), r)
		utils.E(err)

		err = dst.Close()
		utils.E(err)
	}

	return zr.Close()
}

func getSelfClosePage() string {
	path := filepath.Join(os.TempDir(), "rod", "self-close.html")

	if !utils.FileExists(path) {
		_ = utils.OutputFile(path, "<script>close()</script>")
	}

	return path
}
