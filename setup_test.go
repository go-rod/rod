package rod_test

import (
	"context"
	"errors"
	"flag"
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
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
	"github.com/ysmood/gotrace/pkg/testleak"
	"github.com/ysmood/gson"
)

var TimeoutEach = flag.Duration("timeout-each", time.Minute, "timeout for each test")

var LogDir = slash(fmt.Sprintf("tmp/cdp-log/%s", time.Now().Format("2006-01-02_15-04-05")))

func init() {
	got.DefaultFlags("timeout=5m", "run=/")

	utils.E(os.MkdirAll(slash("tmp/cdp-log"), 0755))
}

// entry point for all tests
func Test(t *testing.T) {
	testleak.Check(t, 0)

	got.Each(t, newTesterPool(t).get)
}

// T is a tester. Testers are thread-safe, they shouldn't race each other.
type T struct {
	got.G

	mc      *MockClient
	browser *rod.Browser
	page    *rod.Page
}

type TesterPool chan *T

func newTesterPool(t *testing.T) TesterPool {
	parallel := got.Parallel()
	if parallel == 0 {
		parallel = runtime.GOMAXPROCS(0)
	}
	fmt.Println("parallel test", parallel)

	cp := TesterPool(make(chan *T, parallel))

	t.Cleanup(func() {
		go func() {
			for i := 0; i < parallel; i++ {
				if t := <-cp; t != nil {
					t.browser.MustClose()
				}
			}
		}()
	})

	for i := 0; i < parallel; i++ {
		cp <- nil
	}

	return cp
}

// new tester
func (cp TesterPool) new() *T {
	u := launcher.New().MustLaunch()

	mc := newMockClient(u)

	browser := rod.New().ControlURL("").Client(mc).MustConnect().
		MustIgnoreCertErrors(false).
		DefaultDevice(devices.Test)

	page := browser.MustPage("")

	return &T{
		mc:      mc,
		browser: browser,
		page:    page,
	}
}

// get a tester
func (cp TesterPool) get(t *testing.T) T {
	parallel := got.Parallel() != 1
	if parallel {
		t.Parallel()
	}

	tester := <-cp
	if tester == nil {
		tester = cp.new()
	}
	t.Cleanup(func() { cp <- tester })

	tester.G = got.New(t)
	tester.mc.t = t
	tester.mc.log.SetOutput(tester.Open(true, LogDir, tester.mc.id, t.Name()[5:]+".log"))

	tester.checkLeaking(!parallel)
	tester.PanicAfter(*TimeoutEach)

	return *tester
}

func (t T) enableCDPLog() {
	t.mc.principal.Logger(rod.DefaultLogger)
}

func (t T) dump(args ...interface{}) {
	t.Log(utils.Dump(args))
}

func (t T) blank() string {
	return t.srcFile("./fixtures/blank.html")
}

// get abs file path from fixtures folder, return sample "file:///a/b/click.html"
func (t T) srcFile(path string) string {
	t.Helper()
	f, err := filepath.Abs(slash(path))
	t.E(err)
	return "file://" + f
}

func (t T) newPage(u string) *rod.Page {
	t.Helper()
	p, err := t.browser.Page(proto.TargetCreateTarget{URL: u})
	t.E(err)
	t.Cleanup(func() {
		if !t.Failed() {
			p.MustClose()
		}
	})
	return p
}

func (t T) checkLeaking(checkGoroutine bool) {
	if checkGoroutine {
		testleak.Check(t.Testable.(*testing.T), 0)
	}

	t.Cleanup(func() {
		if t.Failed() {
			return
		}

		for _, p := range t.browser.MustPages() {
			if p.TargetID != t.page.TargetID {
				t.Logf("leaking page: %#v", p.MustInfo())
				t.FailNow()
			}
		}

		if t.browser.LoadState(t.page.SessionID, proto.FetchEnable{}) {
			t.Logf("leaking FetchEnable")
			t.FailNow()
		}

		t.mc.setCall(nil)
	})
}

type Call func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error)

var _ rod.CDPClient = &MockClient{}

type MockClient struct {
	sync.RWMutex
	id        string
	t         got.Testable
	log       *log.Logger
	principal *cdp.Client
	call      Call
	connect   func() error
	event     <-chan *cdp.Event
}

var mockClientCount int32

func newMockClient(u string) *MockClient {
	id := fmt.Sprintf("%02d", atomic.AddInt32(&mockClientCount, 1))

	// create init log file
	utils.E(os.MkdirAll(filepath.Join(LogDir, id), 0755))
	f, err := os.Create(filepath.Join(LogDir, id, "_.log"))
	log := log.New(f, "", log.Ltime)
	utils.E(err)

	client := cdp.New(u).Logger(utils.MultiLogger(defaults.CDP, log))

	return &MockClient{id: id, principal: client, log: log}
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
func (mc *MockClient) stub(nth int, p proto.Request, fn func(send StubSend) (gson.JSON, error)) {
	if p == nil {
		mc.t.Logf("p must be specified")
		mc.t.FailNow()
	}

	count := int64(0)

	mc.setCall(func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		if method == p.ProtoReq() {
			if int(atomic.AddInt64(&count, 1)) == nth {
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
func (mc *MockClient) stubErr(nth int, p proto.Request) {
	mc.stub(nth, p, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), errors.New("mock error")
	})
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

func TestLintIgnore(t *testing.T) {
	_ = rod.Try(func() {
		tt := T{}
		tt.dump()
		tt.enableCDPLog()

		mc := &MockClient{}
		mc.stubCounter()
	})
}

var slash = filepath.FromSlash
