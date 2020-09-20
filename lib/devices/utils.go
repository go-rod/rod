package devices

import (
	"github.com/go-rod/rod/lib/proto"
	"github.com/tidwall/gjson"
)

// Device for devices
type Device struct {
	json gjson.Result
}

// Clear is used to clear overrides
var Clear = Device{}

// Metrics config
func (device Device) Metrics(landscape bool) *proto.EmulationSetDeviceMetricsOverride {
	if device == Clear {
		return nil
	}

	var screen gjson.Result
	var orientation *proto.EmulationScreenOrientation
	if landscape {
		screen = device.json.Get("screen.horizontal")
		orientation = &proto.EmulationScreenOrientation{
			Angle: 90,
			Type:  proto.EmulationScreenOrientationTypeLandscapePrimary,
		}
	} else {
		screen = device.json.Get("screen.vertical")
		orientation = &proto.EmulationScreenOrientation{
			Angle: 0,
			Type:  proto.EmulationScreenOrientationTypePortraitPrimary,
		}
	}

	return &proto.EmulationSetDeviceMetricsOverride{
		Width:             screen.Get("width").Int(),
		Height:            screen.Get("height").Int(),
		DeviceScaleFactor: device.json.Get("screen.device-pixel-ratio").Float(),
		ScreenOrientation: orientation,
		Mobile:            has(device.json.Get("capabilities"), "mobile"),
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
		Enabled:        has(device.json.Get("capabilities"), "touch"),
		MaxTouchPoints: 5,
	}
}

// UserAgent config
func (device Device) UserAgent() *proto.NetworkSetUserAgentOverride {
	if device == Clear {
		return nil
	}

	return &proto.NetworkSetUserAgentOverride{
		UserAgent: device.json.Get("user-agent").String(),
	}
}

func has(arr gjson.Result, str string) bool {
	for _, item := range arr.Array() {
		if item.Str == str {
			return true
		}
	}
	return false
}
