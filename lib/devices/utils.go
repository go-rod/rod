package devices

import (
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

// Device for devices
type Device struct {
	gson.JSON
}

// Clear is used to clear overrides
var Clear = Device{}

// Metrics config
func (device Device) Metrics(landscape bool) *proto.EmulationSetDeviceMetricsOverride {
	if device == Clear {
		return nil
	}

	var screen gson.JSON
	var orientation *proto.EmulationScreenOrientation
	if landscape {
		screen = device.Get("screen.horizontal")
		orientation = &proto.EmulationScreenOrientation{
			Angle: 90,
			Type:  proto.EmulationScreenOrientationTypeLandscapePrimary,
		}
	} else {
		screen = device.Get("screen.vertical")
		orientation = &proto.EmulationScreenOrientation{
			Angle: 0,
			Type:  proto.EmulationScreenOrientationTypePortraitPrimary,
		}
	}

	return &proto.EmulationSetDeviceMetricsOverride{
		Width:             screen.Get("width").Int(),
		Height:            screen.Get("height").Int(),
		DeviceScaleFactor: device.Get("screen.device-pixel-ratio").Num(),
		ScreenOrientation: orientation,
		Mobile:            has(device.Get("capabilities"), "mobile"),
	}
}

// Touch config
func (device Device) Touch() *proto.EmulationSetTouchEmulationEnabled {
	if device == Clear {
		return &proto.EmulationSetTouchEmulationEnabled{
			Enabled: false,
		}
	}

	return &proto.EmulationSetTouchEmulationEnabled{
		Enabled:        has(device.Get("capabilities"), "touch"),
		MaxTouchPoints: 5,
	}
}

// UserAgent config
func (device Device) UserAgent() *proto.NetworkSetUserAgentOverride {
	if device == Clear {
		return nil
	}

	return &proto.NetworkSetUserAgentOverride{
		UserAgent: device.Get("user-agent").String(),
	}
}

func has(arr gson.JSON, str string) bool {
	for _, item := range arr.Arr() {
		if item.Str() == str {
			return true
		}
	}
	return false
}
