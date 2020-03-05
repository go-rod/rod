package rod_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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
func srcFile(path string) string {
	return "file://" + file(path)
}

// get abs file path from fixtures folder, return sample "/a/b/click.html"
func file(path string) string {
	f, err := filepath.Abs(filepath.FromSlash(path))
	kit.E(err)
	return f
}

func ginHTML(body string) gin.HandlerFunc {
	return func(ctx kit.GinContext) {
		ctx.Header("Content-Type", "text/html;")
		kit.E(ctx.Writer.Write([]byte(body)))
	}
}

func Test(t *testing.T) {
	slow, _ := time.ParseDuration(os.Getenv("slow"))
	show := os.Getenv("show") == "true"
	debugCDP := os.Getenv("debug_cdp") == "true"

	url := launcher.New().
		Headless(!show).
		Log(func(s string) { kit.E(os.Stdout.WriteString(s)) }).
		Launch()

	s := new(S)
	s.browser = rod.New().
		ControlURL(url).
		DebugCDP(debugCDP).
		Slowmotion(slow).
		Trace(true).
		Viewport(nil).
		Connect()

	defer s.browser.Close()

	s.page = s.browser.Page(srcFile("fixtures/click.html"))

	suite.Run(t, s)
}
