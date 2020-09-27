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

// Trace is the default of rod.Browser.Trace
var Trace bool

// Slow is the default of rod.Browser.Slowmotion
var Slow time.Duration

// Monitor is the default of rod.Browser.ServeMonitor
var Monitor string

// Show is the default of launcher.Launcher.Headless
var Show bool

// Devtools is the default of launcher.Launcher.Devtools
var Devtools bool

// Dir is the default of launcher.Launcher.UserDataDir
var Dir string

// Port is the default of launcher.Launcher.RemoteDebuggingPort
var Port string

// Bin is the default of launcher.Launcher.Bin
var Bin string

// Proxy is the default of launcher.Launcher.Proxy
var Proxy string

// URL is the default of cdp.Client.New
var URL string

// CDP is the default of cdp.Client.Debug
var CDP bool

// Parse the flags
func init() {
	ResetWithEnv()
}

// Reset all flags to their init values.
func Reset() {
	Show = false
	Devtools = false
	Trace = false
	Slow = 0
	Dir = ""
	Port = "0"
	Bin = ""
	URL = ""
	CDP = false
	Monitor = ""
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
	"devtools": func(string) {
		Devtools = true
	},
	"trace": func(string) {
		Trace = true
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
	"cdp": func(v string) {
		CDP = true
	},
	"monitor": func(v string) {
		Monitor = ":0"
		if v != "" {
			Monitor = v
		}
	},
	"proxy": func(v string) {
		Proxy = v
	},
}
