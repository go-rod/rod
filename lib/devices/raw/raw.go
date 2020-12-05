package raw

// Device represents a emulated device.
type Device struct {
	Capabilities []string `json:"capabilities"`
	UserAgent    string   `json:"user-agent"`
	Screen       Screen   `json:"screen"`
	Title        string   `json:"title"`
}

// Screen represents the screen of a device.
type Screen struct {
	DevicePixelRatio float64    `json:"device-pixel-ratio"`
	Horizontal       ScreenSize `json:"horizontal"`
	Vertical         ScreenSize `json:"vertical"`
}

// ScreenSize represents the size of the screen.
type ScreenSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
