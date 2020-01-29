package rod

// Error ...
type Error struct {
	err     error
	msg     string
	details interface{}
}

// Error ...
func (e *Error) Error() string {
	return "[rod] " + e.msg
}

// Unwrap ...
func (e *Error) Unwrap() error {
	return e.err
}

func newErr(parent error, msg string, details interface{}) error {
	return &Error{parent, msg, details}
}
