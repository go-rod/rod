package launcher

import (
	"archive/zip"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

var inContainer = utils.InContainer

type progresser struct {
	size   int
	count  int
	logger utils.Logger
	last   time.Time
}

func (p *progresser) Write(b []byte) (n int, err error) {
	n = len(b)

	if p.count == 0 {
		p.logger.Println("Progress:")
	}

	p.count += n

	if p.count == p.size {
		p.logger.Println("100%")
		return
	}

	if time.Since(p.last) < time.Second {
		return
	}

	p.last = time.Now()
	p.logger.Println(fmt.Sprintf("%02d%%", p.count*100/p.size))

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

func unzip(logger utils.Logger, from, to string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	logger.Println("Unzip to:", to)

	zr, err := zip.OpenReader(from)
	utils.E(err)

	size := 0
	for _, f := range zr.File {
		size += int(f.FileInfo().Size())
	}

	progress := &progresser{size: size, logger: logger}

	for _, f := range zr.File {
		p := filepath.Join(to, f.Name)

		_ = utils.Mkdir(filepath.Dir(p))

		if f.FileInfo().IsDir() {
			err := os.Mkdir(p, f.Mode())
			utils.E(err)
			continue
		}

		r, err := f.Open()
		utils.E(err)

		dst, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Mode())
		utils.E(err)

		_, err = io.Copy(io.MultiWriter(dst, progress), r)
		utils.E(err)

		err = dst.Close()
		utils.E(err)
	}

	return zr.Close()
}

// certSPKI generates the SPKI of a certificate public key
// https://blog.afoolishmanifesto.com/posts/golang-self-signed-and-pinned-certs/
func certSPKI(pk crypto.PublicKey) ([]byte, error) {
	pubDER, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("x509.MarshalPKIXPublicKey: %w", err)
	}

	sum := sha256.Sum256(pubDER)
	pin := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(pin, sum[:])

	return pin, nil
}
