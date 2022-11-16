// Package devices ...
package devices

import (
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

// Device represents a emulated device.
type Device struct {
	Capabilities   []string
	UserAgent      string
	AcceptLanguage string
	Screen         Screen
	Title          string

	landscape bool
	clear     bool
}

// Screen represents the screen of a device.
type Screen struct {
	DevicePixelRatio float64
	Horizontal       ScreenSize
	Vertical         ScreenSize
}

// ScreenSize represents the size of the screen.
type ScreenSize struct {
	Width  int
	Height int
}

// Landscape clones the device and set it to landscape mode
func (device Device) Landscape() Device {
	d := device
	d.landscape = true
	return d
}

// MetricsEmulation config
func (device Device) MetricsEmulation() *proto.EmulationSetDeviceMetricsOverride {
	if device.IsClear() {
		return nil
	}

	var screen ScreenSize
	var orientation *proto.EmulationScreenOrientation
	if device.landscape {
		screen = device.Screen.Horizontal
		orientation = &proto.EmulationScreenOrientation{
			Angle: 90,
			Type:  proto.EmulationScreenOrientationTypeLandscapePrimary,
		}
	} else {
		screen = device.Screen.Vertical
		orientation = &proto.EmulationScreenOrientation{
			Angle: 0,
			Type:  proto.EmulationScreenOrientationTypePortraitPrimary,
		}
	}

	return &proto.EmulationSetDeviceMetricsOverride{
		Width:             screen.Width,
		Height:            screen.Height,
		DeviceScaleFactor: device.Screen.DevicePixelRatio,
		ScreenOrientation: orientation,
		Mobile:            has(device.Capabilities, "mobile"),
	}
}

// TouchEmulation config
func (device Device) TouchEmulation() *proto.EmulationSetTouchEmulationEnabled {
	if device.IsClear() {
		return &proto.EmulationSetTouchEmulationEnabled{
			Enabled: false,
		}
	}

	return &proto.EmulationSetTouchEmulationEnabled{
		Enabled:        has(device.Capabilities, "touch"),
		MaxTouchPoints: gson.Int(5),
	}
}

// UserAgentEmulation config
func (device Device) UserAgentEmulation() *proto.NetworkSetUserAgentOverride {
	if device.IsClear() {
		return nil
	}

	return &proto.NetworkSetUserAgentOverride{
		UserAgent:      device.UserAgent,
		AcceptLanguage: device.AcceptLanguage,
	}
}

// IsClear type
func (device Device) IsClear() bool {
	return device.clear
}
