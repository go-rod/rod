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
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
	"github.com/ysmood/gotrace"
	"github.com/ysmood/gson"
)

var TimeoutEach = flag.Duration("timeout-each", time.Minute, "timeout for each test")

var LogDir = slash(fmt.Sprintf("tmp/cdp-log/%s", time.Now().Format("2006-01-02_15-04-05")))

func init() {
	got.DefaultFlags("timeout=5m", "run=/")

	utils.E(os.MkdirAll(slash("tmp/cdp-log"), 0o755))

	launcher.NewBrowser().MustGet() // preload browser to local
}

var testerPool TesterPool

func TestMain(m *testing.M) {
	testerPool = newTesterPool()

	code := m.Run()
	if code != 0 {
		os.Exit(code)
	}

	testerPool.cleanup()

	if err := gotrace.Check(0, gotrace.IgnoreFuncs("internal/poll.runtime_pollWait")); err != nil {
		log.Fatal(err)
	}
}

var setup = func(t *testing.T) G {
	return testerPool.get(t)
}

// G is a tester. Testers are thread-safe, they shouldn't race each other.
type G struct {
	got.G

	// mock client for proxy the cdp requests
	mc *MockClient

	// a random browser instance from the pool. If you have changed state of it, you must reset it
	// or it may affect other test cases.
	browser *rod.Browser

	// a random page instance from the pool. If you have changed state of it, you must reset it
	// or it may affect other test cases.
	page *rod.Page

	// use it to cancel the TimeoutEach for each test case
	cancelTimeout func()
}

// TesterPool if we don't use pool to cache, the total time will be much longer.
type TesterPool struct {
	pool     chan *G
	parallel int
}

func newTesterPool() TesterPool {
	parallel := got.Parallel()
	if parallel == 0 {
		parallel = runtime.GOMAXPROCS(0)
	}
	fmt.Println("parallel test", parallel)

	cp := TesterPool{
		pool:     make(chan *G, parallel),
		parallel: parallel,
	}

	for i := 0; i < parallel; i++ {
		cp.pool <- nil
	}

	return cp
}

// new tester
func (tp TesterPool) new() *G {
	u := launcher.New().Set("proxy-bypass-list", "<-loopback>").MustLaunch()

	mc := newMockClient(u)

	browser := rod.New().Client(mc).MustConnect().MustIgnoreCertErrors(false)

	pages := browser.MustPages()

	var page *rod.Page
	if pages.Empty() {
		page = browser.MustPage()
	} else {
		page = pages.First()
	}

	return &G{
		mc:      mc,
		browser: browser,
		page:    page,
	}
}

// get a tester
func (tp TesterPool) get(t *testing.T) G {
	if got.Parallel() != 1 {
		t.Parallel()
	}

	tester := <-tp.pool
	if tester == nil {
		tester = tp.new()
	}
	t.Cleanup(func() { tp.pool <- tester })

	tester.G = got.New(t)
	tester.mc.t = t
	tester.mc.log.SetOutput(tester.Open(true, filepath.Join(LogDir, tester.mc.id, t.Name()+".log")))

	tester.checkLeaking()

	return *tester
}

func (tp TesterPool) cleanup() {
	for i := 0; i < tp.parallel; i++ {
		if t := <-testerPool.pool; t != nil {
			t.browser.MustClose()
		}
	}
}

func (g G) enableCDPLog() {
	g.mc.principal.Logger(rod.DefaultLogger)
}

func (g G) dump(args ...interface{}) {
	g.Log(utils.Dump(args))
}

func (g G) blank() string {
	return g.srcFile("./fixtures/blank.html")
}

func (g G) html(content string) string {
	return g.Serve().Route("/", "", content).URL()
}

// Get abs file path from fixtures folder, such as "file:///a/b/click.html".
// Usually the path can be used for html src attribute like:
//
//	<img src="file:///a/b">
func (g G) srcFile(path string) string {
	g.Helper()
	f, err := filepath.Abs(slash(path))
	g.E(err)
	return "file://" + f
}

func (g G) newPage(u ...string) *rod.Page {
	g.Helper()
	p := g.browser.MustPage(u...)
	g.Cleanup(func() {
		if !g.Failed() {
			p.MustClose()
		}
	})
	return p
}

func (g *G) checkLeaking() {
	ig := gotrace.CombineIgnores(gotrace.IgnoreCurrent(), gotrace.IgnoreNonChildren())
	gotrace.CheckLeak(g.Testable, 0, ig)

	self := gotrace.Get(false)[0]
	g.cancelTimeout = g.DoAfter(*TimeoutEach, func() {
		t := gotrace.Get(true).Filter(func(t *gotrace.Trace) bool {
			if t.GoroutineID == self.GoroutineID {
				return false
			}
			return ig(t)
		}).String()
		panic(fmt.Sprintf(`[rod_test.TimeoutEach] %s timeout after %v
running goroutines: %s`, g.Name(), *TimeoutEach, t))
	})

	g.Cleanup(func() {
		if g.Failed() {
			return
		}

		// close all other pages other than g.page
		res, err := proto.TargetGetTargets{}.Call(g.browser)
		g.E(err)
		for _, info := range res.TargetInfos {
			if info.TargetID != g.page.TargetID {
				g.E(proto.TargetCloseTarget{TargetID: info.TargetID}.Call(g.browser))
			}
		}

		if g.browser.LoadState(g.page.SessionID, &proto.FetchEnable{}) {
			g.Logf("leaking FetchEnable")
			g.FailNow()
		}

		g.mc.setCall(nil)
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
	event     <-chan *cdp.Event
}

var mockClientCount int32

func newMockClient(u string) *MockClient {
	id := fmt.Sprintf("%02d", atomic.AddInt32(&mockClientCount, 1))

	// create init log file
	utils.E(os.MkdirAll(filepath.Join(LogDir, id), 0o755))
	f, err := os.Create(filepath.Join(LogDir, id, "_.log"))
	log := log.New(f, "", log.Ltime)
	utils.E(err)

	client := cdp.New().Logger(utils.MultiLogger(defaults.CDP, log)).Start(cdp.MustConnectWS(u))

	return &MockClient{id: id, principal: client, log: log}
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
		mc.t.Fail()
	}
	mc.call = fn
}

func (mc *MockClient) resetCall() {
	mc.Lock()
	defer mc.Unlock()
	mc.call = nil
}

// Use it to find out which cdp call to intercept. Put a print like log.Println("*****") after the cdp call you want to intercept.
// The output of the test should has something like:
//
//	[stubCounter] begin
//	[stubCounter] 1, proto.DOMResolveNode{}
//	[stubCounter] 1, proto.RuntimeCallFunctionOn{}
//	[stubCounter] 2, proto.RuntimeCallFunctionOn{}
//	01:49:43 *****
//
// So the 3rd call is the one we want to intercept, then you can use the output with s.at or s.errorAt.
func (mc *MockClient) stubCounter() {
	l := sync.Mutex{}
	mCount := map[string]int{}

	fmt.Fprintln(os.Stdout, "[stubCounter] begin")

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

func (mr *MockReader) Read(_ []byte) (n int, err error) {
	return 0, mr.err
}

func TestLintIgnore(t *testing.T) {
	t.Skip()

	_ = rod.Try(func() {
		tt := G{}
		tt.dump()
		tt.enableCDPLog()

		mc := &MockClient{}
		mc.stubCounter()
	})
}

var slash = filepath.FromSlash
