package devices_test

import (
	"testing"

	"github.com/go-rod/rod/lib/devices"
	"github.com/stretchr/testify/assert"
)

func TestErr(t *testing.T) {
	v := devices.IPad.Metrics(false)
	touch := devices.IPad.Touch()
	assert.EqualValues(t, 768, v.Width)
	assert.EqualValues(t, 1024, v.Height)
	assert.EqualValues(t, 2, v.DeviceScaleFactor)
	assert.EqualValues(t, 0, v.ScreenOrientation.Angle)
	assert.True(t, v.Mobile)
	assert.True(t, touch.Enabled)

	v = devices.LaptopWithMDPIScreen.Metrics(true)
	touch = devices.LaptopWithMDPIScreen.Touch()
	assert.EqualValues(t, 1280, v.Width)
	assert.EqualValues(t, 90, v.ScreenOrientation.Angle)
	assert.False(t, v.Mobile)
	assert.False(t, touch.Enabled)

	u := devices.IPad.UserAgent()
	assert.Equal(t, "Mozilla/5.0 (iPad; CPU OS 11_0 like Mac OS X) AppleWebKit/604.1.34 (KHTML, like Gecko) Version/11.0 Mobile/15A5341f Safari/604.1", u.UserAgent)

	assert.Nil(t, devices.Clear.Metrics(true))
	assert.False(t, devices.Clear.Touch().Enabled)
	assert.Nil(t, devices.Clear.UserAgent())
}
