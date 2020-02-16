package rod_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/launcher"
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
	slow, _ := time.ParseDuration(os.Getenv("slow"))
	show := os.Getenv("show") == "true"

	url := launcher.New().Headless(!show).Launch()

	s := new(S)
	s.browser = rod.New().
		ControlURL(url).
		Slowmotion(slow).
		Trace(true).
		Viewport(nil).
		Connect()

	defer s.browser.Close()

	s.page = s.browser.Page(s.htmlFile("fixtures/click.html"))

	suite.Run(t, s)
}
