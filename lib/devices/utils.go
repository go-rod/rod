package devices

// Clear is used to clear overrides
var Clear = Device{clear: true}

func has(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}
