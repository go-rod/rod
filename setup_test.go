package rod_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	mc           *MockClient
	browser      *rod.Browser
	page         *rod.Page
	goleakIgnore goleak.Option
}

func TestMain(m *testing.M) {
	log.SetFlags(log.Ltime)

	// to prevent false positive of goleak
	http.DefaultClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	goleak.VerifyTestMain(m, goleak.MaxRetry(3*time.Second))
}

func Test(t *testing.T) {
	extPath, err := filepath.Abs("fixtures/chrome-extension")
	utils.E(err)

	u := launcher.New().
		Delete("disable-extensions").
		Set("load-extension", extPath).
		MustLaunch()

	s := new(S)
	s.mc = newMockClient(s, cdp.New(u))
	s.browser = rod.New().ControlURL("").
		Client(s.mc).
		MustConnect().
		DefaultViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: 800, Height: 600, DeviceScaleFactor: 1,
		}).Sleeper(rod.DefaultSleeper)

	defer s.browser.MustClose()

	s.browser.MustIgnoreCertErrors(false)
	s.page = getOnePage(s.browser)

	s.goleakIgnore = goleak.IgnoreCurrent()

	suite.Run(t, s)
}

func (s *S) TearDownTest() {
	for _, p := range s.browser.MustPages() {
		if p.TargetID != s.page.TargetID {
			s.T().Fatal("leaking page: " + p.MustInfo().URL)
		}
	}

	goleak.VerifyNone(
		s.T(),
		goleak.MaxRetry(3*time.Second),
		s.goleakIgnore,
	)
}

func getOnePage(b *rod.Browser) (page *rod.Page) {
	for i := 0; i < 50; i++ {
		page = b.MustPages().First()
		if page != nil {
			return
		}
		utils.Sleep(0.1)
	}

	// TODO: I don't know why sometimes windows don't have the init page
	if runtime.GOOS == "windows" {
		page = b.MustPage("")
	}

	return
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
	connect   func() error
	event     <-chan *cdp.Event
}

func newMockClient(s *S, c *cdp.Client) *MockClient {
	return &MockClient{suit: s, principal: c}
}

func (mc *MockClient) Connect(ctx context.Context) error {
	if mc.connect != nil {
		return mc.connect()
	}
	return mc.principal.Connect(ctx)
}

func (mc *MockClient) Event() <-chan *cdp.Event {
	if mc.event != nil {
		return mc.event
	}
	return mc.principal.Event()
}

func (mc *MockClient) Call(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
	return mc.getCall()(ctx, sessionID, method, params)
}

func (mc *MockClient) getCall() Call {
	mc.RLock()
	defer mc.RUnlock()

	if mc.call == nil {
		return mc.principal.Call
	}
	return mc.call
}

func (mc *MockClient) setCall(fn Call) {
	mc.Lock()
	defer mc.Unlock()

	if mc.call != nil {
		mc.suit.T().Fatal("forget to call the cleanup function of previous mock")
	}
	mc.call = fn
}

func (mc *MockClient) resetCall() {
	mc.Lock()
	defer mc.Unlock()
	mc.call = nil
}

// Use it to find out which cdp call to intercept. Put a special like log.Println("*****") after the cdp call you want to intercept.
// The output of the test should has something like:
//
//     [stubCounter] 1, proto.DOMResolveNode{}
//     [stubCounter] 1, proto.RuntimeCallFunctionOn{}
//     [stubCounter] 2, proto.RuntimeCallFunctionOn{}
//     01:49:43 *****
//
// So the 3rd call is the one we want to intercept, then you can use the output with s.at or s.errorAt.
func (mc *MockClient) stubCounter() {
	l := sync.Mutex{}
	mCount := map[string]int{}

	mc.setCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		l.Lock()
		mCount[method]++
		m := fmt.Sprintf("%d, proto.%s{}", mCount[method], proto.GetType(method).Name())
		_, _ = fmt.Fprintln(os.Stdout, "[stubCounter]", m)
		l.Unlock()

		return mc.principal.Call(ctx, sessionID, method, params)
	})
}

// When call the cdp.Client.Call the nth time use fn instead.
// Use p to filter method.
func (mc *MockClient) stub(nth int, p proto.Payload, fn func(send func() ([]byte, error)) ([]byte, error)) {
	if p == nil {
		mc.suit.T().Fatal("p must be specified")
	}

	count := int64(0)

	mc.setCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		if method == p.MethodName() {
			c := atomic.AddInt64(&count, 1)
			if c == int64(nth) {

				mc.resetCall()
				return fn(func() ([]byte, error) {
					return mc.principal.Call(ctx, sessionID, method, params)
				})
			}
		}
		return mc.principal.Call(ctx, sessionID, method, params)
	})
}

// When call the cdp.Client.Call the nth time return error.
// Use p to filter method.
func (mc *MockClient) stubErr(nth int, p proto.Payload) {
	mc.stub(nth, p, func(send func() ([]byte, error)) ([]byte, error) {
		return nil, errors.New("mock error")
	})
}
