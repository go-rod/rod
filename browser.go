//go:generate go run ./lib/proto/generate
//go:generate go run ./lib/assets/generate

package rod

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ysmood/goob"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/defaults"
	"github.com/ysmood/rod/lib/launcher"
	"github.com/ysmood/rod/lib/proto"
)

// Browser implements the proto.Caller interface
var _ proto.Caller = &Browser{}

// Browser represents the browser
// It doesn't depends on file system, it should work with remote browser seamlessly.
// To check the env var you can use to quickly enable options from CLI, check here:
// https://pkg.go.dev/github.com/ysmood/rod/lib/defaults
type Browser struct {
	// these are the handler for ctx
	ctx           context.Context
	ctxCancel     func()
	timeoutCancel func()

	// BrowserContextID is the id for incognito window
	BrowserContextID proto.BrowserBrowserContextID

	slowmotion time.Duration // slowdown user inputs
	trace      bool          // enable show auto tracing of user inputs

	monitorServer *kit.ServerContext

	client *cdp.Client
	event  *goob.Observable // all the browser events from cdp client
}

// New creates a controller
func New() *Browser {
	b := &Browser{
		trace:      defaults.Trace,
		slowmotion: defaults.Slow,
	}

	return b.Context(context.Background())
}

// ControlURL set the url to remote control browser.
func (b *Browser) ControlURL(url string) *Browser {
	b.client = cdp.New(url)
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

	err := b.client.Context(b.ctx).ConnectE()
	if err != nil {
		return err
	}

	b.monitorServer = b.ServeMonitor(defaults.Monitor)

	return b.initEvents()
}

// CloseE doc is similar to the method Close
func (b *Browser) CloseE() error {
	err := proto.BrowserClose{}.Call(b)
	if err != nil && !websocket.IsCloseError(err, 1006) {
		return err
	}

	b.ctxCancel()

	return nil
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

	if b.BrowserContextID != "" {
		req.BrowserContextID = b.BrowserContextID
	}

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

// EventFilter to filter events
type EventFilter func(*cdp.Event) bool

// WaitEventE returns a channel that resolves the next event and close
func (b *Browser) WaitEventE(filter EventFilter) <-chan kit.Nil {
	wait := make(chan kit.Nil)
	go func() {
		goob.Each(b.event.Subscribe(b.ctx), func(e *cdp.Event) bool {
			return filter(e)
		})
		close(wait)
	}()
	return wait
}

// Event returns the observable for browser events
func (b *Browser) Event() *goob.Observable {
	return b.event
}

// HandleAuthE for the next basic HTTP authentication.
// It will prevent the popup that requires user to input user name and password.
// Ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
func (b *Browser) HandleAuthE(username, password string) (func() error, error) {
	err := proto.FetchEnable{
		HandleAuthRequests: true,
	}.Call(b)
	if err != nil {
		return nil, err
	}

	auth := &proto.FetchAuthRequired{}
	paused := &proto.FetchRequestPaused{}
	waitPaused := b.WaitEventE(NewEventFilter(paused))
	waitAuth := b.WaitEventE(NewEventFilter(auth))

	return func() (err error) {
		defer func() {
			e := proto.FetchDisable{}.Call(b)
			if err == nil {
				err = e
			}
		}()

		<-waitPaused

		err = proto.FetchContinueRequest{
			RequestID: paused.RequestID,
		}.Call(b)
		if err != nil {
			return
		}

		<-waitAuth

		err = proto.FetchContinueWithAuth{
			RequestID: auth.RequestID,
			AuthChallengeResponse: &proto.FetchAuthChallengeResponse{
				Response: proto.FetchAuthChallengeResponseResponseProvideCredentials,
				Username: username,
				Password: password,
			},
		}.Call(b)

		return
	}, nil
}

// CallContext parameters for proto
func (b *Browser) CallContext() (context.Context, proto.Client, string) {
	return b.ctx, b.client, ""
}

// PageFromTargetIDE creates a Page instance from a targetID
func (b *Browser) PageFromTargetIDE(targetID proto.TargetTargetID) (*Page, error) {
	page := (&Page{
		browser:             b,
		TargetID:            targetID,
		getDownloadFileLock: &sync.Mutex{},
		viewport:            &proto.EmulationSetDeviceMetricsOverride{},
	}).Context(b.ctx)

	page.Mouse = &Mouse{page: page, id: kit.RandString(8)}
	page.Keyboard = &Keyboard{page: page}

	return page, page.initSession()
}

func (b *Browser) initEvents() error {
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

	err := proto.TargetSetDiscoverTargets{
		Discover: true,
	}.Call(b)

	return err
}
