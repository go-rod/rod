// Package defaults holds some commonly used options parsed from env var "rod".
// Set them will set the default value of options used by rod.
// Each value is separated by a ",", key and value are separated by "=",
// For example:
//
//    rod=show,trace,slow,monitor
//
//    rod=show,trace,slow=1s,port=9222,monitor=:9223
//
package defaults

import (
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

// Show disables headless mode
var Show bool

// Trace enables tracing
var Trace bool

// Quiet is only useful when Trace is enabled. It decides whether to log tracing message or not.
var Quiet bool

// Slow enables slowmotion mode if not zero
var Slow time.Duration

// Dir to store browser profile, such as cookies
var Dir string

// Port of the remote debugging port
var Port string

// Bin path of chrome executable file
var Bin string

// URL of the remote debugging address
var URL string

// Remote enables to launch browser remotely
var Remote bool

// CDP enables cdp log
var CDP bool

// Monitor enables the monitor server that plays the screenshots of each tab, default value is 0.0.0.0:9273
var Monitor string

// Blind is only useful when Monitor is enabled, it decides whether to open a browser to watch the screenshots or not
var Blind bool

// Parse the flags
func init() {
	Reset()
	parse(os.Getenv("rod"))
}

// Reset all flags
func Reset() {
	CDP = false
	Trace = false
	URL = ""
	Dir = ""
	Bin = ""
	Port = "0"
	Blind = false
	Monitor = ""
	Quiet = false
	Show = false
	Slow = 0
}

// parse options and set them globally
func parse(options string) {
	if options == "" {
		return
	}

	for _, f := range strings.Split(options, ",") {
		set(f)
	}
}

func set(f string) {
	kv := strings.Split(f, "=")
	switch kv[0] {
	case "show":
		Show = true
	case "trace":
		Trace = true
	case "quiet":
		Quiet = true
	case "slow":
		var err error
		Slow, err = time.ParseDuration(kv[1])
		utils.E(err)
	case "bin":
		Bin = kv[1]
	case "dir":
		Dir = kv[1]
	case "port":
		Port = kv[1]
	case "url":
		URL = kv[1]
	case "remote":
		Remote = true
	case "cdp":
		CDP = true
	case "monitor":
		Monitor = ":9273"
		if len(kv) == 2 {
			Monitor = kv[1]
		}
	case "blind":
		Blind = true
	default:
		panic("no such rod option: " + kv[0])
	}
}
