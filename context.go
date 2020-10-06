package rod

import (
	"context"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

type timeoutContextKey struct{}
type timeoutContextVal struct {
	parent context.Context
	cancel context.CancelFunc
}

// Context creates a clone with a context that inherits the previous one
func (b *Browser) Context(ctx context.Context) *Browser {
	newObj := *b
	newObj.ctx = ctx
	return &newObj
}

// GetContext interface
func (b *Browser) GetContext() context.Context {
	return b.ctx
}

// Timeout for chained sub-operations
func (b *Browser) Timeout(d time.Duration) *Browser {
	ctx, cancel := context.WithTimeout(b.ctx, d)
	return b.Context(context.WithValue(ctx, timeoutContextKey{}, &timeoutContextVal{b.ctx, cancel}))
}

// CancelTimeout context and reset the context to previous one
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

// Sleeper for chained sub-operations
func (b *Browser) Sleeper(sleeper func() utils.Sleeper) *Browser {
	newObj := *b
	newObj.sleeper = ensureSleeper(sleeper)
	return &newObj
}

// Context creates a clone with a context that inherits the previous one
func (p *Page) Context(ctx context.Context) *Page {
	newObj := *p
	newObj.ctx = ctx
	return &newObj
}

// GetContext interface
func (p *Page) GetContext() context.Context {
	return p.ctx
}

// Timeout for chained sub-operations
func (p *Page) Timeout(d time.Duration) *Page {
	ctx, cancel := context.WithTimeout(p.ctx, d)
	return p.Context(context.WithValue(ctx, timeoutContextKey{}, &timeoutContextVal{p.ctx, cancel}))
}

// CancelTimeout context and reset the context to previous one
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

// Sleeper for chained sub-operations
func (p *Page) Sleeper(sleeper func() utils.Sleeper) *Page {
	newObj := *p
	newObj.sleeper = ensureSleeper(sleeper)
	return &newObj
}

// Context creates a clone with a context that inherits the previous one
func (el *Element) Context(ctx context.Context) *Element {
	newObj := *el
	newObj.ctx = ctx
	return &newObj
}

// GetContext interface
func (el *Element) GetContext() context.Context {
	return el.ctx
}

// Timeout for chained sub-operations
func (el *Element) Timeout(d time.Duration) *Element {
	ctx, cancel := context.WithTimeout(el.ctx, d)
	return el.Context(context.WithValue(ctx, timeoutContextKey{}, &timeoutContextVal{el.ctx, cancel}))
}

// CancelTimeout context and reset the context to previous one
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

// Sleeper for chained sub-operations
func (el *Element) Sleeper(sleeper func() utils.Sleeper) *Element {
	newObj := *el
	newObj.sleeper = ensureSleeper(sleeper)
	return &newObj
}
