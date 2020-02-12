package cdp_test

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

func ExampleClient() {
	ctx := context.Background()

	url, err := cdp.LaunchBrowser("", nil)
	kit.E(err)

	client, err := cdp.New(ctx, url)
	kit.E(err)

	// Such as call this endpoint on the api doc:
	// https://chromedevtools.github.io/devtools-protocol/tot/Page#method-navigate
	// This will create a new tab and navigate to the test.com
	res, err := client.Call(ctx, &cdp.Request{
		Method: "Target.createTarget",
		Params: cdp.Object{
			"url": "https://google.com",
		},
	})
	kit.E(err)

	kit.Log(res.Get("targetId").String())
}
