package devices

// Clear is used to clear overrides
var Clear = Device{clear: true}

// Test device
var Test = Device{
	UserAgent: "Test Agent",
	Screen: Screen{
		DevicePixelRatio: 1,
		Horizontal: ScreenSize{
			Width:  800,
			Height: 600,
		},
		Vertical: ScreenSize{
			Width:  800,
			Height: 600,
		},
	},
}

func has(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}
