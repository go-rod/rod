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

	// use the helper to set flags, this line it's optional, you can construct the args manually
	args := launcher.New().Headless(false).FormatArgs()

	parser := launcher.NewURLParser()

	cmd := exec.Command(bin, args...)
	cmd.Stderr = parser
	err = cmd.Start()
	utils.E(err)

	rod.New().ControlURL(<-parser.URL).MustConnect()
}
