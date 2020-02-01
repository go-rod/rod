package fetcher_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/fetcher"
)

func TestGet(t *testing.T) {
	c := new(fetcher.Chrome)
	p, err := c.Get()
	kit.E(err)
	assert.True(t, kit.FileExists(p), p)
}
