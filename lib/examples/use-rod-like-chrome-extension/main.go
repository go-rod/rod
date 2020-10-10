package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// For example, when you log into your github account, and you want to reuse the login session for automation task.
// You can use this example to achieve such functionality. Rod will be just like your browser extension.
func main() {
	// Make sure you have closed your browser completely, UserMode can't control a browser that is not launched by it.
	// Launches a new browser with the "new user mode" option, and returns the URL to control that browser.
	wsURL := launcher.NewUserMode().MustLaunch()

	browser := rod.New().ControlURL(wsURL).MustConnect().DefaultDevice(devices.Clear, false)

	// Whenever we hover to a link, popup a preview of it.
	go browser.EachEvent(func(e *proto.TargetTargetCreated) {
		if e.TargetInfo.Type != proto.TargetTargetInfoTypePage {
			return
		}

		page := browser.MustPageFromTargetID(e.TargetInfo.TargetID)

		page.MustEvalOnNewDocument(fmt.Sprintf(`(async () => {
			await new Promise((r) => window.addEventListener('load', r))

			%s

			function setup(el) {
				el.classList.add('x-set')
				tippy(el, {onShow: (it) => {
					if (it.props.content.src) return
					let img = document.createElement('img')
					img.style.height = '800px'
					img.src = location.origin + '/rod-preview?url=' + encodeURIComponent(el.href)
					img.onload = () => it.setContent(img)
				}, content: 'loading...', maxWidth: 500})
			}

			(function check() {
				Array.from(document.querySelectorAll('a:not(.x-set)')).forEach(setup)
				setTimeout(check, 1000)
			})()
		})()`, jsLib))
	})()

	// Create a headless browser to generate preview of links on background.
	previewer := rod.New().MustConnect().DefaultDevice(devices.IPhone6or7or8, false)

	previewer.MustSetCookies(browser.MustGetCookies()) // share cookies

	browser.HijackRequests().MustAdd("*/rod-preview*", func(h *rod.Hijack) {
		u := h.Request.URL().Query().Get("url")
		page := previewer.MustPage(u).MustWaitLoad()
		defer page.MustClose()
		h.Response.SetBody(page.MustScreenshot())
	}).Run()
}

var jsLib = get("https://unpkg.com/@popperjs/core@2") + get("https://unpkg.com/tippy.js@6")

func get(u string) string {
	res, err := http.Get(u)
	utils.E(err)
	b, err := ioutil.ReadAll(res.Body)
	utils.E(err)
	return string(b)
}
