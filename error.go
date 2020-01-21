package rod

// Error ...
type Error struct {
	err error
	msg string
}

// Error ...
func (e *Error) Error() string {
	return "[rod] " + e.msg
}

// Unwrap ...
func (e *Error) Unwrap() error {
	return e.err
}

func newErr(parent error, msg string) error {
	return &Error{parent, msg}
}
