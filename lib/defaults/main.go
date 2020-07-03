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

	"github.com/ysmood/kit"
)

// Show disables headless mode
var Show bool

// Trace enables tracing
var Trace bool

// Slow enables slowmotion mode if not zero
var Slow time.Duration

// Dir to store browser profile, such as cookies
var Dir string

// Port of the remote debugging port
var Port = "0"

// URL of the remote debugging address
var URL = ""

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
	parse(os.Getenv("rod"))
}

// parse options and set them globally
func parse(options string) {
	if options == "" {
		return
	}

	for _, f := range strings.Split(options, ",") {
		kv := strings.Split(f, "=")
		switch kv[0] {
		case "show":
			Show = true
		case "trace":
			Trace = true
		case "slow":
			var err error
			Slow, err = time.ParseDuration(kv[1])
			kit.E(err)
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

}
