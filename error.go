package rod

import (
	"context"
	"fmt"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// ErrTry error
type ErrTry struct {
	Value interface{}
	Stack string
}

func (e *ErrTry) Error() string {
	return fmt.Sprintf("error value: %#v\n%s", e.Value, e.Stack)
}

// Is interface
func (e *ErrTry) Is(err error) bool { _, ok := err.(*ErrTry); return ok }

// Unwrap stdlib interface
func (e *ErrTry) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return fmt.Errorf("%v", e.Value)
}

// ErrExpectElement error
type ErrExpectElement struct {
	*proto.RuntimeRemoteObject
}

func (e *ErrExpectElement) Error() string {
	return fmt.Sprintf("expect js to return an element, but got: %s", utils.MustToJSON(e))
}

// Is interface
func (e *ErrExpectElement) Is(err error) bool { _, ok := err.(*ErrExpectElement); return ok }

// ErrExpectElements error
type ErrExpectElements struct {
	*proto.RuntimeRemoteObject
}

func (e *ErrExpectElements) Error() string {
	return fmt.Sprintf("expect js to return an array of elements, but got: %s", utils.MustToJSON(e))
}

// Is interface
func (e *ErrExpectElements) Is(err error) bool { _, ok := err.(*ErrExpectElements); return ok }

// ErrElementNotFound error
type ErrElementNotFound struct{}

func (e *ErrElementNotFound) Error() string {
	return "cannot find element"
}

// NotFoundSleeper returns ErrElementNotFound on the first call
func NotFoundSleeper() utils.Sleeper {
	return func(context.Context) error {
		return &ErrElementNotFound{}
	}
}

// ErrObjectNotFound error
type ErrObjectNotFound struct {
	*proto.RuntimeRemoteObject
}

func (e *ErrObjectNotFound) Error() string {
	return fmt.Sprintf("cannot find object: %s", utils.MustToJSON(e))
}

// Is interface
func (e *ErrObjectNotFound) Is(err error) bool { _, ok := err.(*ErrObjectNotFound); return ok }

// ErrEval error
type ErrEval struct {
	*proto.RuntimeExceptionDetails
}

func (e *ErrEval) Error() string {
	exp := e.Exception
	return fmt.Sprintf("eval js error: %s %s", exp.Description, exp.Value)
}

// Is interface
func (e *ErrEval) Is(err error) bool { _, ok := err.(*ErrEval); return ok }

// ErrNavigation error
type ErrNavigation struct {
	Reason string
}

func (e *ErrNavigation) Error() string {
	return "navigation failed: " + e.Reason
}

// Is interface
func (e *ErrNavigation) Is(err error) bool { _, ok := err.(*ErrNavigation); return ok }

// ErrPageCloseCanceled error
type ErrPageCloseCanceled struct{}

func (e *ErrPageCloseCanceled) Error() string {
	return "page close canceled"
}

// ErrNotInteractable error. Check the doc of Element.Interactable for details.
type ErrNotInteractable struct{}

func (e *ErrNotInteractable) Error() string {
	return "element is not cursor interactable"
}

// ErrInvisibleShape error.
type ErrInvisibleShape struct {
	*Element
}

// Error ...
func (e *ErrInvisibleShape) Error() string {
	return fmt.Sprintf("element has no visible shape or outside the viewport: %s", e.String())
}

// Is interface
func (e *ErrInvisibleShape) Is(err error) bool { _, ok := err.(*ErrInvisibleShape); return ok }

// Unwrap ...
func (e *ErrInvisibleShape) Unwrap() error {
	return &ErrNotInteractable{}
}

// ErrCovered error.
type ErrCovered struct {
	*Element
}

// Error ...
func (e *ErrCovered) Error() string {
	return fmt.Sprintf("element covered by: %s", e.String())
}

// Unwrap ...
func (e *ErrCovered) Unwrap() error {
	return &ErrNotInteractable{}
}

// Is interface
func (e *ErrCovered) Is(err error) bool { _, ok := err.(*ErrCovered); return ok }

// ErrNoPointerEvents error.
type ErrNoPointerEvents struct {
	*Element
}

// Error ...
func (e *ErrNoPointerEvents) Error() string {
	return fmt.Sprintf("element's pointer-events is none: %s", e.String())
}

// Unwrap ...
func (e *ErrNoPointerEvents) Unwrap() error {
	return &ErrNotInteractable{}
}

// Is interface
func (e *ErrNoPointerEvents) Is(err error) bool { _, ok := err.(*ErrNoPointerEvents); return ok }

// ErrPageNotFound error
type ErrPageNotFound struct{}

func (e *ErrPageNotFound) Error() string {
	return "cannot find page"
}

// ErrNoShadowRoot error
type ErrNoShadowRoot struct {
	*Element
}

// Error ...
func (e *ErrNoShadowRoot) Error() string {
	return fmt.Sprintf("element has no shadow root: %s", e.String())
}

// Is interface
func (e *ErrNoShadowRoot) Is(err error) bool { _, ok := err.(*ErrNoShadowRoot); return ok }
