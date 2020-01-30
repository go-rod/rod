package rod_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
)

// S test suite
type S struct {
	suite.Suite
	browser *rod.Browser
	page    *rod.Page
}

// get abs file path from fixtures folder, return sample "file:///a/b/click.html"
func (s *S) htmlFile(path string) string {
	f, err := filepath.Abs(filepath.FromSlash(path))
	kit.E(err)
	return "file://" + f
}

func Test(t *testing.T) {
	slowmotion, _ := time.ParseDuration(os.Getenv("slowmotion"))

	s := new(S)
	s.browser = rod.Open(&rod.Browser{
		ControlURL: os.Getenv("chrome"),
		Foreground: os.Getenv("headless") == "false",
		Slowmotion: slowmotion,
	})
	defer s.browser.Close()

	s.page = s.browser.Page("about:blank")

	suite.Run(t, s)
}
