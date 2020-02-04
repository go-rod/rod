package rod_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/cdp"
)

// S test suite
type S struct {
	suite.Suite
	browser *rod.Browser
	page    *rod.Page
}

// get abs file path from fixtures folder, return sample "file:///a/b/click.html"
func (s *S) htmlFile(path string) string {
	return "file://" + s.file(path)
}

// get abs file path from fixtures folder, return sample "/a/b/click.html"
func (s *S) file(path string) string {
	f, err := filepath.Abs(filepath.FromSlash(path))
	kit.E(err)
	return f
}

func Test(t *testing.T) {
	slowmotion, _ := time.ParseDuration(os.Getenv("slow"))
	show := os.Getenv("show") == "true"

	s := new(S)
	s.browser = rod.Open(&rod.Browser{
		ControlURL: os.Getenv("chrome"),
		Foreground: show,
		Slowmotion: slowmotion,
		Trace:      true,
	})
	defer s.browser.Close()

	if show {
		go func() {
			for e := range s.browser.Event().Subscribe() {
				msg := e.(*cdp.Message)
				kit.Log(msg.Method, kit.MustToJSON(msg.Params))
			}
		}()
	}

	s.page = s.browser.Page(s.htmlFile("fixtures/click.html"))

	suite.Run(t, s)
}
