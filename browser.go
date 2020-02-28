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
// It doesn't depends on file system, it should work with remote browser seamlessly.
type Browser struct {
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	controlURL string
	viewport   *cdp.Object
	slowmotion time.Duration
	trace      bool

	client *cdp.Client
	event  *kit.Observable
}

// New creates a controller
func New() *Browser {
	return &Browser{
		client: cdp.New(),
	}
}

// ControlURL set the url to remote control browser.
func (b *Browser) ControlURL(url string) *Browser {
	b.controlURL = url
	return b
}

// Viewport set the default viewport for newly created page
// options: https://chromedevtools.github.io/devtools-protocol/tot/Emulation#method-setDeviceMetricsOverride
func (b *Browser) Viewport(opts *cdp.Object) *Browser {
	b.viewport = opts
	return b
}

// Slowmotion set the delay for each chrome control action
func (b *Browser) Slowmotion(delay time.Duration) *Browser {
	b.slowmotion = delay
	return b
}

// Trace enables/disables the visual tracing of the input actions on the page
func (b *Browser) Trace(enable bool) *Browser {
	b.trace = enable
	return b
}

// DebugCDP enables/disables the log of all cdp interface traffic
func (b *Browser) DebugCDP(enable bool) *Browser {
	b.client.Debug(enable)
	return b
}

// ConnectE ...
func (b *Browser) ConnectE() error {
	*b = *b.Context(b.ctx)

	if b.controlURL == "" {
		u, err := launcher.New().Context(b.ctx).LaunchE()
		if err != nil {
			return err
		}
		b.controlURL = u
	}

	b.client.URL(b.controlURL).Context(b.ctx).Connect()

	return b.initEvents()
}

// CloseE ...
func (b *Browser) CloseE() error {
	_, err := b.CallE(nil, &cdp.Request{Method: "Browser.close"})
	if err != nil {
		return err
	}

	return nil
}

// PageE ...
func (b *Browser) PageE(url string) (*Page, error) {
	target, err := b.CallE(nil, &cdp.Request{
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

// PagesE ...
func (b *Browser) PagesE() ([]*Page, error) {
	list, err := b.CallE(nil, &cdp.Request{Method: "Target.getTargets"})
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

// EventFilter to filter events
type EventFilter func(*cdp.Event) bool

// WaitEventE returns wait and cancel methods
func (b *Browser) WaitEventE(ctx context.Context, filter EventFilter) func() (*cdp.Event, error) {
	if ctx == nil {
		ctx = b.ctx
	}

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
	}
}

// CallE sends a control message to browser
func (b *Browser) CallE(ctx context.Context, req *cdp.Request) (kit.JSONResult, error) {
	b.trySlowmotion(req.Method)

	if ctx == nil {
		ctx = b.ctx
	}

	return b.client.Call(ctx, req)
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
		b.event.UnsubscribeAll()
	}()

	_, err := b.CallE(nil, &cdp.Request{
		Method: "Target.setDiscoverTargets",
		Params: cdp.Object{"discover": true},
	})

	return err
}
