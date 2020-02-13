package rod

import (
	"context"
	"sync"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/launcher"
)

// Browser represents the browser
type Browser struct {
	// ControlURL is the url to remote control browser.
	// If fails to connect to it, rod will try to open a local browser, if local browser is not found rod will try to download one.
	ControlURL string

	// Viewport is the default value to set after page creation
	// options: https://chromedevtools.github.io/devtools-protocol/tot/Emulation#method-setDeviceMetricsOverride
	Viewport *cdp.Object

	// Foreground enables the browser to run on foreground mode
	Foreground bool

	// Slowmotion delay each chrome control action
	Slowmotion time.Duration

	// Trace enables the visual tracing of the device input on the page
	Trace bool

	ctx           context.Context
	timeoutCancel func()
	close         func()
	client        *cdp.Client
	event         *kit.Observable
}

// OpenE ...
func (b *Browser) OpenE() (*Browser, error) {
	if b.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		b.ctx = ctx
		b.close = cancel
	}

	if _, err := launcher.GetWebSocketDebuggerURL(b.ControlURL); err != nil {
		args := launcher.Args()
		if b.Foreground {
			delete(args, "--headless")
		}
		u, err := launcher.LaunchE("", "", args)
		if err != nil {
			return nil, err
		}
		b.ControlURL = u
	}

	client, err := cdp.New(b.ctx, b.ControlURL)
	if err != nil {
		return nil, err
	}

	b.client = client

	return b, b.initEvents()
}

// Open a new browser controller
func Open(b *Browser) *Browser {
	if b == nil {
		b = &Browser{}
	}

	kit.E(b.OpenE())

	return b
}

// Ctx creates a clone with specified context
func (b *Browser) Ctx(ctx context.Context) *Browser {
	newObj := *b
	newObj.ctx = ctx
	return &newObj
}

// Timeout sets the timeout for chained sub-operations
func (b *Browser) Timeout(d time.Duration) *Browser {
	ctx, cancel := context.WithTimeout(b.ctx, d)
	b.timeoutCancel = cancel
	return b.Ctx(ctx)
}

// CloseE ...
func (b *Browser) CloseE() error {
	_, err := b.Call(&cdp.Request{Method: "Browser.close"})
	if err != nil {
		return err
	}

	if b.close != nil {
		b.close()
	}

	return nil
}

// Close the browser and release related resources
func (b *Browser) Close() {
	kit.E(b.CloseE())
}

// PageE ...
func (b *Browser) PageE(url string) (*Page, error) {
	target, err := b.Call(&cdp.Request{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": "about:blank",
		},
	})
	if err != nil {
		return nil, err
	}

	page, err := b.page(target.Get("targetId").String())
	if err != nil {
		return nil, err
	}

	err = page.NavigateE(url)
	if err != nil {
		return nil, err
	}

	return page, nil
}

// Page creates a new tab
func (b *Browser) Page(url string) *Page {
	p, err := b.PageE(url)
	kit.E(err)
	return p
}

// PagesE ...
func (b *Browser) PagesE() ([]*Page, error) {
	list, err := b.Call(&cdp.Request{Method: "Target.getTargets"})
	if err != nil {
		return nil, err
	}

	pageList := []*Page{}
	for _, target := range list.Get("targetInfos").Array() {
		if target.Get("type").String() != "page" {
			continue
		}

		page, err := b.page(target.Get("targetId").String())
		if err != nil {
			return nil, err
		}
		pageList = append(pageList, page)
	}

	return pageList, nil
}

// Pages returns all visible pages
func (b *Browser) Pages() []*Page {
	list, err := b.PagesE()
	kit.E(err)
	return list
}

// EventFilter to filter events
type EventFilter func(*cdp.Event) bool

// WaitEventE ...
func (b *Browser) WaitEventE(filter EventFilter) (func() (*cdp.Event, error), func()) {
	ctx, cancel := context.WithCancel(b.ctx)
	var event *cdp.Event
	var err error
	w := kit.All(func() {
		_, err = b.Event().Until(ctx, func(e kit.Event) bool {
			event = e.(*cdp.Event)
			return filter(event)
		})
	})

	return func() (*cdp.Event, error) {
		w()
		return event, err
	}, cancel
}

// WaitEvent resolves the wait function when the filter returns true, call cancel to release the resource
func (b *Browser) WaitEvent(name string) (wait func() *cdp.Event, cancel func()) {
	w, c := b.WaitEventE(Method(name))
	return func() *cdp.Event {
		e, err := w()
		kit.E(err)
		return e
	}, c
}

// Call sends a control message to browser
func (b *Browser) Call(req *cdp.Request) (kit.JSONResult, error) {
	b.slowmotion(req.Method)

	return b.client.Call(b.ctx, req)
}

// Event returns the observable for browser events
func (b *Browser) Event() *kit.Observable {
	return b.event
}

func (b *Browser) page(targetID string) (*Page, error) {
	page := &Page{
		ctx:                 b.ctx,
		browser:             b,
		TargetID:            targetID,
		getDownloadFileLock: &sync.Mutex{},
	}

	page.Mouse = &Mouse{page: page}

	page.Keyboard = &Keyboard{page: page}

	return page, page.initSession()
}

func (b *Browser) initEvents() error {
	b.event = kit.NewObservable()

	go func() {
		for msg := range b.client.Event() {
			go b.event.Publish(msg)
		}
	}()

	_, err := b.Call(&cdp.Request{
		Method: "Target.setDiscoverTargets",
		Params: cdp.Object{"discover": true},
	})

	return err
}
