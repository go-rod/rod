package rod

import "fmt"

// ErrCode for errors
type ErrCode string

const (
	// ErrExpectElement error code
	ErrExpectElement ErrCode = "expect js to return an element"
	// ErrExpectElements error code
	ErrExpectElements ErrCode = "expect js to return an array of elements"
	// ErrElementNotFound error code
	ErrElementNotFound ErrCode = "cannot find element"
	// ErrWaitJSTimeout error code
	ErrWaitJSTimeout ErrCode = "wait js timeout"
	// ErrSrcNotFound error code
	ErrSrcNotFound ErrCode = "element doesn't have src attribute"
	// ErrEval error code
	ErrEval ErrCode = "eval error"
	// ErrNavigation error code
	ErrNavigation ErrCode = "navigation failed"
)

// Error ...
type Error struct {
	Err     error
	Code    ErrCode
	Details interface{}
}

// Error ...
func (e *Error) Error() string {
	return fmt.Sprintf("[rod] %s\n%v", e.Code, e.Details)
}

// Unwrap ...
func (e *Error) Unwrap() error {
	return e.Err
}

// IsError type matches
func IsError(err error, code ErrCode) bool {
	if err == nil {
		return false
	}

	e, ok := err.(*Error)
	if !ok {
		return false
	}

	return e.Code == code
}
