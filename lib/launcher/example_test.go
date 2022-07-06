package launcher_test

import (
	"os"
	"os/exec"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/leakless"
)

func Example_use_system_browser() {
	if path, exists := launcher.LookPath(); exists {
		u := launcher.New().Bin(path).MustLaunch()
		rod.New().ControlURL(u).MustConnect()
	}
}

func Example_print_browser_CLI_output() {
	// Pipe the browser stderr and stdout to os.Stdout .
	u := launcher.New().Logger(os.Stdout).MustLaunch()
	rod.New().ControlURL(u).MustConnect()
}

func Example_custom_launch() {
	// get the browser executable path
	path := launcher.NewBrowser().MustGet()

	// use the FormatArgs to construct args, this line is optional, you can construct the args manually
	args := launcher.New().FormatArgs()

	var cmd *exec.Cmd
	if true { // decide whether to use leakless or not
		cmd = leakless.New().Command(path, args...)
	} else {
		cmd = exec.Command(path, args...)
	}

	parser := launcher.NewURLParser()
	cmd.Stderr = parser
	utils.E(cmd.Start())
	u := launcher.MustResolveURL(<-parser.URL)

	rod.New().ControlURL(u).MustConnect()
}
