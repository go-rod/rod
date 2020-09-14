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

// Proxy for the browser
var Proxy string

// Parse the flags
func init() {
	ResetWithEnv()
}

// Reset all flags to their init values.
func Reset() {
	Show = false
	Trace = false
	Quiet = false
	Slow = 0
	Dir = ""
	Port = "0"
	Bin = ""
	URL = ""
	Remote = false
	CDP = false
	Monitor = ""
	Blind = false
	Proxy = ""
}

// ResetWithEnv all flags by the value of the rod env var.
func ResetWithEnv() {
	Reset()
	parse(os.Getenv("rod"))
}

// parse options and set them globally
func parse(options string) {
	if options == "" {
		return
	}

	for _, f := range strings.Split(options, ",") {
		kv := strings.Split(f, "=")
		if len(kv) == 2 {
			rules[kv[0]](kv[1])
		} else {
			rules[kv[0]]("")
		}
	}
}

var rules = map[string]func(string){
	"show": func(string) {
		Show = true
	},
	"trace": func(string) {
		Trace = true
	},
	"quiet": func(string) {
		Quiet = true
	},
	"slow": func(v string) {
		var err error
		Slow, err = time.ParseDuration(v)
		utils.E(err)
	},
	"bin": func(v string) {
		Bin = v
	},
	"dir": func(v string) {
		Dir = v
	},
	"port": func(v string) {
		Port = v
	},
	"url": func(v string) {
		URL = v
	},
	"remote": func(v string) {
		Remote = true
	},
	"cdp": func(v string) {
		CDP = true
	},
	"monitor": func(v string) {
		Monitor = ":9273"
		if v != "" {
			Monitor = v
		}
	},
	"blind": func(v string) {
		Blind = true
	},
	"proxy": func(v string) {
		Proxy = v
	},
}
