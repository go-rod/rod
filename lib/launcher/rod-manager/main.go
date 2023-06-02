// A server to help launch browser remotely
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

var (
	addr         = flag.String("address", ":7317", "the address to listen to")
	quiet        = flag.Bool("quiet", false, "silence the log")
	allowAllPath = flag.Bool("allow-all", false, "allow all path set by the client")
)

func main() {
	flag.Parse()

	m := launcher.NewManager()

	if !*quiet {
		m.Logger = log.New(os.Stdout, "", 0)
	}

	if *allowAllPath {
		m.BeforeLaunch = func(l *launcher.Launcher, rw http.ResponseWriter, r *http.Request) {}
	}

	l, err := net.Listen("tcp", *addr)
	if err != nil {
		utils.E(err)
	}

	if !*quiet {
		fmt.Println("[rod-manager] listening on:", l.Addr().String())
	}

	srv := &http.Server{Handler: m}
	utils.E(srv.Serve(l))
}
