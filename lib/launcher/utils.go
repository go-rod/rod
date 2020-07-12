package launcher

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ysmood/kit"
)

type progresser struct {
	size  int
	count int
	r     io.Reader
	log   func(string)
	last  time.Time
}

func (p *progresser) Read(buf []byte) (n int, err error) {
	n, err = p.r.Read(buf)
	if err == io.EOF {
		p.log("\r\n")
		return
	}

	p.count += n

	if time.Since(p.last) < time.Second {
		return
	}

	p.last = time.Now()
	p.log(fmt.Sprintf("%02d%% ", p.count*100/p.size))
	return
}

func toHTTP(u *url.URL) {
	if u.Scheme == "ws" {
		u.Scheme = "http"
	} else if u.Scheme == "wss" {
		u.Scheme = "https"
	}
}

func unzip(from, to string) error {
	zr, err := zip.OpenReader(from)
	if err != nil {
		return err
	}

	err = kit.Mkdir(to, nil)
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		p := filepath.Join(to, f.Name)

		if f.FileInfo().IsDir() {
			err := os.Mkdir(p, f.Mode())
			if err != nil {
				return err
			}
			continue
		}

		r, err := f.Open()
		if err != nil {
			return err
		}

		data, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, f.Mode())
		if err != nil {
			return err
		}

		_, err = dst.Write(data)
		if err != nil {
			return err
		}

		err = dst.Close()
		if err != nil {
			return err
		}
	}

	return zr.Close()
}
