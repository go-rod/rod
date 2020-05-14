package rod

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ysmood/rod/lib/cdp"
)

// Observable is a thread-safe event helper
type Observable struct {
	subscribers *sync.Map
	idCount     int64
}

// NewObservable creates a new observable
func NewObservable() *Observable {
	return &Observable{
		subscribers: &sync.Map{},
	}
}

// Publish event to all subscribers, no internal goroutine is used,
// so the publish can block the goroutine. Use goroutine or buffer to prevent the blocking.
func (o *Observable) Publish(e *cdp.Event) {
	o.subscribers.Range(func(_, s interface{}) (goOn bool) {
		goOn = true
		defer func() { _ = recover() }()
		s.(*Subscriber).C <- e
		return
	})
}

// Subscribe returns a subscriber to emit events
func (o *Observable) Subscribe() *Subscriber {
	id := atomic.AddInt64(&o.idCount, 1)

	subscriber := &Subscriber{
		C:  make(chan *cdp.Event),
		id: id,
	}

	o.subscribers.Store(id, subscriber)

	return subscriber
}

// Unsubscribe from the observable
func (o *Observable) Unsubscribe(s *Subscriber) {
	defer func() { _ = recover() }()

	close(s.C)
	o.subscribers.Delete(s.id)
}

// UnsubscribeAll current subscribers
func (o *Observable) UnsubscribeAll() {
	defer func() { _ = recover() }()

	o.subscribers.Range(func(_, s interface{}) bool {
		o.Unsubscribe(s.(*Subscriber))
		return true
	})
}

// Count returns the number of subscribers
func (o *Observable) Count() int {
	c := 0
	o.subscribers.Range(func(key, value interface{}) bool {
		c++
		return true
	})
	return c
}

// Until check returns true keep waiting
func (o *Observable) Until(ctx context.Context, check func(*cdp.Event) bool) (*cdp.Event, error) {
	s := o.Subscribe()
	defer o.Unsubscribe(s)

	for {
		select {
		case e := <-s.C:
			if check(e) {
				return e, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Subscriber of the observable
type Subscriber struct {
	C  chan *cdp.Event
	id int64
}

// Filter events
func (s *Subscriber) Filter(filter func(*cdp.Event) bool) chan *cdp.Event {
	filtered := make(chan *cdp.Event)
	go func() {
		for e := range s.C {
			if filter(e) {
				filtered <- e
			}
		}
		close(filtered)
	}()
	return filtered
}
