package defaults

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	Show = true
	URL = "test"
	Monitor = "test"
	Remote = true

	ResetWithEnv()
	parse("")
	assert.False(t, Show)
	assert.Equal(t, "", Monitor)
	assert.Equal(t, "", URL)
	assert.False(t, Remote)

	parse("show,trace,slow=2s,port=8080,remote,dir=tmp," +
		"url=http://test.com,cdp,monitor,blind,quiet,bin=/path/to/chrome," +
		"proxy=localhost:8080",
	)

	assert.True(t, Show)
	assert.True(t, Trace)
	assert.True(t, Quiet)
	assert.Equal(t, 2*time.Second, Slow)
	assert.Equal(t, "8080", Port)
	assert.Equal(t, true, Remote)
	assert.Equal(t, "/path/to/chrome", Bin)
	assert.Equal(t, "tmp", Dir)
	assert.Equal(t, "http://test.com", URL)
	assert.True(t, CDP)
	assert.Equal(t, ":9273", Monitor)
	assert.Equal(t, true, Blind)
	assert.Equal(t, "localhost:8080", Proxy)

	parse("monitor=:1234")
	assert.Equal(t, ":1234", Monitor)

	assert.Panics(t, func() {
		parse("a")
	})
}
