package rod

import (
	"errors"
	"fmt"
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
	// ErrWaitJSTimeout error
	ErrWaitJSTimeout = errors.New("wait js timeout")
	// ErrSrcNotFound error
	ErrSrcNotFound = errors.New("element doesn't have src attribute")
	// ErrEval error
	ErrEval = errors.New("eval error")
	// ErrNavigation error
	ErrNavigation = errors.New("navigation failed")
)

// Error ...
type Error struct {
	Err     error
	Details interface{}
}

// Error ...
func (e *Error) Error() string {
	return fmt.Sprintf("[rod] %v\n%v", e.Err, e.Details)
}

// Unwrap ...
func (e *Error) Unwrap() error {
	return e.Err
}
