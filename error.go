package rod

import (
	"github.com/pkg/errors"
)

var (
	// ErrValue error
	ErrValue = errors.New("error value")

	// ErrExpectElement error
	ErrExpectElement = errors.New("expect js to return an element")

	// ErrExpectElements error
	ErrExpectElements = errors.New("expect js to return an array of elements")

	// ErrElementNotFound error
	ErrElementNotFound = errors.New("cannot find element")

	// ErrSrcNotFound error
	ErrSrcNotFound = errors.New("element doesn't have src attribute")

	// ErrEval error
	ErrEval = errors.New("eval error")

	// ErrNavigation error
	ErrNavigation = errors.New("navigation failed")

	// ErrPageCloseCanceled error
	ErrPageCloseCanceled = errors.New("page close canceled")

	// ErrNotInteractable error. Check the doc of Element.Interactable for details.
	ErrNotInteractable = errors.New("element is not cursor interactable")
)

// Error type for rod
type Error struct {
	// Code is used to tell error types
	Code error

	// Details is a JSON object
	Details interface{}
}

func newErr(code error, details interface{}, msg string) error {
	return errors.WithStack(errors.WithMessage(&Error{code, details}, msg))
}

// AsError of *rod.Error
func AsError(err error) (e *Error) {
	errors.As(err, &e)
	return
}

// Error interface
func (e *Error) Error() string {
	return e.Code.Error()
}

// Unwrap interface
func (e *Error) Unwrap() error {
	return e.Code
}
