package devices_test

import (
	"testing"

	"github.com/go-rod/rod/lib/devices"
	"github.com/stretchr/testify/assert"
)

func TestErr(t *testing.T) {
	v, touch := devices.Get(devices.IPad, false)
	assert.EqualValues(t, 768, v.Width)
	assert.EqualValues(t, 1024, v.Height)
	assert.EqualValues(t, 2, v.DeviceScaleFactor)
	assert.EqualValues(t, 0, v.ScreenOrientation.Angle)
	assert.True(t, v.Mobile)
	assert.True(t, touch.Enabled)

	v, touch = devices.Get(devices.LaptopWithMDPIScreen, true)
	assert.EqualValues(t, 1280, v.Width)
	assert.EqualValues(t, 90, v.ScreenOrientation.Angle)
	assert.False(t, v.Mobile)
	assert.False(t, touch.Enabled)

	v, _ = devices.Get("", true)
	assert.Nil(t, v)

	u := devices.GetUserAgent(devices.IPad)
	assert.Equal(t, "Mozilla/5.0 (iPad; CPU OS 11_0 like Mac OS X) AppleWebKit/604.1.34 (KHTML, like Gecko) Version/11.0 Mobile/15A5341f Safari/604.1", u.UserAgent)

	assert.Nil(t, devices.GetUserAgent(""))

	assert.Panics(t, func() {
		devices.GetUserAgent("xxx")
	})
}
