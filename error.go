package rod

import (
	errors "github.com/pkg/errors"
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
	// ErrNotClickable error
	ErrNotClickable = errors.New("element is not clickable")
)

// Error type for rod
type Error struct {
	Code    error
	Details interface{}
}

func newErr(code error, details interface{}, msg string) error {
	return errors.WithStack(errors.WithMessage(&Error{code, details}, msg))
}

// Error interface
func (e *Error) Error() string {
	return e.Code.Error()
}

// Unwrap interface
func (e *Error) Unwrap() error {
	return e.Code
}
