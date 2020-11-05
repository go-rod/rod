package rod

import (
	"fmt"
	"reflect"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// ErrTry error
type ErrTry struct {
	Value interface{}
}

func (e *ErrTry) Error() string {
	return fmt.Sprintf("error value: %#v", e.Value)
}

// Is interface
func (e *ErrTry) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

// ErrExpectElement error
type ErrExpectElement struct {
	*proto.RuntimeRemoteObject
}

func (e *ErrExpectElement) Error() string {
	return fmt.Sprintf("expect js to return an element, but got: %s", utils.MustToJSON(e))
}

// Is interface
func (e *ErrExpectElement) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

// ErrExpectElements error
type ErrExpectElements struct {
	*proto.RuntimeRemoteObject
}

func (e *ErrExpectElements) Error() string {
	return fmt.Sprintf("expect js to return an array of elements, but got: %s", utils.MustToJSON(e))
}

// Is interface
func (e *ErrExpectElements) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

// ErrElementNotFound error
type ErrElementNotFound struct {
}

func (e *ErrElementNotFound) Error() string {
	return "cannot find element"
}

// ErrObjectNotFound error
type ErrObjectNotFound struct {
	*proto.RuntimeRemoteObject
}

func (e *ErrObjectNotFound) Error() string {
	return fmt.Sprintf("cannot find object: %s", utils.MustToJSON(e))
}

// Is interface
func (e *ErrObjectNotFound) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

// ErrEval error
type ErrEval struct {
	*proto.RuntimeExceptionDetails
}

func (e *ErrEval) Error() string {
	exp := e.Exception
	return fmt.Sprintf("eval js error: %s %s", exp.Description, exp.Value)
}

// Is interface
func (e *ErrEval) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

// ErrNavigation error
type ErrNavigation struct {
	Reason string
}

func (e *ErrNavigation) Error() string {
	return "navigation failed: " + e.Reason
}

// Is interface
func (e *ErrNavigation) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

// ErrPageCloseCanceled error
type ErrPageCloseCanceled struct {
}

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
}

func (e *ErrInvisibleShape) Error() string {
	return "element has no visible shape"
}

func (e *ErrInvisibleShape) Unwrap() error {
	return &ErrNotInteractable{}
}

// ErrCovered error.
type ErrCovered struct {
	*Element
}

func (e *ErrCovered) Error() string {
	return fmt.Sprintf("element covered by: %v", e.MustHTML())
}

func (e *ErrCovered) Unwrap() error {
	return &ErrNotInteractable{}
}

// Is interface
func (e *ErrCovered) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}
