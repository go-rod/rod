package rod

import (
	"context"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

type (
	timeoutContextKey struct{}
	timeoutContextVal struct {
		parent context.Context
		cancel context.CancelFunc
	}
)

// Context returns a clone with the specified ctx for chained sub-operations
func (b *Browser) Context(ctx context.Context) *Browser {
	newObj := *b
	newObj.ctx = ctx
	return &newObj
}

// GetContext of current instance
func (b *Browser) GetContext() context.Context {
	return b.ctx
}

// Timeout returns a clone with the specified total timeout of all chained sub-operations
func (b *Browser) Timeout(d time.Duration) *Browser {
	ctx, cancel := context.WithTimeout(b.ctx, d)
	return b.Context(context.WithValue(ctx, timeoutContextKey{}, &timeoutContextVal{b.ctx, cancel}))
}

// CancelTimeout cancels the current timeout context and returns a clone with the parent context
func (b *Browser) CancelTimeout() *Browser {
	val := b.ctx.Value(timeoutContextKey{}).(*timeoutContextVal)
	val.cancel()
	return b.Context(val.parent)
}

// WithCancel returns a clone with a context cancel function
func (b *Browser) WithCancel() (*Browser, func()) {
	ctx, cancel := context.WithCancel(b.ctx)
	return b.Context(ctx), cancel
}

// Sleeper returns a clone with the specified sleeper for chained sub-operations
func (b *Browser) Sleeper(sleeper func() utils.Sleeper) *Browser {
	newObj := *b
	newObj.sleeper = sleeper
	return &newObj
}

// Context returns a clone with the specified ctx for chained sub-operations
func (p *Page) Context(ctx context.Context) *Page {
	newObj := *p
	newObj.ctx = ctx
	return &newObj
}

// GetContext of current instance
func (p *Page) GetContext() context.Context {
	return p.ctx
}

// Timeout returns a clone with the specified total timeout of all chained sub-operations
func (p *Page) Timeout(d time.Duration) *Page {
	ctx, cancel := context.WithTimeout(p.ctx, d)
	return p.Context(context.WithValue(ctx, timeoutContextKey{}, &timeoutContextVal{p.ctx, cancel}))
}

// CancelTimeout cancels the current timeout context and returns a clone with the parent context
func (p *Page) CancelTimeout() *Page {
	val := p.ctx.Value(timeoutContextKey{}).(*timeoutContextVal)
	val.cancel()
	return p.Context(val.parent)
}

// WithCancel returns a clone with a context cancel function
func (p *Page) WithCancel() (*Page, func()) {
	ctx, cancel := context.WithCancel(p.ctx)
	return p.Context(ctx), cancel
}

// Sleeper returns a clone with the specified sleeper for chained sub-operations
func (p *Page) Sleeper(sleeper func() utils.Sleeper) *Page {
	newObj := *p
	newObj.sleeper = sleeper
	return &newObj
}

// Context returns a clone with the specified ctx for chained sub-operations
func (el *Element) Context(ctx context.Context) *Element {
	newObj := *el
	newObj.ctx = ctx
	return &newObj
}

// GetContext of current instance
func (el *Element) GetContext() context.Context {
	return el.ctx
}

// Timeout returns a clone with the specified total timeout of all chained sub-operations
func (el *Element) Timeout(d time.Duration) *Element {
	ctx, cancel := context.WithTimeout(el.ctx, d)
	return el.Context(context.WithValue(ctx, timeoutContextKey{}, &timeoutContextVal{el.ctx, cancel}))
}

// CancelTimeout cancels the current timeout context and returns a clone with the parent context
func (el *Element) CancelTimeout() *Element {
	val := el.ctx.Value(timeoutContextKey{}).(*timeoutContextVal)
	val.cancel()
	return el.Context(val.parent)
}

// WithCancel returns a clone with a context cancel function
func (el *Element) WithCancel() (*Element, func()) {
	ctx, cancel := context.WithCancel(el.ctx)
	return el.Context(ctx), cancel
}

// Sleeper returns a clone with the specified sleeper for chained sub-operations
func (el *Element) Sleeper(sleeper func() utils.Sleeper) *Element {
	newObj := *el
	newObj.sleeper = sleeper
	return &newObj
}
