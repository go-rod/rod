//go:generate go run ./lib/utils/setup
//go:generate go run ./lib/launcher/revision
//go:generate go run ./lib/proto/generate
//go:generate go run ./lib/js/generate
//go:generate go run ./lib/assets/generate
//go:generate go run ./lib/devices/generate
//go:generate go run ./lib/utils/lint

package rod

import (
	"context"
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

// Browser implements these interfaces
var _ proto.Client = &Browser{}
var _ proto.Contextable = &Browser{}

// Browser represents the browser.
// It doesn't depends on file system, it should work with remote browser seamlessly.
// To check the env var you can use to quickly enable options from CLI, check here:
// https://pkg.go.dev/github.com/go-rod/rod/lib/defaults
type Browser struct {
	// BrowserContextID is the id for incognito window
	BrowserContextID proto.BrowserBrowserContextID

	ctx context.Context

	sleeper func() utils.Sleeper

	logger utils.Logger

	slowMotion time.Duration // see defaults.slow
	trace      bool          // see defaults.Trace
	monitor    string

	defaultDevice devices.Device

	client      CDPClient
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
		ctx:           context.Background(),
		sleeper:       DefaultSleeper,
		slowMotion:    defaults.Slow,
		trace:         defaults.Trace,
		monitor:       defaults.Monitor,
		logger:        DefaultLogger,
		defaultDevice: devices.LaptopWithMDPIScreen.Landescape(),
		targetsLock:   &sync.Mutex{},
		states:        &sync.Map{},
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
	if url == "" {
		b.client = nil
	} else {
		b.client = cdp.New(url)
	}
	return b
}

// SlowMotion set the delay for each control action, such as the simulation of the human inputs
func (b *Browser) SlowMotion(delay time.Duration) *Browser {
	b.slowMotion = delay
	return b
}

// Trace enables/disables the visual tracing of the input actions on the page
func (b *Browser) Trace(enable bool) *Browser {
	b.trace = enable
	return b
}

// Monitor address to listen if not empty. Shortcut for Browser.ServeMonitor
func (b *Browser) Monitor(url string) *Browser {
	b.monitor = url
	return b
}

// Logger overrides the default log functions for tracing
func (b *Browser) Logger(l utils.Logger) *Browser {
	b.logger = l
	return b
}

// Client set the cdp client
func (b *Browser) Client(c CDPClient) *Browser {
	b.client = c
	return b
}

// DefaultDevice sets the default device for new page in the future. Default is devices.LaptopWithMDPIScreen .
// Set it to devices.Clear to disable it.
func (b *Browser) DefaultDevice(d devices.Device) *Browser {
	b.defaultDevice = d
	return b
}

// NoDefaultDevice is the same as DefaultDevice(devices.Clear)
func (b *Browser) NoDefaultDevice() *Browser {
	return b.DefaultDevice(devices.Clear)
}

// Connect to the browser and start to control it.
// If fails to connect, try to launch a local browser, if local browser not found try to download one.
func (b *Browser) Connect() error {
	if b.client == nil {
		u := defaults.URL
		if u == "" {
			u = launcher.New().Context(b.ctx).MustLaunch()
		}
		b.client = cdp.New(u)
	}

	err := b.client.Connect(b.ctx)
	if err != nil {
		return err
	}

	b.initEvents()

	err = proto.TargetSetDiscoverTargets{Discover: true}.Call(b)
	if err != nil {
		return err
	}

	if b.monitor != "" {
		launcher.Open(b.ServeMonitor(b.monitor))
	}

	return nil
}

// Close the browser
func (b *Browser) Close() error {
	if b.BrowserContextID == "" {
		return proto.BrowserClose{}.Call(b)
	}
	return proto.TargetDisposeBrowserContext{BrowserContextID: b.BrowserContextID}.Call(b)
}

// Page creates a new browser tab. If url is empty, the default target will be "about:blank".
func (b *Browser) Page(opts proto.TargetCreateTarget) (p *Page, err error) {
	req := opts
	req.BrowserContextID = b.BrowserContextID
	req.URL = "about:blank"

	target, err := req.Call(b)
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
	if err != nil {
		return
	}

	if opts.URL == "" {
		return
	}

	err = p.Navigate(opts.URL)

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
		if target.Type != proto.TargetTargetInfoTypePage {
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

// Call raw cdp interface directly
func (b *Browser) Call(ctx context.Context, sessionID, methodName string, params interface{}) (res []byte, err error) {
	res, err = b.client.Call(ctx, sessionID, methodName, params)
	if err != nil {
		return nil, err
	}

	b.set(proto.TargetSessionID(sessionID), methodName, params)
	return
}

// PageFromSession is used for low-level debugging
func (b *Browser) PageFromSession(sessionID proto.TargetSessionID) *Page {
	return &Page{
		ctx:       b.ctx,
		sleeper:   b.sleeper,
		browser:   b,
		SessionID: sessionID,
	}
}

// PageFromTarget gets or creates a Page instance.
func (b *Browser) PageFromTarget(targetID proto.TargetTargetID) (*Page, error) {
	b.targetsLock.Lock()
	defer b.targetsLock.Unlock()

	page := b.loadCachedPage(targetID)
	if page != nil {
		return page, nil
	}

	session, err := proto.TargetAttachToTarget{
		TargetID: targetID,
		Flatten:  true, // if it's not set no response will return
	}.Call(b)
	if err != nil {
		return nil, err
	}

	page = &Page{
		ctx:       b.ctx,
		sleeper:   b.sleeper,
		browser:   b,
		TargetID:  targetID,
		SessionID: session.SessionID,
		FrameID:   proto.PageFrameID(targetID),
		jsCtxLock: &sync.Mutex{},
		jsCtxID:   new(proto.RuntimeRemoteObjectID),
	}

	page.root = page
	page.Mouse = &Mouse{page: page, id: utils.RandString(8)}
	page.Keyboard = &Keyboard{page: page}
	page.Touch = &Touch{page: page}

	if !b.defaultDevice.IsClear() {
		err = page.Emulate(b.defaultDevice)
		if err != nil {
			return nil, err
		}
	}

	b.cachePage(page)

	// If we don't enable it, it will cause a lot of unexpected browser behavior.
	// Such as proto.PageAddScriptToEvaluateOnNewDocument won't work.
	page.EnableDomain(&proto.PageEnable{})

	return page, nil
}

// EachEvent is similar to Page.EachEvent, but catches events of the entire browser.
func (b *Browser) EachEvent(callbacks ...interface{}) (wait func()) {
	return b.eachEvent("", callbacks...)
}

// WaitEvent waits for the next event for one time. It will also load the data into the event object.
func (b *Browser) WaitEvent(e proto.Event) (wait func()) {
	return b.waitEvent("", e)
}

// waits for the next event for one time. It will also load the data into the event object.
func (b *Browser) waitEvent(sessionID proto.TargetSessionID, e proto.Event) (wait func()) {
	valE := reflect.ValueOf(e)
	valTrue := reflect.ValueOf(true)

	if valE.Kind() != reflect.Ptr {
		valE = reflect.New(valE.Type())
	}

	// dynamically creates a function on runtime:
	//
	// func(ee proto.Event) bool {
	//   *e = *ee
	//   return true
	// }
	fnType := reflect.FuncOf([]reflect.Type{valE.Type()}, []reflect.Type{valTrue.Type()}, false)
	fnVal := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		valE.Elem().Set(args[0].Elem())
		return []reflect.Value{valTrue}
	})

	return b.eachEvent(sessionID, fnVal.Interface())
}

// If the any callback returns true the event loop will stop.
// It will enable the related domains if not enabled, and restore them after wait ends.
func (b *Browser) eachEvent(sessionID proto.TargetSessionID, callbacks ...interface{}) (wait func()) {
	cbMap := map[string]reflect.Value{}
	restores := []func(){}

	for _, cb := range callbacks {
		cbVal := reflect.ValueOf(cb)
		eType := cbVal.Type().In(0)
		name := reflect.New(eType.Elem()).Interface().(proto.Event).ProtoEvent()
		cbMap[name] = cbVal

		// Only enabled domains will emit events to cdp client.
		// We enable the domains for the event types if it's not enabled.
		// We restore the domains to their previous states after the wait ends.
		domain, _ := proto.ParseMethodName(name)
		if req := proto.GetType(domain + ".enable"); req != nil {
			enable := reflect.New(req).Interface().(proto.Request)
			restores = append(restores, b.EnableDomain(sessionID, enable))
		}
	}

	b, cancel := b.WithCancel()
	messages := b.Event()

	return func() {
		if messages == nil {
			panic("can't use wait function twice")
		}

		defer func() {
			cancel()
			messages = nil
			for _, restore := range restores {
				restore()
			}
		}()

		for msg := range messages {
			if !(sessionID == "" || msg.SessionID == sessionID) {
				continue
			}

			if cbVal, has := cbMap[msg.Method]; has {
				e := reflect.New(proto.GetType(msg.Method))
				msg.Load(e.Interface().(proto.Event))
				args := []reflect.Value{e}
				if cbVal.Type().NumIn() == 2 {
					args = append(args, reflect.ValueOf(sessionID))
				}
				res := cbVal.Call(args)
				if len(res) > 0 {
					if res[0].Bool() {
						return
					}
				}
			}
		}
	}
}

// Event of the browser
func (b *Browser) Event() <-chan *Message {
	src := b.event.Subscribe()
	dst := make(chan *Message)
	go func() {
		defer b.event.Unsubscribe(src)
		defer close(dst)
		for {
			select {
			case <-b.ctx.Done():
				return
			case e, ok := <-src:
				if !ok {
					return
				}
				select {
				case <-b.ctx.Done():
					return
				case dst <- e.(*Message):
				}
			}
		}
	}()
	return dst
}

func (b *Browser) initEvents() {
	b.event = goob.New()

	go func() {
		defer b.event.Close()
		for e := range b.client.Event() {
			b.event.Publish(&Message{
				SessionID: proto.TargetSessionID(e.SessionID),
				Method:    e.Method,
				lock:      &sync.Mutex{},
				data:      e.Params,
			})
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

// GetCookies from the browser
func (b *Browser) GetCookies() ([]*proto.NetworkCookie, error) {
	res, err := proto.StorageGetCookies{BrowserContextID: b.BrowserContextID}.Call(b)
	if err != nil {
		return nil, err
	}
	return res.Cookies, nil
}

// SetCookies to the browser
func (b *Browser) SetCookies(cookies []*proto.NetworkCookieParam) error {
	return proto.StorageSetCookies{
		Cookies:          cookies,
		BrowserContextID: b.BrowserContextID,
	}.Call(b)
}

// WaitDownload returns a helper to get the next download file.
// The file path will be:
//     filepath.Join(dir, info.GUID)
func (b *Browser) WaitDownload(dir string) func() (info *proto.PageDownloadWillBegin) {
	var oldDownloadBehavior proto.BrowserSetDownloadBehavior
	has := b.LoadState("", &oldDownloadBehavior)

	_ = proto.BrowserSetDownloadBehavior{
		Behavior:         proto.BrowserSetDownloadBehaviorBehaviorAllowAndName,
		BrowserContextID: b.BrowserContextID,
		DownloadPath:     dir,
	}.Call(b)

	var start *proto.PageDownloadWillBegin

	waitProgress := b.EachEvent(func(e *proto.PageDownloadWillBegin) {
		start = e
	}, func(e *proto.PageDownloadProgress) bool {
		return start.GUID == e.GUID && e.State == proto.PageDownloadProgressStateCompleted
	})

	return func() *proto.PageDownloadWillBegin {
		defer func() {
			if has {
				_ = oldDownloadBehavior.Call(b)
			} else {
				_ = proto.BrowserSetDownloadBehavior{
					Behavior:         proto.BrowserSetDownloadBehaviorBehaviorDefault,
					BrowserContextID: b.BrowserContextID,
				}.Call(b)
			}
		}()

		waitProgress()

		return start
	}
}
