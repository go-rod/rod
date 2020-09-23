//go:generate go run ./lib/utils/setup
//go:generate go run ./lib/utils/lint
//go:generate go run ./lib/proto/generate
//go:generate go run ./lib/assets/generate
//go:generate go run ./lib/devices/generate
//go:generate go run ./lib/launcher/revision

package rod

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/goob"
)

// Browser implements the proto.Caller interface
var _ proto.Caller = &Browser{}

// Browser represents the browser.
// It doesn't depends on file system, it should work with remote browser seamlessly.
// To check the env var you can use to quickly enable options from CLI, check here:
// https://pkg.go.dev/github.com/go-rod/rod/lib/defaults
type Browser struct {
	// these are the handler for ctx
	ctx     context.Context
	sleeper func() utils.Sleeper

	// BrowserContextID is the id for incognito window
	BrowserContextID proto.BrowserBrowserContextID

	slowmotion time.Duration // see defaults.slow
	trace      bool          // see defaults.Trace
	traceLog   TraceLog
	headless   bool

	defaultViewport *proto.EmulationSetDeviceMetricsOverride

	client      Client
	event       *goob.Observable // all the browser events from cdp client
	targetsLock *sync.Mutex

	// stores all the previous cdp call of same type. Browser doesn't have enough API
	// for us to retrieve all its internal states. This is an workaround to map them to local.
	// For example you can't use cdp API to get the current position of mouse.
	states *sync.Map
}

// New creates a controller
func New() *Browser {
	return &Browser{
		ctx:             context.Background(),
		sleeper:         DefaultSleeper,
		slowmotion:      defaults.Slow,
		trace:           defaults.Trace,
		traceLog:        defaultTraceLog,
		defaultViewport: devices.LaptopWithMDPIScreen.Metrics(true),
		targetsLock:     &sync.Mutex{},
		states:          &sync.Map{},
	}
}

// Incognito creates a new incognito browser
func (b *Browser) Incognito() (*Browser, error) {
	res, err := proto.TargetCreateBrowserContext{}.Call(b)
	if err != nil {
		return nil, err
	}

	incognito := *b
	incognito.BrowserContextID = res.BrowserContextID

	return &incognito, nil
}

// ControlURL set the url to remote control browser.
func (b *Browser) ControlURL(url string) *Browser {
	b.client = cdp.New(url)
	return b
}

// Slowmotion set the delay for each control action, such as the simulation of the human inputs
func (b *Browser) Slowmotion(delay time.Duration) *Browser {
	b.slowmotion = delay
	return b
}

// Trace enables/disables the visual tracing of the input actions on the page
func (b *Browser) Trace(enable bool) *Browser {
	b.trace = enable
	return b
}

// TraceLog overrides the default log functions for tracing
func (b *Browser) TraceLog(l TraceLog) *Browser {
	if l == nil {
		b.traceLog = defaultTraceLog
	} else {
		b.traceLog = l
	}
	return b
}

// Client set the cdp client
func (b *Browser) Client(c Client) *Browser {
	b.client = c
	return b
}

// DefaultViewport sets the default viewport for new page in the future. Default size is 1200x900.
// Set it to nil to disable it.
func (b *Browser) DefaultViewport(viewport *proto.EmulationSetDeviceMetricsOverride) *Browser {
	b.defaultViewport = viewport
	return b
}

// Connect to the browser and start to control it.
// If fails to connect, try to launch a local browser, if local browser not found try to download one.
func (b *Browser) Connect() error {
	if b.client == nil {
		u := defaults.URL
		if defaults.Remote {
			if u == "" {
				u = "ws://127.0.0.1:9222"
			}
			b.client = launcher.NewRemote(u).Client()
		} else {
			if u == "" {
				u = launcher.New().Context(b.ctx).MustLaunch()
			}
			b.client = cdp.New(u)
		}
	}

	err := b.client.Connect(b.ctx)
	if err != nil {
		return err
	}

	b.initEvents()

	if defaults.Monitor != "" {
		launcher.NewBrowser().Open(b.ServeMonitor(defaults.Monitor))
	}

	return b.setHeadless()
}

// Close the browser
func (b *Browser) Close() error {
	return proto.BrowserClose{}.Call(b)
}

// Page creates a new browser tab. If url is empty, the default target will be "about:blank".
func (b *Browser) Page(url string) (p *Page, err error) {
	target, err := proto.TargetCreateTarget{
		URL:              "about:blank",
		BrowserContextID: b.BrowserContextID,
	}.Call(b)
	if err != nil {
		return nil, err
	}
	defer func() {
		// If Navigate or PageFromTarget fails we should close the target to prevent leak
		if err != nil {
			_, _ = proto.TargetCloseTarget{TargetID: target.TargetID}.Call(b)
		}
	}()

	p, err = b.PageFromTarget(target.TargetID)
	if err == nil && url != "" { // no need to navigate if url is empty
		err = p.Navigate(url)
	}

	return
}

// Pages retrieves all visible pages
func (b *Browser) Pages() (Pages, error) {
	list, err := proto.TargetGetTargets{}.Call(b)
	if err != nil {
		return nil, err
	}

	pageList := Pages{}
	for _, target := range list.TargetInfos {
		if target.Type != "page" {
			continue
		}

		page, err := b.PageFromTarget(target.TargetID)
		if err != nil {
			return nil, err
		}
		pageList = append(pageList, page)
	}

	return pageList, nil
}

// Event returns the observable for browser events
func (b *Browser) Event() *goob.Observable {
	return b.event
}

// EachEvent of the specified event type, if any callback returns true the event loop will stop.
func (b *Browser) EachEvent(callbacks ...interface{}) (wait func()) {
	return b.eachEvent(b.ctx, "", callbacks...)
}

// WaitEvent waits for the next event for one time. It will also load the data into the event object.
func (b *Browser) WaitEvent(e proto.Payload) (wait func()) {
	return b.waitEvent(b.ctx, "", e)
}

// If the any callback returns true the event loop will stop.
// It will enable the related domains if not enabled, and recover them after wait ends.
func (b *Browser) eachEvent(
	ctx context.Context,
	sessionID proto.TargetSessionID,
	callbacks ...interface{},
) (wait func()) {
	cbValues := make([]reflect.Value, len(callbacks))
	eventTypes := make([]reflect.Type, len(callbacks))
	recovers := make([]func(), len(callbacks))

	for i, cb := range callbacks {
		cbValues[i] = reflect.ValueOf(cb)
		eType := cbValues[i].Type().In(0).Elem()
		eventTypes[i] = eType

		// Only enabled domains will emit events to cdp client.
		// We enable the domains for the event types if it's not enabled.
		// We recover the domains to their previous states after the wait ends.
		domain, _ := proto.ParseMethodName(reflect.New(eType).Interface().(proto.Payload).MethodName())
		var enable proto.Payload
		if domain == "Target" { // only Target domain is special
			enable = proto.TargetSetDiscoverTargets{Discover: true}
		} else {
			enable = reflect.New(proto.GetType(domain + ".enable")).Interface().(proto.Payload)
		}
		recovers[i] = b.Context(ctx).EnableDomain(sessionID, enable)
	}

	ctx, cancel := context.WithCancel(ctx)
	stream := b.event.Subscribe(ctx)

	return func() {
		defer func() {
			cancel()
			stream = nil
			for _, recover := range recovers {
				recover()
			}
		}()

		if stream == nil {
			panic("can't use wait function twice")
		}

		// Check each event, if an event matches the the type of the arg call the fn with the even.
		goob.Each(stream, func(e *cdp.Event) bool {
			for i, eType := range eventTypes {
				eVal := reflect.New(eType)
				if Event(e, eVal.Interface().(proto.Payload)) {
					// The type of callback can be one of:
					//   func(e proto.Payload) bool
					//   func(e proto.Payload)
					res := cbValues[i].Call([]reflect.Value{eVal})
					if len(res) > 0 {
						return res[0].Bool()
					}
					break
				}
			}
			return false
		})
	}
}

// waits for the next event for one time. It will also load the data into the event object.
func (b *Browser) waitEvent(ctx context.Context, sessionID proto.TargetSessionID, e proto.Payload) (wait func()) {
	valE := reflect.ValueOf(e)
	valTrue := reflect.ValueOf(true)

	// dynamically creates a function on runtime:
	//
	// func(ee proto.Payload) bool {
	//   *e = *ee
	//   return true
	// }
	fnType := reflect.FuncOf([]reflect.Type{valE.Type()}, []reflect.Type{valTrue.Type()}, false)
	fnVal := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		valE.Elem().Set(args[0].Elem())
		return []reflect.Value{valTrue}
	})

	return b.eachEvent(ctx, sessionID, fnVal.Interface())
}

// Call raw cdp interface directly
func (b *Browser) Call(ctx context.Context, sessionID, methodName string, params json.RawMessage) (res []byte, err error) {
	res, err = b.client.Call(ctx, sessionID, methodName, params)
	if err != nil {
		return nil, err
	}

	b.set(proto.TargetSessionID(sessionID), methodName, params)

	return
}

// CallContext parameters for proto
func (b *Browser) CallContext() (context.Context, proto.Client, string) {
	return b.ctx, b, ""
}

// PageFromTarget creates a Page instance from a targetID
func (b *Browser) PageFromTarget(targetID proto.TargetTargetID) (*Page, error) {
	b.targetsLock.Lock()
	defer b.targetsLock.Unlock()

	page := b.loadPage(targetID)
	if page != nil {
		return page, nil
	}

	page = (&Page{
		sleeper:       b.sleeper,
		jsContextLock: &sync.Mutex{},
		browser:       b,
		TargetID:      targetID,
		executionIDs:  map[proto.PageFrameID]proto.RuntimeExecutionContextID{},
	}).Context(b.ctx)

	page.Mouse = &Mouse{page: page, id: utils.RandString(8)}
	page.Keyboard = &Keyboard{page: page}
	page.Touch = &Touch{page: page}

	err := page.initSession()
	if err != nil {
		return nil, err
	}

	if b.defaultViewport != nil {
		err = page.SetViewport(b.defaultViewport)
		if err != nil {
			return nil, err
		}
	}

	b.storePage(page)

	return page, nil
}

func (b *Browser) initEvents() {
	b.event = goob.New()

	go func() {
		for msg := range b.client.Event() {
			b.event.Publish(msg)
		}
	}()
}

func (b *Browser) pageInfo(id proto.TargetTargetID) (*proto.TargetTargetInfo, error) {
	res, err := proto.TargetGetTargetInfo{TargetID: id}.Call(b)
	if err != nil {
		return nil, err
	}
	return res.TargetInfo, nil
}

// IgnoreCertErrors switch. If enabled, all certificate errors will be ignored.
func (b *Browser) IgnoreCertErrors(enable bool) error {
	return proto.SecuritySetIgnoreCertificateErrors{Ignore: enable}.Call(b)
}

// Headless mode or not
func (b *Browser) Headless() bool {
	return b.headless
}

func (b *Browser) setHeadless() error {
	res, err := proto.BrowserGetBrowserCommandLine{}.Call(b)
	if err != nil {
		return err
	}

	for _, arg := range res.Arguments {
		if arg == "--headless" {
			b.headless = true
		}
	}
	return nil
}
