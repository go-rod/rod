// Package defaults of commonly used options parsed from environment.
// Check ResetWithEnv for details.
package defaults

import (
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

// Trace is the default of rod.Browser.Trace .
// Env name is "trace".
var Trace bool

// Slow is the default of rod.Browser.Slowmotion .
// The format is same as https://golang.org/pkg/time/#ParseDuration
// Env name is "slow".
var Slow time.Duration

// Monitor is the default of rod.Browser.ServeMonitor .
// Env name is "monitor".
var Monitor string

// Show is the default of launcher.Launcher.Headless .
// Env name is "show".
var Show bool

// Devtools is the default of launcher.Launcher.Devtools .
// Env name is "devtools".
var Devtools bool

// Dir is the default of launcher.Launcher.UserDataDir .
// Env name is "dir".
var Dir string

// Port is the default of launcher.Launcher.RemoteDebuggingPort .
// Env name is "port".
var Port string

// Bin is the default of launcher.Launcher.Bin .
// Env name is "bin".
var Bin string

// Proxy is the default of launcher.Launcher.Proxy
// Env name is "trace".
var Proxy string

// LockPort is the default of launcher.Browser.LockPort
// Env name is "lock".
var LockPort int

// URL is the default of cdp.Client.New .
// Env name is "url".
var URL string

// CDP is the default of cdp.Client.Logger
// Env name is "cdp".
var CDP utils.Logger

// Reset all flags to their init values.
func Reset() {
	Trace = false
	Slow = 0
	Monitor = ""
	Show = false
	Devtools = false
	Dir = ""
	Port = "0"
	Bin = ""
	Proxy = ""
	LockPort = 2978
	URL = ""
	CDP = utils.LoggerQuiet
}

var envParsers = map[string]func(string){
	"trace": func(string) {
		Trace = true
	},
	"slow": func(v string) {
		var err error
		Slow, err = time.ParseDuration(v)
		if err != nil {
			msg := "invalid value for \"slow\": " + err.Error() +
				" (learn format from https://golang.org/pkg/time/#ParseDuration)"
			panic(msg)
		}
	},
	"monitor": func(v string) {
		Monitor = ":0"
		if v != "" {
			Monitor = v
		}
	},
	"show": func(string) {
		Show = true
	},
	"devtools": func(string) {
		Devtools = true
	},
	"dir": func(v string) {
		Dir = v
	},
	"port": func(v string) {
		Port = v
	},
	"bin": func(v string) {
		Bin = v
	},
	"proxy": func(v string) {
		Proxy = v
	},
	"lock": func(v string) {
		i, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			LockPort = int(i)
		}
	},
	"url": func(v string) {
		URL = v
	},
	"cdp": func(v string) {
		CDP = log.New(log.Writer(), "[cdp] ", log.LstdFlags)
	},
}

// Parse the flags
func init() {
	ResetWithEnv("")
}

// ResetWithEnv set the default value of options used by rod.
// It will be called in an init() , so you don't have to call it manually.
// The followings will be parsed and merged, later overrides previous:
//
//     os.Open(".rod")
//     os.Getenv("rod")
//     env
//
// Values are separated by commas, key and value are separated by "=",
// For example, on unix-like OS:
//
//    rod="show,trace,slow=1s,monitor" go run main.go
//
//    rod="slow=1s,dir=path/has /space,monitor=:9223" go run main.go
//
// An example of ".rod" file content:
//
//    slow=1s
//    dir=path/has /space
//    monitor=:9223
//
func ResetWithEnv(env string) {
	Reset()

	b, _ := ioutil.ReadFile(".rod")
	parse(string(b))

	parse(os.Getenv("rod"))

	parse(env)
}

// parse options and set them globally
func parse(options string) {
	if options == "" {
		return
	}

	reg := regexp.MustCompile(`[,\r\n]`)

	for _, str := range reg.Split(options, -1) {
		kv := strings.SplitN(str, "=", 2)

		v := ""
		if len(kv) == 2 {
			v = kv[1]
		}

		n := strings.TrimSpace(kv[0])
		if n == "" {
			continue
		}

		f := envParsers[n]
		if f == nil {
			panic("unknown rod env option: " + n)
		}
		f(v)
	}
}
