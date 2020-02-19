package launcher_test

import (
	"errors"
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
	url := launcher.New().Delete("test").Bin("").
		Headless(false).Headless(true).RemoteDebuggingPort(0).
		Launch()
	url, err := launcher.GetWebSocketDebuggerURL(url)
	kit.E(err)
	assert.NotEmpty(t, url)
}

func TestDownloadErr(t *testing.T) {
	c := launcher.NewChrome()
	c.ErrInjector.CountInject(1, errors.New("err"))
	assert.Error(t, c.Download())

	c.ErrInjector.CountInject(2, errors.New("err"))
	assert.Error(t, c.Download())
}
