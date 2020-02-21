package launcher_test

import (
	"context"
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
	ctx := context.Background()
	url := launcher.New().Context(ctx).Delete("test").Bin("").
		Headless(false).Headless(true).RemoteDebuggingPort(0).
		Launch()
	url, err := launcher.GetWebSocketDebuggerURL(ctx, url)
	kit.E(err)
	assert.NotEmpty(t, url)
}
