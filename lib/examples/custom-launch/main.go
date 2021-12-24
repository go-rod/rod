package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func main() {
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).MustLaunch()
	mypage := rod.New().ControlURL(u).MustConnect().MustPage()
	mypage1 := mypage.Timeout(30 * time.Second).MustNavigate("https://sometimes.gitee.io/toutalk/2021/12/22/hello-world/")
	mypage1.Race().Element("iframe").MustHandle(func(e *rod.Element) {
		// 获取iframe
		fmt.Println(e.MustHTML())
		fmt.Println(e.MustFrame().MustHTML())
		iframe01 := e.MustFrame()
		el := iframe01.MustElement("a")
		fmt.Println(el.MustHTML())
	}).MustDo()

}
