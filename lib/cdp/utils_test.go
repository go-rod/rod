package cdp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils(t *testing.T) {
	err := errors.New("err")
	assert.False(t, isClosedErr(err))

	assert.Panics(t, func() {
		checkPanic(err)
	})

	debug("info")

	old := Debug
	Debug = true
	defer func() { Debug = old }()
	debug(nil)
}
