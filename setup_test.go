package rod_test

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

var slash = filepath.FromSlash

// S test suite
type S struct {
	suite.Suite
	client  *cdp.Client
	browser *rod.Browser
	page    *rod.Page
}

func init() {
	log.SetFlags(log.Ltime)
}

func TestMain(m *testing.M) {
	// to prevent false positive of goleak
	http.DefaultClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	goleak.VerifyTestMain(
		m,
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
		goleak.IgnoreTopFunction("net/http.(*persistConn).writeLoop"),
		goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreTopFunction("github.com/ramr/go-reaper.sigChildHandler"),
		goleak.IgnoreTopFunction("github.com/ramr/go-reaper.reapChildren"),
	)
}

func Test(t *testing.T) {
	extPath, err := filepath.Abs("fixtures/chrome-extension")
	utils.E(err)

	u := launcher.New().
		Delete("disable-extensions").
		Set("load-extension", extPath).
		MustLaunch()

	s := new(S)
	s.client = cdp.New(u)
	s.browser = rod.New().ControlURL("").Client(s.client).MustConnect().
		DefaultViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: 800, Height: 600, DeviceScaleFactor: 1,
		})

	defer s.browser.MustClose()

	s.page = s.browser.MustPage("")

	suite.Run(t, s)
}

// get abs file path from fixtures folder, return sample "file:///a/b/click.html"
func srcFile(path string) string {
	return "file://" + file(path)
}

// get abs file path from fixtures folder, return sample "/a/b/click.html"
func file(path string) string {
	f, err := filepath.Abs(slash(path))
	utils.E(err)
	return f
}

func httpHTML(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		utils.E(w.Write([]byte(body)))
	}
}

func httpString(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.E(w.Write([]byte(body)))
	}
}

func httpHTMLFile(path string) http.HandlerFunc {
	body, err := utils.ReadString(path)
	utils.E(err)
	return httpHTML(body)
}

func serveStatic() (string, func()) {
	u, mux, close := utils.Serve("")
	mux.Handle("/fixtures", http.FileServer(http.Dir("fixtures")))

	return u + "/", close
}

func (s *S) countCall() {
	count := 0
	s.browser.CDPCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		count++
		log.Println("[call count]", count)
		return s.client.Call(ctx, sessionID, method, params)
	})
}

// when call the cdp.Client.Call the nth time use fn instead
func (s *S) at(n int, fn func([]byte, error) ([]byte, error)) (recover func()) {
	count := 0
	s.browser.CDPCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		res, err := s.client.Call(ctx, sessionID, method, params)
		count++
		if count == n {
			return fn(res, err)
		}
		return res, err
	})
	cancel := preventLeak()

	return func() {
		s.browser.CDPCall(nil)
		cancel()
	}
}

// when call the cdp.Client.Call the nth time return error
func (s *S) errorAt(n int, err error) (recover func()) {
	if err == nil {
		err = errors.New("")
	}
	count := 0
	s.browser.CDPCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		count++
		if count == n {
			return nil, err
		}
		return s.client.Call(ctx, sessionID, method, params)
	})

	cancel := preventLeak()

	return func() {
		s.browser.CDPCall(nil)
		cancel()
	}
}

func preventLeak() func() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done() // go.uber.org/goleak will report error if it's not released
	}()
	return cancel
}
