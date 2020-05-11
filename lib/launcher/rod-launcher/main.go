package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/defaults"
	"github.com/ysmood/rod/lib/launcher"
)

func main() {
	app := kit.TasksNew("rod-launcher", "a cli tool to launch chrome remotely")
	app.Version(defaults.Version)

	kit.Tasks().App(app).Add(kit.Task("serve", "start server").Init(func(cmd kit.TaskCmd) func() {
		cmd.Default()
		addr := cmd.Arg("address", "the address to listen to").Default(":9222").String()
		quiet := cmd.Flag("quiet", "silent the log").Short('q').Bool()

		return func() {
			proxy := &launcher.Proxy{
				Log: func(s string) {
					if !*quiet {
						fmt.Println(s)
					}
				},
			}

			srv := kit.MustServer(*addr)
			srv.Engine.NoRoute(gin.WrapH(proxy))
			fmt.Println("Remote control url is", kit.C("ws://"+srv.Listener.Addr().String(), "green"))
			srv.MustDo()
		}
	})).Do()
}
