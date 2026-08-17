package rod

import (
	"sync"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

func TestPageSetHelperAfterContextReset(t *testing.T) {
	ctxID := proto.RuntimeRemoteObjectID("context")
	fnID := proto.RuntimeRemoteObjectID("function")

	for _, reset := range []struct {
		name       string
		fn         func(*Page)
		wantStored bool
	}{
		{
			name: "all contexts",
			fn: func(p *Page) {
				p.helpers = nil
			},
		},
		{
			name: "one context",
			fn: func(p *Page) {
				delete(p.helpers, ctxID)
			},
		},
		{
			name:       "same context",
			fn:         func(*Page) {},
			wantStored: true,
		},
	} {
		t.Run(reset.name, func(t *testing.T) {
			p := &Page{helpersLock: &sync.Mutex{}}
			p.getHelper(ctxID, "helper")

			p.helpersLock.Lock()
			reset.fn(p)
			p.helpersLock.Unlock()

			p.setHelper(ctxID, "helper", fnID)

			p.helpersLock.Lock()
			got := p.helpers[ctxID]["helper"]
			p.helpersLock.Unlock()
			if stored := got == fnID; stored != reset.wantStored {
				t.Fatalf("helper stored = %t; want %t", stored, reset.wantStored)
			}
		})
	}
}
