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
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/goob"
	"github.com/ysmood/kit"
)

// Browser implements the proto.Caller interface
var _ proto.Caller = &Browser{}

// Browser represents the browser
// It doesn't depends on file system, it should work with remote browser seamlessly.
// To check the env var you can use to quickly enable options from CLI, check here:
// https://pkg.go.dev/github.com/go-rod/rod/lib/defaults
type Browser struct {
	// these are the handler for ctx
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	// BrowserContextID is the id for incognito window
	BrowserContextID proto.BrowserBrowserContextID

	slowmotion time.Duration // see defaults.slow
	trace      bool          // see defaults.Trace
	quiet      bool          // see defaults.Quiet

	monitorServer *kit.ServerContext

	client *cdp.Client
	event  *goob.Observable // all the browser events from cdp client

	// stores all the previous cdp call of same type. Browser doesn't have enough API
	// for us to retrieve all its internal states. This is an workaround to map them to local.
	// For example you can't use cdp API to get the current position of mouse.
	states *sync.Map
}

// New creates a controller
func New() *Browser {
	b := &Browser{
		slowmotion: defaults.Slow,
		trace:      defaults.Trace,
		quiet:      defaults.Quiet,
		states:     &sync.Map{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	return b.Context(ctx, cancel)
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

// Quiet enables/disables log of the. Only useful when Trace is set to true.
func (b *Browser) Quiet(quiet bool) *Browser {
	b.quiet = quiet
	return b
}

// Client set the cdp client
func (b *Browser) Client(c *cdp.Client) *Browser {
	b.client = c
	return b
}

// ConnectE doc is similar to the method Connect
func (b *Browser) ConnectE() error {
	if b.client == nil {
		u := defaults.URL
		if defaults.Remote {
			if u == "" {
				u = "ws://127.0.0.1:9222"
			}
			b.client = launcher.NewRemote(u).Client()
		} else {
			if u == "" {
				var err error
				u, err = launcher.New().Context(b.ctx).LaunchE()
				if err != nil {
					return err
				}
			}
			b.client = cdp.New(u)
		}
	}

	err := b.client.Context(b.ctx, b.ctxCancel).ConnectE()
	if err != nil {
		return err
	}

	b.monitorServer = b.ServeMonitor(defaults.Monitor, !defaults.Blind)

	b.initEvents()

	return nil
}

// CloseE doc is similar to the method Close
func (b *Browser) CloseE() error {
	defer b.ctxCancel()
	return proto.BrowserClose{}.Call(b)
}

// IncognitoE creates a new incognito browser
func (b *Browser) IncognitoE() (*Browser, error) {
	res, err := proto.TargetCreateBrowserContext{}.Call(b)
	if err != nil {
		return nil, err
	}

	incognito := *b
	incognito.BrowserContextID = res.BrowserContextID

	return &incognito, nil
}

// PageE doc is similar to the method Page
func (b *Browser) PageE(url string) (*Page, error) {
	if url == "" {
		url = "about:blank"
	}

	req := proto.TargetCreateTarget{
		URL: url,
	}

	req.BrowserContextID = b.BrowserContextID

	target, err := req.Call(b)
	if err != nil {
		return nil, err
	}

	return b.PageFromTargetIDE(target.TargetID)
}

// PagesE doc is similar to the method Pages
func (b *Browser) PagesE() (Pages, error) {
	list, err := proto.TargetGetTargets{}.Call(b)
	if err != nil {
		return nil, err
	}

	pageList := Pages{}
	for _, target := range list.TargetInfos {
		if target.Type != "page" {
			continue
		}

		page, err := b.PageFromTargetIDE(target.TargetID)
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

// EachEvent of the specified event type, if the fn returns true the event loop will stop.
// The fn can accpet multiple events, such as EachEvent(func(e1 *proto.PageLoadEventFired, e2 *proto.PageLifecycleEvent) {}),
// only one argument will be non-null, others will null.
func (b *Browser) EachEvent(fn interface{}) (wait func()) {
	return b.eachEvent(b.ctx, "", fn)
}

// WaitEvent waits for the next event for one time. It will also load the data into the event object.
func (b *Browser) WaitEvent(e proto.Payload) (wait func()) {
	return b.waitEvent(b.ctx, "", e)
}

// If the fn returns true the event loop will stop.
// The fn can accpet multiple events, such as EachEventE("", func(e1 *proto.PageLoadEventFired, e2 *proto.PageLifecycleEvent) {}),
// only one argument will be non-null, others will null.
// It will enable the related domains if not enabled, and recover them after wait ends.
func (b *Browser) eachEvent(ctx context.Context, sessionID proto.TargetSessionID, fn interface{}) (wait func()) {
	type argInfo struct {
		argType reflect.Type
		recover func()
	}

	ctx, cancel := context.WithCancel(ctx)

	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)
	argInfos := []argInfo{}
	for i := 0; i < fnType.NumIn(); i++ {
		info := argInfo{
			argType: fnType.In(i),
		}

		// handle enable and recover domain
		arg := reflect.New(info.argType.Elem()).Interface().(proto.Payload)
		domain, _ := proto.ParseMethodName(arg.MethodName())
		var enable proto.Payload
		if domain == "Target" { // only Target domain is special
			enable = proto.TargetSetDiscoverTargets{Discover: true}
		} else {
			enable = reflect.New(proto.GetType(domain + ".enable")).Interface().(proto.Payload)
		}
		info.recover = b.EnableDomain(ctx, sessionID, enable)

		argInfos = append(argInfos, info)
	}

	s := b.event.Subscribe(ctx)

	return func() {
		defer func() {
			for _, state := range argInfos {
				if state.recover != nil {
					state.recover()
				}
			}

			cancel()
			s = nil
		}()

		if s == nil {
			panic("can't use wait function twice")
		}

		goob.Each(s, func(e *cdp.Event) bool {
			args := []reflect.Value{}
			has := false
			for _, info := range argInfos {
				event := reflect.New(info.argType.Elem())
				if Event(e, event.Interface().(proto.Payload)) {
					has = true
				} else {
					event = reflect.Zero(info.argType)
				}
				args = append(args, event)
			}
			if has {
				ret := fnVal.Call(args)
				if len(ret) > 0 {
					return ret[0].Bool()
				}
			}
			return false
		})
	}
}

// waits for the next event for one time. It will also load the data into the event object.
func (b *Browser) waitEvent(ctx context.Context, sessionID proto.TargetSessionID, e proto.Payload) (wait func()) {
	val := reflect.ValueOf(e)
	fnType := reflect.FuncOf([]reflect.Type{val.Type()}, []reflect.Type{reflect.TypeOf(true)}, false)
	fnVal := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		val.Elem().Set(args[0].Elem())
		return []reflect.Value{reflect.ValueOf(true)}
	})
	return b.eachEvent(ctx, sessionID, fnVal.Interface())
}

// Call raw cdp interface directly
func (b *Browser) Call(ctx context.Context, sessionID, methodName string, params json.RawMessage) (res []byte, err error) {
	b.set(proto.TargetSessionID(sessionID), methodName, params)

	return b.client.Call(ctx, sessionID, methodName, params)
}

// CallContext parameters for proto
func (b *Browser) CallContext() (context.Context, proto.Client, string) {
	return b.ctx, b, ""
}

// PageFromTargetIDE creates a Page instance from a targetID
func (b *Browser) PageFromTargetIDE(targetID proto.TargetTargetID) (*Page, error) {
	page := (&Page{
		browser:  b,
		TargetID: targetID,
	}).Context(context.WithCancel(b.ctx))

	page.Mouse = &Mouse{page: page, id: kit.RandString(8)}
	page.Keyboard = &Keyboard{page: page}

	return page, page.initSession()
}

func (b *Browser) initEvents() {
	b.event = goob.New()

	go func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			case msg := <-b.client.Event():
				b.event.Publish(msg)
			}
		}
	}()
}

// InfoE of the page
func (b *Browser) pageInfo(id proto.TargetTargetID) (*proto.TargetTargetInfo, error) {
	res, err := proto.TargetGetTargetInfo{TargetID: id}.Call(b)
	if err != nil {
		return nil, err
	}
	return res.TargetInfo, nil
}
