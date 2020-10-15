package cdp

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
