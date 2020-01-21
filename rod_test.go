package rod_test

import (
	"os"
	"path/filepath"
	"testing"

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
	f, err := filepath.Abs(filepath.Join("fixtures", path))
	kit.E(err)
	return "file://" + f
}

func Test(t *testing.T) {
	s := new(S)
	s.browser = rod.Open(&rod.Browser{Foreground: os.Getenv("headless") == "false"})
	defer s.browser.Close()

	s.page = s.browser.Page("about:blank")

	suite.Run(t, s)
}
