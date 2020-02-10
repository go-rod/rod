package cdp_test

import (
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

func ExampleClient() {
	url, err := cdp.LaunchBrowser("", nil)
	kit.E(err)

	client, err := cdp.New(nil, url)
	kit.E(err)

	// Such as call this endpoint on the api doc:
	// https://chromedevtools.github.io/devtools-protocol/tot/Page#method-navigate
	// This will create a new tab and navigate to the test.com
	res, err := client.Call(nil, &cdp.Message{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": "https://google.com",
		},
	})
	kit.E(err)

	kit.Log(res.Get("targetId").String())
}
