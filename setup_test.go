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
)

// entry point for all tests
func Test(t *testing.T) {
	testleak.Check(t, 0)

	got.Each(t, newTesterPool(t).get)
}

// T is a tester. Testers are thread-safe, they won't race each other.
// Usually, we only have one tester for the whole test. But if testing.Short() is true,
// we will create runtime.GOMAXPROCS(0) testers, each one holds a standalone browser.
type T struct {
	got.G

	mc      *MockClient
	browser *rod.Browser
	page    *rod.Page
}

type TesterPool struct {
	list   chan *T
	logger *log.Logger
}

func newTesterPool(t *testing.T) TesterPool {
	parallel := 1
	if testing.Short() {
		parallel = runtime.GOMAXPROCS(0)
		fmt.Println("parallel test:", parallel)
	}

	logName := fmt.Sprintf("%s test_cdp.log", time.Now().Local().Format("01-02_15-04-05"))
	lf := got.New(t).Open(true, "tmp", logName)

	cp := TesterPool{
		list:   make(chan *T, parallel),
		logger: log.New(lf, "", log.Ltime),
	}

	t.Cleanup(func() {
		go func() {
			for i := 0; i < parallel; i++ {
				(<-cp.list).browser.MustClose()
			}
		}()
	})

	for i := 0; i < parallel; i++ {
		cp.list <- nil
	}

	return cp
}

// new tester
func (cp TesterPool) new() *T {
	u := launcher.New().MustLaunch()

	mc := newMockClient(cdp.New(u).Logger(cp.logger))

	browser := rod.New().ControlURL("").Client(mc).MustConnect().
		MustIgnoreCertErrors(false).
		DefaultViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: 800, Height: 600, DeviceScaleFactor: 1,
		})

	page := getOnePage(browser)

	return &T{
		mc:      mc,
		browser: browser,
		page:    page,
	}
}

// get a tester
func (cp TesterPool) get(t *testing.T) T {
	if testing.Short() {
		t.Parallel()
	}

	tester := <-cp.list
	if tester == nil {
		tester = cp.new()
	}
	t.Cleanup(func() { cp.list <- tester })

	if !testing.Short() {
		testleak.Check(t, 0)
	}

	t.Cleanup(func() {
		for _, p := range tester.browser.MustPages() {
			if p.TargetID != tester.page.TargetID {
				t.Fatalf("%s is leaking page: %s", t.Name(), p.MustInfo().URL)
			}
		}

		tester.mc.setCall(nil)
	})

	tester.mc.t = t
	tester.G = got.New(t)

	heartBeat(tester)

	return *tester
}

// when concurrently run tests, indicate the busy ones
func heartBeat(t *T) {
	if !testing.Short() {
		return
	}

	ctx := t.Context()
	go func() {
		t.Helper()
		tmr := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tmr.C:
			}
			t.Log(t.Name(), "busy...")
		}
	}()
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
func (t T) srcFile(path string) string {
	f, err := filepath.Abs(slash(path))
	t.E(err)
	return "file://" + f
}

func (t T) newPage(u string) *rod.Page {
	p := t.browser.MustPage(u)
	t.Cleanup(p.MustClose)
	return p
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

func newMockClient(client *cdp.Client) *MockClient {
	return &MockClient{principal: client}
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
func (mc *MockClient) stubErr(nth int, p proto.Payload) {
	mc.stub(nth, p, func(send StubSend) (gson.JSON, error) {
		return gson.New(nil), errors.New("mock error")
	})
}

var slash = filepath.FromSlash
