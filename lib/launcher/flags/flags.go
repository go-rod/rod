// Package flags ...
package flags

import "strings"

// Flag name of a command line argument of the browser, also known as command line flag or switch.
// List of available flags: https://peter.sh/experiments/chromium-command-line-switches
type Flag string

// TODO: we should automatically generate all the flags here.
const (
	// UserDataDir https://chromium.googlesource.com/chromium/src/+/master/docs/user_data_dir.md
	UserDataDir Flag = "user-data-dir"

	// Headless mode. Whether to run browser in headless mode. A mode without visible UI.
	Headless Flag = "headless"

	// App flag.
	App Flag = "app"

	// RemoteDebuggingPort flag.
	RemoteDebuggingPort Flag = "remote-debugging-port"

	// NoSandbox flag.
	NoSandbox Flag = "no-sandbox"

	// ProxyServer flag.
	ProxyServer Flag = "proxy-server"

	// WorkingDir flag.
	WorkingDir Flag = "rod-working-dir"

	// Env flag.
	Env Flag = "rod-env"

	// XVFB flag.
	XVFB Flag = "rod-xvfb"

	// ProfileDir flag.
	ProfileDir = "profile-directory"

	// Preferences flag.
	Preferences Flag = "rod-preferences"

	// Leakless flag.
	Leakless Flag = "rod-leakless"

	// Bin is the browser executable file path. If it's empty, launcher will automatically search or download the bin.
	Bin Flag = "rod-bin"

	// KeepUserDataDir flag.
	KeepUserDataDir Flag = "rod-keep-user-data-dir"

	// Arguments for the command. Such as
	//     chrome-bin http://a.com http://b.com
	// The "http://a.com" and "http://b.com" are the arguments.
	Arguments Flag = ""
)

// Check if the flag name is valid.
func (f Flag) Check() {
	if strings.Contains(string(f), "=") {
		panic("flag name should not contain '='")
	}
}

// NormalizeFlag normalize the flag name, remove the leading dash.
func (f Flag) NormalizeFlag() Flag {
	return Flag(strings.TrimLeft(string(f), "-"))
}
