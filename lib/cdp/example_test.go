package cdp_test

import (
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/launcher"
)

func ExampleClient() {
	// launch a chrome
	url := launcher.New().Headless(false).Launch()

	// create a controller
	client := cdp.New(url).Connect()

	// Such as call this endpoint on the api doc:
	// https://chromedevtools.github.io/devtools-protocol/tot/Page#method-navigate
	// This will create a new tab and navigate to the test.com
	res, err := client.Call(nil, &cdp.Request{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": "https://google.com",
		},
	})
	kit.E(err)

	kit.Log(res.Get("targetId").String())

	kit.Pause()

	// Skip
	// Output: id
}
