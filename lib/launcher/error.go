package launcher

import "errors"

// ErrAlreadyLaunched is an error that indicates the launcher has already been launched.
var ErrAlreadyLaunched = errors.New("already launched")
