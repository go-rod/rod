// Patches to normalize the proto types

package proto

import (
	"encoding/json"
	"strconv"
	"time"
)

// TimeSinceEpoch UTC time in seconds, counted from January 1, 1970.
type TimeSinceEpoch struct {
	time.Time
}

// UnmarshalJSON interface
func (t *TimeSinceEpoch) UnmarshalJSON(b []byte) error {
	v, _ := strconv.ParseFloat(string(b), 64)
	t.Time = (time.Unix(0, 0)).Add(
		time.Duration(v * float64(time.Second)),
	)
	return nil
}

// MarshalJSON interface
func (t TimeSinceEpoch) MarshalJSON() ([]byte, error) {
	d := float64(t.Time.UnixNano()) / float64(time.Second)
	return json.Marshal(d)
}

// MonotonicTime Monotonically increasing time in seconds since an arbitrary point in the past.
type MonotonicTime struct {
	time.Duration
}

// UnmarshalJSON interface
func (t *MonotonicTime) UnmarshalJSON(b []byte) error {
	v, _ := strconv.ParseFloat(string(b), 64)
	t.Duration = time.Duration(v * float64(time.Second))
	return nil
}

// MarshalJSON interface
func (t MonotonicTime) MarshalJSON() ([]byte, error) {
	d := float64(t.Duration) / float64(time.Second)
	return json.Marshal(d)
}

type inputDispatchMouseEvent struct {
	Type        InputDispatchMouseEventType        `json:"type"`
	X           float64                            `json:"x"`
	Y           float64                            `json:"y"`
	Modifiers   int                                `json:"modifiers,omitempty"`
	Timestamp   *TimeSinceEpoch                    `json:"timestamp,omitempty"`
	Button      InputMouseButton                   `json:"button,omitempty"`
	Buttons     int                                `json:"buttons,omitempty"`
	ClickCount  int                                `json:"clickCount,omitempty"`
	DeltaX      float64                            `json:"deltaX,omitempty"`
	DeltaY      float64                            `json:"deltaY,omitempty"`
	PointerType InputDispatchMouseEventPointerType `json:"pointerType,omitempty"`
}

type inputDispatchMouseWheelEvent struct {
	Type        InputDispatchMouseEventType        `json:"type"`
	X           float64                            `json:"x"`
	Y           float64                            `json:"y"`
	Modifiers   int                                `json:"modifiers,omitempty"`
	Timestamp   *TimeSinceEpoch                    `json:"timestamp,omitempty"`
	Button      InputMouseButton                   `json:"button,omitempty"`
	Buttons     int                                `json:"buttons,omitempty"`
	ClickCount  int                                `json:"clickCount,omitempty"`
	DeltaX      float64                            `json:"deltaX"`
	DeltaY      float64                            `json:"deltaY"`
	PointerType InputDispatchMouseEventPointerType `json:"pointerType,omitempty"`
}

// MarshalJSON interface
// TODO: make sure deltaX and deltaY are never omitted. Or it will cause a browser bug.
func (e InputDispatchMouseEvent) MarshalJSON() ([]byte, error) {
	var ee interface{}

	if e.Type == InputDispatchMouseEventTypeMouseWheel {
		ee = &inputDispatchMouseWheelEvent{}
	} else {
		ee = &inputDispatchMouseEvent{}
	}

	assign(e, ee)

	return json.Marshal(ee)
}

// Point from the origin (0, 0)
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Len is the number of vertices
func (q DOMQuad) Len() int {
	return len(q) / 2
}

// Each point
func (q DOMQuad) Each(fn func(pt Point, i int)) {
	for i := 0; i < q.Len(); i++ {
		fn(Point{q[i*2], q[i*2+1]}, i)
	}
}

// Center of the polygon
func (q DOMQuad) Center() Point {
	var x, y float64
	q.Each(func(pt Point, _ int) {
		x += pt.X
		y += pt.Y
	})
	return Point{x / float64(q.Len()), y / float64(q.Len())}
}

// OnePointInside the shape
func (res *DOMGetContentQuadsResult) OnePointInside() *Point {
	if len(res.Quads) == 0 {
		return nil
	}

	center := res.Quads[0].Center()

	return &center
}

// MoveTo X and Y to x and y
func (p *InputTouchPoint) MoveTo(x, y float64) {
	p.X = x
	p.Y = y
}

// CookiesToParams converts Cookies list to NetworkCookieParam list
func CookiesToParams(cookies []*NetworkCookie) []*NetworkCookieParam {
	list := []*NetworkCookieParam{}
	for _, c := range cookies {
		list = append(list, &NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
			SameSite: c.SameSite,
			Expires:  c.Expires,
			Priority: c.Priority,
		})
	}
	return list
}
