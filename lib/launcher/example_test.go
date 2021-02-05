package launcher_test

import (
	"os/exec"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

func Example_use_system_browser() {
	if path, exists := launcher.LookPath(); exists {
		u := launcher.New().Bin(path).MustLaunch()
		rod.New().ControlURL(u).MustConnect()
	}
}

func Example_custom_launch() {
	// get the browser executable path
	path := launcher.NewBrowser().MustGet()

	// use the FormatArgs to construct args, this line is optional, you can construct the args manually
	args := launcher.New().FormatArgs()

	parser := launcher.NewURLParser()

	cmd := exec.Command(path, args...)
	cmd.Stderr = parser
	utils.E(cmd.Start())

	rod.New().ControlURL(<-parser.URL).MustConnect()
}
