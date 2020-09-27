package defaults

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	Show = true
	Devtools = true
	URL = "test"
	Monitor = "test"

	ResetWithEnv()
	parse("")
	assert.False(t, Show)
	assert.False(t, Devtools)
	assert.Equal(t, "", Monitor)
	assert.Equal(t, "", URL)

	parse("show,devtools,trace,slow=2s,port=8080,dir=tmp," +
		"url=http://test.com,cdp,monitor,bin=/path/to/chrome," +
		"proxy=localhost:8080",
	)

	assert.True(t, Show)
	assert.True(t, Devtools)
	assert.True(t, Trace)
	assert.Equal(t, 2*time.Second, Slow)
	assert.Equal(t, "8080", Port)
	assert.Equal(t, "/path/to/chrome", Bin)
	assert.Equal(t, "tmp", Dir)
	assert.Equal(t, "http://test.com", URL)
	assert.True(t, CDP)
	assert.Equal(t, ":0", Monitor)
	assert.Equal(t, "localhost:8080", Proxy)

	parse("monitor=:1234")
	assert.Equal(t, ":1234", Monitor)

	assert.Panics(t, func() {
		parse("a")
	})
}
