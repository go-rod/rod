package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

var addr = flag.String("address", ":9222", "the address to listen to")
var quiet = flag.Bool("quiet", false, "silent the log")

// a cli tool to launch browser remotely
func main() {
	flag.Parse()

	rl := launcher.NewRemoteLauncher()
	if !*quiet {
		rl.Logger = os.Stdout
	}

	l, err := net.Listen("tcp", *addr)
	if err != nil {
		utils.E(err)
	}

	fmt.Println("Remote control url is", "ws://"+l.Addr().String())

	srv := &http.Server{Handler: rl}
	utils.E(srv.Serve(l))
}
