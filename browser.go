package rod

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Browser represents the browser
type Browser struct {
	// ControlURL is the url to remote control browser.
	// If fails to connect to it, rod will try to open a local browser.
	ControlURL string

	// Foreground enables the browser to run on foreground mode
	Foreground bool

	// OnEvent calls when a browser event happens
	OnEvent func(*cdp.Message)

	// OnFatal calls when a fatal error happens
	OnFatal func(error)

	ctx    context.Context
	close  func()
	client *cdp.Client
}

// OpenE ...
func (b *Browser) OpenE() (*Browser, error) {
	if b.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		b.ctx = ctx
		b.close = cancel
	}

	if _, err := cdp.GetWebSocketDebuggerURL(b.ControlURL); err != nil {
		u, err := cdp.LaunchBrowser(cdp.FindChrome(), !b.Foreground)
		if err != nil {
			return nil, err
		}
		b.ControlURL = u
	}

	client, err := cdp.New(b.ctx, b.ControlURL)
	if err != nil {
		return nil, err
	}

	go func() {
		if b.OnFatal == nil {
			kit.Err(<-client.Fatal())
		}
		b.OnFatal(<-client.Fatal())
	}()

	go func() {
		for msg := range client.Event() {
			if b.OnEvent != nil {
				b.OnEvent(msg)
			}
		}
	}()

	b.client = client

	go func() {
		<-b.ctx.Done()
		_, err := client.Call(nil, &cdp.Message{Method: "Browser.close"})
		if err != nil {
			kit.Err(err)
		}
	}()

	return b, nil
}

// Open a new browser controller
func Open(b *Browser) *Browser {
	if b == nil {
		b = &Browser{}
	}

	kit.E(b.OpenE())

	return b
}

// Close the browser and release related resources
func (b *Browser) Close() {
	if b.close != nil {
		b.close()
	}
}

// Ctx creates a clone with specified context
func (b *Browser) Ctx(ctx context.Context) *Browser {
	newObj := *b

	newObj.ctx = ctx

	return &newObj
}

// PageE ...
func (b *Browser) PageE(url string) (*Page, error) {
	target, err := b.client.Call(b.ctx, &cdp.Message{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": url,
		},
	})
	if err != nil {
		return nil, err
	}

	page := &Page{
		ctx:      b.ctx,
		browser:  b,
		TargetID: target.Get("targetId").String(),
	}

	page.mouse = &Mouse{
		ctx:  b.ctx,
		page: page,
	}

	return page, page.initSession()
}

// Page creates a new page
func (b *Browser) Page(url string) *Page {
	p, err := b.PageE(url)
	kit.E(err)
	return p
}
