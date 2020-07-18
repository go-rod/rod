package rod

import (
	"errors"
)

var (
	// ErrValue error
	ErrValue = errors.New("[rod] error value")
	// ErrExpectElement error
	ErrExpectElement = errors.New("[rod] expect js to return an element")
	// ErrExpectElements error
	ErrExpectElements = errors.New("[rod] expect js to return an array of elements")
	// ErrElementNotFound error
	ErrElementNotFound = errors.New("[rod] cannot find element")
	// ErrWaitJSTimeout error
	ErrWaitJSTimeout = errors.New("[rod] wait js timeout")
	// ErrSrcNotFound error
	ErrSrcNotFound = errors.New("[rod] element doesn't have src attribute")
	// ErrEval error
	ErrEval = errors.New("[rod] eval error")
	// ErrNavigation error
	ErrNavigation = errors.New("[rod] navigation failed")
)

// Error ...
type Error struct {
	Err     error
	Details interface{}
}

func newErr(e error, details interface{}) *Error {
	return &Error{e, details}
}

// Error ...
func (e *Error) Error() string {
	return e.Err.Error()
}

// Unwrap ...
func (e *Error) Unwrap() error {
	return e.Err
}
