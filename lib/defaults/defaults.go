// Package defaults of commonly used options parsed from environment.
// Check ResetWith for details.
package defaults

import (
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

// Trace is the default of rod.Browser.Trace .
// Option name is "trace".
var Trace bool

// Slow is the default of rod.Browser.SlowMotion .
// The format is same as https://golang.org/pkg/time/#ParseDuration
// Option name is "slow".
var Slow time.Duration

// Monitor is the default of rod.Browser.ServeMonitor .
// Option name is "monitor".
var Monitor string

// Show is the default of launcher.Launcher.Headless .
// Option name is "show".
var Show bool

// Devtools is the default of launcher.Launcher.Devtools .
// Option name is "devtools".
var Devtools bool

// Dir is the default of launcher.Launcher.UserDataDir .
// Option name is "dir".
var Dir string

// Port is the default of launcher.Launcher.RemoteDebuggingPort .
// Option name is "port".
var Port string

// Bin is the default of launcher.Launcher.Bin .
// Option name is "bin".
var Bin string

// Proxy is the default of launcher.Launcher.Proxy
// Option name is "proxy".
var Proxy string

// LockPort is the default of launcher.Browser.LockPort
// Option name is "lock".
var LockPort int

// URL is the default websocket url for remote control a browser.
// Option name is "url".
var URL string

// CDP is the default of cdp.Client.Logger
// Option name is "cdp".
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
	"cdp": func(_ string) {
		CDP = log.New(log.Writer(), "[cdp] ", log.LstdFlags)
	},
}

// Parse the flags.
func init() {
	ResetWith("")
}

// ResetWith options and "-rod" command line flag.
// It will be called in an init() , so you don't have to call it manually.
// It will try to load the cli flag "-rod" and then the options, the later override the former.
// If you want to disable the global cli argument flag, set env DISABLE_ROD_FLAG.
// Values are separated by commas, key and value are separated by "=". For example:
//
//	go run main.go -rod=show
//	go run main.go -rod show,trace,slow=1s,monitor
//	go run main.go --rod="slow=1s,dir=path/has /space,monitor=:9223"
func ResetWith(options string) {
	Reset()

	if _, has := os.LookupEnv("DISABLE_ROD_FLAG"); !has {
		if !flag.Parsed() && flag.Lookup("rod") == nil {
			flag.String("rod", "", `Set the default value of options used by rod.`)
		}

		parseFlag(os.Args)
	}

	parse(options)
}

func parseFlag(args []string) {
	reg := regexp.MustCompile(`^--?rod$`)
	regEq := regexp.MustCompile(`^--?rod=(.*)$`)
	opts := ""
	for i, arg := range args {
		if reg.MatchString(arg) && i+1 < len(args) {
			opts = args[i+1]
		} else if m := regEq.FindStringSubmatch(arg); len(m) == 2 {
			opts = m[1]
		}
	}

	parse(opts)
}

// parse options and set them globally.
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
