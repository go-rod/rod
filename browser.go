package rod

import (
	"context"
	"time"

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

	// Slowmotion delay each chrome control action
	Slowmotion time.Duration

	// Trace enables the visual tracing of the device input on the page
	Trace bool

	// OnFatal calls when a fatal error happens
	OnFatal func(error)

	ctx    context.Context
	close  func()
	client *cdp.Client
	event  *kit.Observable
	fatal  *kit.Observable
}

// OpenE ...
func (b *Browser) OpenE() (*Browser, error) {
	if b.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		b.ctx = ctx
		b.close = cancel
	}

	if _, err := cdp.GetWebSocketDebuggerURL(b.ControlURL); err != nil {
		u, err := cdp.LaunchBrowser("", !b.Foreground)
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

// CloseE ...
func (b *Browser) CloseE() error {
	_, err := b.Call(b.ctx, &cdp.Message{Method: "Browser.close"})
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

// Ctx creates a clone with specified context
func (b *Browser) Ctx(ctx context.Context) *Browser {
	newObj := *b

	newObj.ctx = ctx

	return &newObj
}

// PageE ...
func (b *Browser) PageE(url string) (*Page, error) {
	target, err := b.Call(b.ctx, &cdp.Message{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": url,
		},
	})
	if err != nil {
		return nil, err
	}

	return b.page(target.Get("targetId").String())
}

// Page creates a new page
func (b *Browser) Page(url string) *Page {
	p, err := b.PageE(url)
	kit.E(err)
	return p
}

// WaitPageE ...
func (b *Browser) WaitPageE(p *Page) (*Page, error) {
	var targetInfo cdp.Object

	_, err := b.event.Until(b.ctx, func(e kit.Event) bool {
		msg := e.(*cdp.Message)
		if msg.Method == "Target.targetCreated" {
			targetInfo = msg.Params.(map[string]interface{})["targetInfo"].(map[string]interface{})

			if targetInfo["openerId"] == p.TargetID {
				return true
			}
		}
		return false
	})

	if err != nil {
		return nil, err
	}

	return b.page(targetInfo["targetId"].(string))
}

// WaitPage to be opened from the specified page
func (b *Browser) WaitPage(p *Page) *Page {
	newPage, err := b.WaitPageE(p)
	kit.E(err)
	return newPage
}

// PagesE ...
func (b *Browser) PagesE() ([]*Page, error) {
	list, err := b.Call(b.ctx, &cdp.Message{Method: "Target.getTargets"})
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

// Call sends a control message to browser
func (b *Browser) Call(ctx context.Context, msg *cdp.Message) (kit.JSONResult, error) {
	b.slowmotion(msg.Method)

	return b.client.Call(ctx, msg)
}

// Event returns the observable for browser events
func (b *Browser) Event() *kit.Observable {
	return b.event
}

func (b *Browser) page(targetID string) (*Page, error) {
	page := &Page{
		ctx:      b.ctx,
		browser:  b,
		TargetID: targetID,
	}

	page.Mouse = &Mouse{
		ctx:  b.ctx,
		page: page,
	}

	page.Keyboard = &Keyboard{
		ctx:  b.ctx,
		page: page,
	}

	return page, page.initSession()
}

func (b *Browser) initEvents() error {
	b.event = kit.NewObservable()
	b.fatal = kit.NewObservable()

	go func() {
		for msg := range b.client.Event() {
			go b.event.Publish(msg)
		}
	}()

	go func() {
		for err := range b.client.Fatal() {
			go b.fatal.Publish(err)
		}
	}()

	go func() {
		for err := range b.fatal.Subscribe() {
			if b.OnFatal == nil {
				kit.Err(err)
			} else {
				b.OnFatal(err.(error))
			}
		}
	}()

	_, err := b.Call(b.ctx, &cdp.Message{
		Method: "Target.setDiscoverTargets",
		Params: cdp.Object{"discover": true},
	})

	return err
}
