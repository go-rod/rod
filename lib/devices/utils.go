package devices

import (
	"github.com/go-rod/rod/lib/devices/raw"
	"github.com/go-rod/rod/lib/proto"
)

// Device for devices
type Device struct {
	raw       *raw.Device
	landscape bool
}

// Clear is used to clear overrides
var Clear = Device{}

// Test device
var Test = Device{
	raw: &raw.Device{
		UserAgent: "Test Agent",
		Screen: raw.Screen{
			DevicePixelRatio: 1,
			Horizontal: raw.ScreenSize{
				Width:  800,
				Height: 600,
			},
			Vertical: raw.ScreenSize{
				Width:  800,
				Height: 600,
			},
		},
	},
}

// Landescape clones the device and set it to landscape mode
func (device Device) Landescape() Device {
	d := device
	d.landscape = true
	return d
}

// Metrics config
func (device Device) Metrics() *proto.EmulationSetDeviceMetricsOverride {
	if device == Clear {
		return nil
	}

	var screen raw.ScreenSize
	var orientation *proto.EmulationScreenOrientation
	if device.landscape {
		screen = device.raw.Screen.Horizontal
		orientation = &proto.EmulationScreenOrientation{
			Angle: 90,
			Type:  proto.EmulationScreenOrientationTypeLandscapePrimary,
		}
	} else {
		screen = device.raw.Screen.Vertical
		orientation = &proto.EmulationScreenOrientation{
			Angle: 0,
			Type:  proto.EmulationScreenOrientationTypePortraitPrimary,
		}
	}

	return &proto.EmulationSetDeviceMetricsOverride{
		Width:             screen.Width,
		Height:            screen.Height,
		DeviceScaleFactor: device.raw.Screen.DevicePixelRatio,
		ScreenOrientation: orientation,
		Mobile:            has(device.raw.Capabilities, "mobile"),
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
		Enabled:        has(device.raw.Capabilities, "touch"),
		MaxTouchPoints: 5,
	}
}

// UserAgent config
func (device Device) UserAgent() *proto.NetworkSetUserAgentOverride {
	if device == Clear {
		return nil
	}

	return &proto.NetworkSetUserAgentOverride{
		UserAgent: device.raw.UserAgent,
	}
}

func has(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}
