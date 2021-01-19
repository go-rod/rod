package cdp

import (
	"fmt"
)

// Error of the Response
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// Error stdlib interface
func (e *Error) Error() string {
	return fmt.Sprintf("%v", *e)
}

// Is stdlib interface
func (e Error) Is(target error) bool {
	err, ok := target.(*Error)
	return ok && e == *err
}

// ErrCtxNotFound type
var ErrCtxNotFound = &Error{
	Code:    -32000,
	Message: "Cannot find context with specified id",
}

// ErrCtxDestroyed type
var ErrCtxDestroyed = &Error{
	Code:    -32000,
	Message: "Execution context was destroyed.",
}

// ErrObjNotFound type
var ErrObjNotFound = &Error{
	Code:    -32000,
	Message: "Could not find object with given id",
}

// ErrNodeNotFoundAtPos type
var ErrNodeNotFoundAtPos = &Error{
	Code:    -32000,
	Message: "No node found at given location",
}

// ErrNoContentQuads type
var ErrNoContentQuads = &Error{
	Code:    -32000,
	Message: "Could not compute content quads.",
}

// ErrConnClosed type
var ErrConnClosed = &errConnClosed{}

type errConnClosed struct {
	details error
}

// Error stdlib interface
func (e *errConnClosed) Error() string {
	return fmt.Sprintf("cdp connection closed: %v", e.details)
}

// Is stdlib interface
func (e errConnClosed) Is(target error) bool {
	_, ok := target.(*errConnClosed)
	return ok
}

// Unwrap stdlib interface
func (e errConnClosed) Unwrap() error {
	return e.details
}
