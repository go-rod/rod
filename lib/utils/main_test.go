package utils_test

import (
	"errors"
	"testing"

	"github.com/go-rod/rod/lib/utils"
	"github.com/stretchr/testify/assert"
)

func TestErr(t *testing.T) {
	utils.E(nil)

	assert.Panics(t, func() {
		utils.E(errors.New("err"))
	})
}
