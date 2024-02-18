package rod

import (
	"context"
	"fmt"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// TryError error.
type TryError struct {
	Value interface{}
	Stack string
}

func (e *TryError) Error() string {
	return fmt.Sprintf("error value: %#v\n%s", e.Value, e.Stack)
}

// Is interface.
func (e *TryError) Is(err error) bool { _, ok := err.(*TryError); return ok }

// Unwrap stdlib interface.
func (e *TryError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return fmt.Errorf("%v", e.Value)
}

// ExpectElementError error.
type ExpectElementError struct {
	*proto.RuntimeRemoteObject
}

func (e *ExpectElementError) Error() string {
	return fmt.Sprintf("expect js to return an element, but got: %s", utils.MustToJSON(e))
}

// Is interface.
func (e *ExpectElementError) Is(err error) bool { _, ok := err.(*ExpectElementError); return ok }

// ExpectElementsError error.
type ExpectElementsError struct {
	*proto.RuntimeRemoteObject
}

func (e *ExpectElementsError) Error() string {
	return fmt.Sprintf("expect js to return an array of elements, but got: %s", utils.MustToJSON(e))
}

// Is interface.
func (e *ExpectElementsError) Is(err error) bool { _, ok := err.(*ExpectElementsError); return ok }

// ElementNotFoundError error.
type ElementNotFoundError struct{}

func (e *ElementNotFoundError) Error() string {
	return "cannot find element"
}

// NotFoundSleeper returns ErrElementNotFound on the first call.
func NotFoundSleeper() utils.Sleeper {
	return func(context.Context) error {
		return &ElementNotFoundError{}
	}
}

// ObjectNotFoundError error.
type ObjectNotFoundError struct {
	*proto.RuntimeRemoteObject
}

func (e *ObjectNotFoundError) Error() string {
	return fmt.Sprintf("cannot find object: %s", utils.MustToJSON(e))
}

// Is interface.
func (e *ObjectNotFoundError) Is(err error) bool { _, ok := err.(*ObjectNotFoundError); return ok }

// EvalError error.
type EvalError struct {
	*proto.RuntimeExceptionDetails
}

func (e *EvalError) Error() string {
	exp := e.Exception
	return fmt.Sprintf("eval js error: %s %s", exp.Description, exp.Value)
}

// Is interface.
func (e *EvalError) Is(err error) bool { _, ok := err.(*EvalError); return ok }

// NavigationError error.
type NavigationError struct {
	Reason string
}

func (e *NavigationError) Error() string {
	return "navigation failed: " + e.Reason
}

// Is interface.
func (e *NavigationError) Is(err error) bool { _, ok := err.(*NavigationError); return ok }

// PageCloseCanceledError error.
type PageCloseCanceledError struct{}

func (e *PageCloseCanceledError) Error() string {
	return "page close canceled"
}

// NotInteractableError error. Check the doc of Element.Interactable for details.
type NotInteractableError struct{}

func (e *NotInteractableError) Error() string {
	return "element is not cursor interactable"
}

// InvisibleShapeError error.
type InvisibleShapeError struct {
	*Element
}

// Error ...
func (e *InvisibleShapeError) Error() string {
	return fmt.Sprintf("element has no visible shape or outside the viewport: %s", e.String())
}

// Is interface.
func (e *InvisibleShapeError) Is(err error) bool { _, ok := err.(*InvisibleShapeError); return ok }

// Unwrap ...
func (e *InvisibleShapeError) Unwrap() error {
	return &NotInteractableError{}
}

// CoveredError error.
type CoveredError struct {
	*Element
}

// Error ...
func (e *CoveredError) Error() string {
	return fmt.Sprintf("element covered by: %s", e.String())
}

// Unwrap ...
func (e *CoveredError) Unwrap() error {
	return &NotInteractableError{}
}

// Is interface.
func (e *CoveredError) Is(err error) bool { _, ok := err.(*CoveredError); return ok }

// NoPointerEventsError error.
type NoPointerEventsError struct {
	*Element
}

// Error ...
func (e *NoPointerEventsError) Error() string {
	return fmt.Sprintf("element's pointer-events is none: %s", e.String())
}

// Unwrap ...
func (e *NoPointerEventsError) Unwrap() error {
	return &NotInteractableError{}
}

// Is interface.
func (e *NoPointerEventsError) Is(err error) bool { _, ok := err.(*NoPointerEventsError); return ok }

// PageNotFoundError error.
type PageNotFoundError struct{}

func (e *PageNotFoundError) Error() string {
	return "cannot find page"
}

// NoShadowRootError error.
type NoShadowRootError struct {
	*Element
}

// Error ...
func (e *NoShadowRootError) Error() string {
	return fmt.Sprintf("element has no shadow root: %s", e.String())
}

// Is interface.
func (e *NoShadowRootError) Is(err error) bool { _, ok := err.(*NoShadowRootError); return ok }
