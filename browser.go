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
	// these are the handler for ctx
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	// BrowserContextID is the id for incognito window
	BrowserContextID string

	controlURL string
	viewport   cdp.Object    // default viewport, such as window demension and dpi
	slowmotion time.Duration // slowdown user inputs
	trace      bool          // enable show auto tracing of user inputs

	client *cdp.Client
	event  *kit.Observable // all the browser events from cdp client
}

// New creates a controller
func New() *Browser {
	return &Browser{
		ctx:    context.Background(),
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
func (b *Browser) Viewport(opts cdp.Object) *Browser {
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

// ConnectE doc is the same as the method Connect
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

// CloseE doc is the same as the method Close
func (b *Browser) CloseE() error {
	_, err := b.CallE(&cdp.Request{Method: "Browser.close"})
	if err != nil {
		return err
	}

	return nil
}

// IncognitoE creates a new incognito browser
func (b *Browser) IncognitoE() (*Browser, error) {
	res, err := b.CallE(&cdp.Request{
		Method: "Target.createBrowserContext",
	})
	if err != nil {
		return nil, err
	}

	id := res.Get("browserContextId").String()

	incognito := *b
	incognito.BrowserContextID = id

	return &incognito, nil
}

// PageE doc is the same as the method Page
func (b *Browser) PageE(url string) (*Page, error) {
	if url == "" {
		url = "about:blank"
	}

	params := cdp.Object{
		"url": url,
	}

	if b.BrowserContextID != "" {
		params["browserContextId"] = b.BrowserContextID
	}

	target, err := b.CallE(&cdp.Request{
		Method: "Target.createTarget",
		Params: params,
	})
	if err != nil {
		return nil, err
	}

	return b.page(target.Get("targetId").String())
}

// PagesE doc is the same as the method Pages
func (b *Browser) PagesE() (Pages, error) {
	list, err := b.CallE(&cdp.Request{Method: "Target.getTargets"})
	if err != nil {
		return nil, err
	}

	pageList := Pages{}
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
func (b *Browser) WaitEventE(filter EventFilter) func() (*cdp.Event, error) {
	var event *cdp.Event
	var err error
	w := kit.All(func() {
		_, err = b.Event().Until(b.ctx, func(e kit.Event) bool {
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
func (b *Browser) CallE(req *cdp.Request) (kit.JSONResult, error) {
	b.trySlowmotion(req.Method)
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
		b.event.UnsubscribeAll()
	}()

	_, err := b.CallE(&cdp.Request{
		Method: "Target.setDiscoverTargets",
		Params: cdp.Object{"discover": true},
	})

	return err
}
