// +build !windows

package launcher

import "github.com/ramr/go-reaper"

var reaperRunning = false

func runReaper() {
	if reaperRunning {
		return
	}

	reaperRunning = true

	go reaper.Reap()
}
