package rod_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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
	mockClient *MockClient
	browser    *rod.Browser
	page       *rod.Page
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
	s.mockClient = newMockClient(s, cdp.New(u))
	s.browser = rod.New().ControlURL("").
		Client(s.mockClient).
		MustConnect().
		DefaultViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: 800, Height: 600, DeviceScaleFactor: 1,
		}).Sleeper(rod.DefaultSleeper)

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

type MockRoundTripper struct {
	res *http.Response
	err error
}

func (mrt *MockRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return mrt.res, mrt.err
}

type MockReader struct {
	err error
}

func (mr *MockReader) Read(p []byte) (n int, err error) {
	return 0, mr.err
}

type Call func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error)

var _ rod.Client = &MockClient{}

type MockClient struct {
	sync.RWMutex
	suit      *S
	principal *cdp.Client
	call      Call
}

func newMockClient(s *S, c *cdp.Client) *MockClient {
	return &MockClient{suit: s, principal: c}
}

func (c *MockClient) Connect(ctx context.Context) error {
	return c.principal.Connect(ctx)
}

func (c *MockClient) Event() <-chan *cdp.Event {
	return c.principal.Event()
}

func (c *MockClient) Call(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
	return c.getCall()(ctx, sessionID, method, params)
}

func (c *MockClient) getCall() Call {
	c.RLock()
	defer c.RUnlock()

	if c.call == nil {
		return c.principal.Call
	}
	return c.call
}

func (c *MockClient) setCall(fn Call) {
	c.Lock()
	defer c.Unlock()

	if c.call != nil {
		c.suit.T().Fatal("forget to call the cleanup function of previous mock")
	}
	c.call = fn
}

func (c *MockClient) resetCall() {
	c.Lock()
	defer c.Unlock()
	c.call = nil
}

// Use it to find out which cdp call to intercept. Put a special like log.Println("*****") after the cdp call you want to intercept.
// The output of the test should has something like:
//
//     [countCall] 1, proto.DOMResolveNode{}
//     [countCall] 1, proto.RuntimeCallFunctionOn{}
//     [countCall] 2, proto.RuntimeCallFunctionOn{}
//     01:49:43 *****
//
// So the 3rd call is the one we want to intercept, then you can use the output with s.at or s.errorAt.
func (s *S) countCall() {
	l := sync.Mutex{}
	mCount := map[string]int{}

	s.mockClient.setCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		l.Lock()
		mCount[method]++
		m := fmt.Sprintf("%d, proto.%s{}", mCount[method], proto.GetType(method).Name())
		utils.E(fmt.Fprintln(os.Stdout, "[countCall]", m))
		l.Unlock()

		return s.mockClient.principal.Call(ctx, sessionID, method, params)
	})
}

// When call the cdp.Client.Call the nth time use fn instead.
// Use p to filter method.
func (s *S) at(nth int, p proto.Payload, fn func(send func() ([]byte, error)) ([]byte, error)) {
	if p == nil {
		s.T().Fatal("p must be specified")
	}

	count := int64(0)

	s.mockClient.setCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		if method == p.MethodName() {
			c := atomic.AddInt64(&count, 1)
			if c == int64(nth) {

				s.mockClient.resetCall()
				return fn(func() ([]byte, error) {
					return s.mockClient.principal.Call(ctx, sessionID, method, params)
				})
			}
		}
		return s.mockClient.principal.Call(ctx, sessionID, method, params)
	})
}

// When call the cdp.Client.Call the nth time return error.
// Use p to filter method.
func (s *S) errorAt(nth int, p proto.Payload) {
	s.at(nth, p, func(send func() ([]byte, error)) ([]byte, error) {
		return nil, errors.New("mock error")
	})
}
