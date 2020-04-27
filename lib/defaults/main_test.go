package defaults

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	parse("")
	assert.Equal(t, "", Monitor)

	parse("show,trace,slow=2s,port=8080,cdp,monitor")

	assert.True(t, Show)
	assert.True(t, Trace)
	assert.Equal(t, 2*time.Second, Slow)
	assert.Equal(t, "8080", Port)
	assert.True(t, CDP)
	assert.Equal(t, ":9273", Monitor)

	parse("monitor=:1234")
	assert.Equal(t, ":1234", Monitor)

	assert.Panics(t, func() {
		parse("a")
	})
}
