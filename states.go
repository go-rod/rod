package rod

import (
	"context"
	"encoding/json"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/proto"
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
	case "Target.setDiscoverTargets": // only Target domain is special
		method := &proto.TargetSetDiscoverTargets{}
		kit.E(json.Unmarshal(params, method))
		if !method.Discover {
			key = "Target.setDiscoverTargets"
		}
	case "Emulation.clearDeviceMetricsOverride":
		key = "Emulation.setDeviceMetricsOverride"
	case "Emulation.clearGeolocationOverride":
		key = "Emulation.setGeolocationOverride"
	default:
		domain, name := proto.ParseMethodName(methodName)
		if name == "disable" {
			key = domain + ".enable"
		}
	}
	b.states.Delete(b.key(sessionID, key))
}

// LoadState into the method, seesionID can be empty.
func (b *Browser) LoadState(sessionID proto.TargetSessionID, method proto.Payload) (has bool) {
	data, has := b.states.Load(b.key(sessionID, method.MethodName()))
	if has {
		kit.E(json.Unmarshal(data.(json.RawMessage), method))
	}
	return
}

// LoadState into the method.
func (p *Page) LoadState(method proto.Payload) (has bool) {
	return p.browser.LoadState(p.SessionID, method)
}

// EnableDomain and returns a recover function to restore previous state
func (b *Browser) EnableDomain(ctx context.Context, sessionID proto.TargetSessionID, method proto.Payload) (recover func()) {
	_, enabled := b.states.Load(b.key(sessionID, method.MethodName()))

	if !enabled {
		payload, _ := proto.Normalize(method)
		_, _ = b.Call(ctx, string(sessionID), method.MethodName(), payload)
	}

	return func() {
		if !enabled {
			if method.MethodName() == "Target.setDiscoverTargets" { // only Target domain is special
				_ = proto.TargetSetDiscoverTargets{Discover: false}.Call(b)
				return
			}

			domain, _ := proto.ParseMethodName(method.MethodName())
			_, _ = b.Call(ctx, string(sessionID), domain+".disable", nil)
		}
	}
}

// DisableDomain and returns a recover function to restore previous state
func (b *Browser) DisableDomain(ctx context.Context, sessionID proto.TargetSessionID, method proto.Payload) (recover func()) {
	_, enabled := b.states.Load(b.key(sessionID, method.MethodName()))
	domain, _ := proto.ParseMethodName(method.MethodName())

	if enabled {
		if method.MethodName() == "Target.setDiscoverTargets" { // only Target domain is special
			_ = proto.TargetSetDiscoverTargets{Discover: false}.Call(b)
		} else {
			_, _ = b.Call(ctx, string(sessionID), domain+".disable", nil)
		}
	}

	return func() {
		if enabled {
			payload, _ := proto.Normalize(method)
			_, _ = b.Call(ctx, string(sessionID), method.MethodName(), payload)
		}
	}
}

// EnableDomain and returns a recover function to restore previous state
func (p *Page) EnableDomain(method proto.Payload) (recover func()) {
	return p.browser.EnableDomain(p.ctx, p.SessionID, method)
}

// DisableDomain and returns a recover function to restore previous state
func (p *Page) DisableDomain(method proto.Payload) (recover func()) {
	return p.browser.DisableDomain(p.ctx, p.SessionID, method)
}
