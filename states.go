package rod

import (
	"reflect"

	"github.com/go-rod/rod/lib/proto"
)

type stateKey struct {
	browserContextID proto.BrowserBrowserContextID
	sessionID        proto.TargetSessionID
	methodName       string
}

func (b *Browser) key(sessionID proto.TargetSessionID, methodName string) stateKey {
	return stateKey{
		browserContextID: b.BrowserContextID,
		sessionID:        sessionID,
		methodName:       methodName,
	}
}

func (b *Browser) set(sessionID proto.TargetSessionID, methodName string, params interface{}) {
	b.states.Store(b.key(sessionID, methodName), params)

	key := ""
	switch methodName {
	case (proto.TargetSetDiscoverTargets{}).ProtoName(): // only Target domain is special
		method := reflect.Indirect(reflect.ValueOf(params)).Interface().(proto.TargetSetDiscoverTargets)
		if !method.Discover {
			key = (proto.TargetSetDiscoverTargets{}).ProtoName()
		}
	case (proto.EmulationClearDeviceMetricsOverride{}).ProtoName():
		key = (proto.EmulationSetDeviceMetricsOverride{}).ProtoName()
	case (proto.EmulationClearGeolocationOverride{}).ProtoName():
		key = (proto.EmulationSetGeolocationOverride{}).ProtoName()
	default:
		domain, name := proto.ParseMethodName(methodName)
		if name == "disable" {
			key = domain + ".enable"
		}
	}
	if key != "" {
		b.states.Delete(b.key(sessionID, key))
	}
}

// LoadState into the method, seesionID can be empty.
func (b *Browser) LoadState(sessionID proto.TargetSessionID, method proto.Request) (has bool) {
	data, has := b.states.Load(b.key(sessionID, method.ProtoName()))
	if has {
		reflect.Indirect(reflect.ValueOf(method)).Set(
			reflect.Indirect(reflect.ValueOf(data)),
		)
	}
	return
}

// EnableDomain and returns a recover function to restore previous state
func (b *Browser) EnableDomain(sessionID proto.TargetSessionID, req proto.Request) (recover func()) {
	_, enabled := b.states.Load(b.key(sessionID, req.ProtoName()))

	if !enabled {
		_, _ = b.Call(b.ctx, string(sessionID), req.ProtoName(), req)
	}

	return func() {
		if !enabled {
			if req.ProtoName() == (proto.TargetSetDiscoverTargets{}).ProtoName() { // only Target domain is special
				_ = proto.TargetSetDiscoverTargets{Discover: false}.Call(b)
				return
			}

			domain, _ := proto.ParseMethodName(req.ProtoName())
			_, _ = b.Call(b.ctx, string(sessionID), domain+".disable", nil)
		}
	}
}

// DisableDomain and returns a recover function to restore previous state
func (b *Browser) DisableDomain(sessionID proto.TargetSessionID, req proto.Request) (recover func()) {
	_, enabled := b.states.Load(b.key(sessionID, req.ProtoName()))
	domain, _ := proto.ParseMethodName(req.ProtoName())

	if enabled {
		if req.ProtoName() == (proto.TargetSetDiscoverTargets{}).ProtoName() { // only Target domain is special
			_ = proto.TargetSetDiscoverTargets{Discover: false}.Call(b)
		} else {
			_, _ = b.Call(b.ctx, string(sessionID), domain+".disable", nil)
		}
	}

	return func() {
		if enabled {
			_, _ = b.Call(b.ctx, string(sessionID), req.ProtoName(), req)
		}
	}
}

func (b *Browser) storePage(page *Page) {
	b.states.Store(page.TargetID, page)
}

func (b *Browser) loadPage(id proto.TargetTargetID) *Page {
	if cache, ok := b.states.Load(id); ok {
		return cache.(*Page)
	}
	return nil
}

// LoadState into the method.
func (p *Page) LoadState(method proto.Request) (has bool) {
	return p.browser.LoadState(p.SessionID, method)
}

// EnableDomain and returns a recover function to restore previous state
func (p *Page) EnableDomain(method proto.Request) (recover func()) {
	return p.browser.Context(p.ctx).EnableDomain(p.SessionID, method)
}

// DisableDomain and returns a recover function to restore previous state
func (p *Page) DisableDomain(method proto.Request) (recover func()) {
	return p.browser.Context(p.ctx).DisableDomain(p.SessionID, method)
}

func (p *Page) cleanupStates() {
	p.browser.states.Delete(p.TargetID)
}
