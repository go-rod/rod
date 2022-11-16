package devices_test

import (
	"testing"

	"github.com/go-rod/rod/lib/devices"
	"github.com/ysmood/got"
)

func TestErr(t *testing.T) {
	as := got.New(t)

	v := devices.IPad.MetricsEmulation()
	touch := devices.IPad.TouchEmulation()
	as.Eq(768, v.Width)
	as.Eq(1024, v.Height)
	as.Eq(2, v.DeviceScaleFactor)
	as.Eq(0, v.ScreenOrientation.Angle)
	as.True(v.Mobile)
	as.True(touch.Enabled)

	v = devices.LaptopWithMDPIScreen.Landscape().MetricsEmulation()
	touch = devices.LaptopWithMDPIScreen.TouchEmulation()
	as.Eq(1280, v.Width)
	as.Eq(90, v.ScreenOrientation.Angle)
	as.False(v.Mobile)
	as.False(touch.Enabled)

	u := devices.IPad.UserAgentEmulation()
	as.Eq("Mozilla/5.0 (iPad; CPU OS 11_0 like Mac OS X) AppleWebKit/604.1.34 (KHTML, like Gecko) Version/11.0 Mobile/15A5341f Safari/604.1", u.UserAgent)

	as.Nil(devices.Clear.MetricsEmulation())
	as.False(devices.Clear.TouchEmulation().Enabled)
	as.Nil(devices.Clear.UserAgentEmulation())
}
