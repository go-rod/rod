package rod

import "fmt"

const (
	// ErrExpectElement error code
	ErrExpectElement = "expect js to return an element"
	// ErrExpectElements error code
	ErrExpectElements = "expect js to return an array of elements"
	// ErrElementNotFound error code
	ErrElementNotFound = "cannot find element"
)

// Error ...
type Error struct {
	Err     error
	Code    string
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
func IsError(err error, code string) bool {
	if err == nil {
		return false
	}

	e, ok := err.(*Error)
	if !ok {
		return false
	}

	return e.Code == code
}
