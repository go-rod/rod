// Example run:
// go test -bench . ./lib/benchmark

package main_test

import (
	"path/filepath"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

func BenchmarkCleanup(b *testing.B) {
	u := got.New(b).Serve().Route("/", "", "page body").URL("/")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			launch := launcher.New().UserDataDir(filepath.Join("tmp", "cleanup", utils.RandString(8)))
			b.Cleanup(launch.Cleanup)

			url := launch.MustLaunch()

			browser := rod.New().ControlURL(url).MustConnect()
			b.Cleanup(browser.MustClose)

			browser.MustPage(u).MustClose()
		}
	})
}
