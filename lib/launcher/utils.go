package launcher

import (
	"fmt"
	"io"
	"net/url"
	"time"
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
