package launcher_test

import (
	"os/exec"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

func Example_custom_launch() {
	// get the browser executable path
	bin, err := launcher.NewBrowser().Get()
	utils.E(err)

	// use the helper to construct args, this line is optional, you can construct the args manually
	args := launcher.New().Headless(false).Env("TZ=Asia/Tokyo").FormatArgs()

	parser := launcher.NewURLParser()

	cmd := exec.Command(bin, args...)
	cmd.Stderr = parser
	err = cmd.Start()
	utils.E(err)

	rod.New().ControlURL(<-parser.URL).MustConnect()
}

func Example_disable_auto_download() {
	path, found := launcher.NewBrowser().LookPath()
	if found {
		// Check the doc for Bin to learn why
		u := launcher.New().Bin(path).MustLaunch()
		rod.New().ControlURL(u).MustConnect()
	}
}
