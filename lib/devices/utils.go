package devices

import (
	"errors"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/proto"
	"github.com/tidwall/gjson"
)

// DeviceType for devices
type DeviceType string

// ErrDeviceNotExists err
var ErrDeviceNotExists = errors.New("device not exists")

var list = gjson.Parse(assets.DeviceList).Array()

// Get the config of the device
func Get(device DeviceType, landscape bool) (*proto.EmulationSetDeviceMetricsOverride, *proto.EmulationSetTouchEmulationEnabled) {
	if device == "" {
		return nil, nil
	}

	d := find(device)

	var screen gjson.Result
	var orientation *proto.EmulationScreenOrientation
	if landscape {
		screen = d.Get("screen.horizontal")
		orientation = &proto.EmulationScreenOrientation{
			Angle: 90,
			Type:  proto.EmulationScreenOrientationTypeLandscapePrimary,
		}
	} else {
		screen = d.Get("screen.vertical")
		orientation = &proto.EmulationScreenOrientation{
			Angle: 0,
			Type:  proto.EmulationScreenOrientationTypePortraitPrimary,
		}
	}

	return &proto.EmulationSetDeviceMetricsOverride{
			Width:             screen.Get("width").Int(),
			Height:            screen.Get("height").Int(),
			DeviceScaleFactor: d.Get("screen.device-pixel-ratio").Float(),
			ScreenOrientation: orientation,
			Mobile:            has(d.Get("capabilities"), "mobile"),
		}, &proto.EmulationSetTouchEmulationEnabled{
			Enabled:        has(d.Get("capabilities"), "touch"),
			MaxTouchPoints: 5,
		}
}

// GetUserAgent of the device
func GetUserAgent(device DeviceType) *proto.NetworkSetUserAgentOverride {
	if device == "" {
		return nil
	}

	return &proto.NetworkSetUserAgentOverride{
		UserAgent: find(device).Get("user-agent").String(),
	}
}

func find(name DeviceType) gjson.Result {
	for _, d := range list {
		if d.Get("device.title").String() == string(name) {
			return d.Get("device")
		}
	}
	panic(ErrDeviceNotExists)
}

func has(arr gjson.Result, str string) bool {
	for _, item := range arr.Array() {
		if item.Str == str {
			return true
		}
	}
	return false
}
