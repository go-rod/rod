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
	"github.com/ysmood/got"
	"github.com/ysmood/gotrace/pkg/testleak"
	"github.com/ysmood/gson"
	"github.com/ysmood/leakless"
)

// entry point for all tests
func Test(t *testing.T) {
	testleak.Check(t, 0)

	// make sure executables are ready
	leakless.GetLeaklessBin()
	utils.E(launcher.NewBrowser().Get())

	got.Each(t, newCasePool(t).get)
}

// context of a test case
type C struct {
	got.Assertion

	mc      *MockClient
	browser *rod.Browser
	page    *rod.Page
}

// context pool for tests
type ContextPool struct {
	list   chan C
	logger *log.Logger
}

func newCasePool(t *testing.T) ContextPool {
	parallel := 1
	if testing.Short() {
		parallel = runtime.GOMAXPROCS(0)
		fmt.Println("parallel test:", parallel)
	}

	logName := fmt.Sprintf("[%s]test_cdp.log", time.Now().Local().Format("01-02_15:04:05"))
	lf, _ := os.Create(filepath.Join("tmp", logName))

	cp := ContextPool{
		list:   make(chan C, parallel),
		logger: log.New(lf, "", log.Ltime),
	}

	t.Cleanup(func() {
		go func() {
			for i := 0; i < parallel; i++ {
				(<-cp.list).browser.MustClose()
			}
		}()
	})

	wg := &sync.WaitGroup{}
	wg.Add(parallel)
	for i := 0; i < parallel; i++ {
		go func() {
			cp.list <- cp.new()
			wg.Done()
		}()
	}
	wg.Wait()

	return cp
}

func (cp ContextPool) new() C {
	u := launcher.New().MustLaunch()

	mc := newMockClient(cdp.New(u).Logger(cp.logger))

	browser := rod.New().ControlURL("").Client(mc).MustConnect().
		MustIgnoreCertErrors(false).
		DefaultViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: 800, Height: 600, DeviceScaleFactor: 1,
		})

	page := getOnePage(browser)

	return C{
		mc:      mc,
		browser: browser,
		page:    page,
	}
}

// get a test context
func (cp ContextPool) get(t *testing.T) C {
	if testing.Short() {
		t.Parallel()
	}

	c := <-cp.list
	t.Cleanup(func() { cp.list <- c })

	if !testing.Short() {
		testleak.Check(t, 0)
	}

	t.Cleanup(func() {
		for _, p := range c.browser.MustPages() {
			if p.TargetID != c.page.TargetID {
				t.Fatal("leaking page: " + p.MustInfo().URL)
			}
		}

		c.mc.setCall(nil)
	})

	c.mc.t = t
	c.Assertion = got.New(t)

	return c
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

var _ rod.CDPClient = &MockClient{}

type MockClient struct {
	sync.RWMutex
	t         got.Testable
	principal *cdp.Client
	call      Call
	connect   func() error
	event     <-chan *cdp.Event
}

func newMockClient(c *cdp.Client) *MockClient {
	return &MockClient{principal: c}
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
		mc.t.Logf("leaking MockClient.stub")
		mc.t.FailNow()
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

type StubSend func() (gson.JSON, error)

// When call the cdp.Client.Call the nth time use fn instead.
// Use p to filter method.
func (mc *MockClient) stub(nth int, p proto.Payload, fn func(send StubSend) (gson.JSON, error)) {
	if p == nil {
		mc.t.Logf("p must be specified")
		mc.t.FailNow()
	}

	count := int64(0)

	mc.setCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		if method == p.ProtoName() {
			c := atomic.AddInt64(&count, 1)
			if int(c) == nth {
				mc.resetCall()
				j, err := fn(func() (gson.JSON, error) {
					b, err := mc.principal.Call(ctx, sessionID, method, params)
					return gson.New(b), err
				})
				if err != nil {
					return nil, err
				}
				return j.MarshalJSON()
			}
		}
		return mc.principal.Call(ctx, sessionID, method, params)
	})
}

// When call the cdp.Client.Call the nth time return error.
// Use p to filter method.
func (mc *MockClient) stubErr(nth int, p proto.Payload) {
	mc.stub(nth, p, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), errors.New("mock error")
	})
}

var slash = filepath.FromSlash
