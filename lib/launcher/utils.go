package launcher

import (
	"fmt"
	"io"
	"time"

	"github.com/ramr/go-reaper"
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

var reaperRunning = false

func runReaper() {
	if reaperRunning {
		return
	}

	reaperRunning = true

	go reaper.Reap()
}
