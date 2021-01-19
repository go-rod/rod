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

// Area of the polygon
// https://en.wikipedia.org/wiki/Polygon#Area
func (q DOMQuad) Area() float64 {
	area := 0.0
	l := len(q)/2 - 1

	for i := 0; i < l; i++ {
		area += q[i*2]*q[i*2+3] - q[i*2+2]*q[i*2+1]
	}
	area += q[l*2]*q[1] - q[0]*q[l*2+1]

	return area / 2
}

// OnePointInside the shape
func (res *DOMGetContentQuadsResult) OnePointInside() *Point {
	for _, q := range res.Quads {
		if q.Area() >= 1 {
			pt := q.Center()
			return &pt
		}
	}

	return nil
}

// Box returns the smallest leveled rectangle that can cover the whole shape.
func (res *DOMGetContentQuadsResult) Box() (box *DOMRect) {
	return Shape(res.Quads).Box()
}

// Shape is a list of DOMQuad
type Shape []DOMQuad

// Box returns the smallest leveled rectangle that can cover the whole shape.
func (qs Shape) Box() (box *DOMRect) {
	if len(qs) == 0 {
		return
	}

	left := qs[0][0]
	top := qs[0][1]
	right := left
	bottom := top

	for _, q := range qs {
		q.Each(func(pt Point, _ int) {
			if pt.X < left {
				left = pt.X
			}
			if pt.Y < top {
				top = pt.Y
			}
			if pt.X > right {
				right = pt.X
			}
			if pt.Y > bottom {
				bottom = pt.Y
			}
		})
	}

	box = &DOMRect{left, top, right - left, bottom - top}

	return
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
