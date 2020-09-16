package rod

import (
	"encoding/json"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
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

func (b *Browser) set(sessionID proto.TargetSessionID, methodName string, params json.RawMessage) {
	b.states.Store(b.key(sessionID, methodName), params)

	key := ""
	switch methodName {
	case (proto.TargetSetDiscoverTargets{}).MethodName(): // only Target domain is special
		method := &proto.TargetSetDiscoverTargets{}
		utils.E(json.Unmarshal(params, method))
		if !method.Discover {
			key = (proto.TargetSetDiscoverTargets{}).MethodName()
		}
	case (proto.EmulationClearDeviceMetricsOverride{}).MethodName():
		key = (proto.EmulationSetDeviceMetricsOverride{}).MethodName()
	case (proto.EmulationClearGeolocationOverride{}).MethodName():
		key = (proto.EmulationSetGeolocationOverride{}).MethodName()
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
func (b *Browser) LoadState(sessionID proto.TargetSessionID, method proto.Payload) (has bool) {
	data, has := b.states.Load(b.key(sessionID, method.MethodName()))
	if has {
		utils.E(json.Unmarshal(data.(json.RawMessage), method))
	}
	return
}

// EnableDomain and returns a recover function to restore previous state
func (b *Browser) EnableDomain(sessionID proto.TargetSessionID, method proto.Payload) (recover func()) {
	_, enabled := b.states.Load(b.key(sessionID, method.MethodName()))

	if !enabled {
		payload, err := proto.Normalize(method)
		utils.E(err)
		_, _ = b.Call(b.ctx, string(sessionID), method.MethodName(), payload)
	}

	return func() {
		if !enabled {
			if method.MethodName() == (proto.TargetSetDiscoverTargets{}).MethodName() { // only Target domain is special
				_ = proto.TargetSetDiscoverTargets{Discover: false}.Call(b)
				return
			}

			domain, _ := proto.ParseMethodName(method.MethodName())
			_, _ = b.Call(b.ctx, string(sessionID), domain+".disable", nil)
		}
	}
}

// DisableDomain and returns a recover function to restore previous state
func (b *Browser) DisableDomain(sessionID proto.TargetSessionID, method proto.Payload) (recover func()) {
	_, enabled := b.states.Load(b.key(sessionID, method.MethodName()))
	domain, _ := proto.ParseMethodName(method.MethodName())

	if enabled {
		if method.MethodName() == (proto.TargetSetDiscoverTargets{}).MethodName() { // only Target domain is special
			_ = proto.TargetSetDiscoverTargets{Discover: false}.Call(b)
		} else {
			_, _ = b.Call(b.ctx, string(sessionID), domain+".disable", nil)
		}
	}

	return func() {
		if enabled {
			payload, err := proto.Normalize(method)
			utils.E(err)
			_, _ = b.Call(b.ctx, string(sessionID), method.MethodName(), payload)
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
func (p *Page) LoadState(method proto.Payload) (has bool) {
	return p.browser.LoadState(p.SessionID, method)
}

// EnableDomain and returns a recover function to restore previous state
func (p *Page) EnableDomain(method proto.Payload) (recover func()) {
	return p.browser.Context(p.ctx).EnableDomain(p.SessionID, method)
}

// DisableDomain and returns a recover function to restore previous state
func (p *Page) DisableDomain(method proto.Payload) (recover func()) {
	return p.browser.Context(p.ctx).DisableDomain(p.SessionID, method)
}

func (p *Page) cleanupStates() {
	p.browser.states.Delete(p.TargetID)
}
