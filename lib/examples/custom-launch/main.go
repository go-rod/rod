package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
)

func main() {
	var container string
	var browser *rod.Browser

	const debug = true
	const incognito = true
	const ignoreCertErrors = true
	const remoteContainerAddr = "127.0.0.1:9222"
	const proxyAddr = "127.0.0.1:8001"

	if debug {
		l := launcher.New().Headless(false).Devtools(true)
		if proxyAddr != "" {
			l.Set(flags.ProxyServer, proxyAddr)
		}
		container = l.MustLaunch()
		browser = rod.New().ControlURL(container)
		browser.Trace(true).SlowMotion(2 * time.Second)
	} else {
		container = launcher.MustResolveURL(remoteContainerAddr)
		browser = rod.New().ControlURL(container)
	}

	if ignoreCertErrors {
		browser.MustIgnoreCertErrors(true)
	}
	if incognito {
		browser.MustIncognito()
	}
	browser.MustConnect()

	page := browser.MustPage()
	page.MustEvalOnNewDocument(`window.alert = () => {}`)
	page.MustEvalOnNewDocument(`window.prompt = () => {}`)
	page.MustNavigate("https://www.bilibili.com/read/home")
	page.MustWaitLoad()
	page.MustElement(".article-item")

	fmt.Println(
		page.MustEval("() => document.title"),
	)

	result := page.MustEval(`() => Array.from(document.querySelectorAll('.article-list .article-item .article-title')).map(article=>article.innerText).join("\n")`)
	fmt.Println(result)
}
