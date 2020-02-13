package launcher_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/launcher"
)

func TestDownload(t *testing.T) {
	c := launcher.NewChrome()
	kit.E(c.Download())
	assert.FileExists(t, c.ExecPath())
}

func TestLaunch(t *testing.T) {
	url := launcher.Launch("", "", nil)
	url, err := launcher.GetWebSocketDebuggerURL(url)
	kit.E(err)
	assert.NotEmpty(t, url)
}
