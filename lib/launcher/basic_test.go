package launcher_test

import (
	"context"
	"os"
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
	portFlag, _ := launcher.New().Get("remote-debugging-port")
	assert.Equal(t, "0", portFlag[0])

	ctx := context.Background()
	url := launcher.NewUserMode().Context(ctx).Delete("test").Bin("").
		Log(func(s string) { kit.E(os.Stdout.WriteString(s)) }).
		Leakless(true).
		Headless(false).Headless(true).RemoteDebuggingPort(0).
		Launch()
	url, err := launcher.GetWebSocketDebuggerURL(ctx, url)
	kit.E(err)
	assert.NotEmpty(t, url)
}
