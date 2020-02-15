package cdp_test

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/launcher"
)

func ExampleClient() {
	ctx, cancel := context.WithCancel(context.Background())

	url := launcher.New().Launch()

	client, err := cdp.New(ctx, cancel, url)
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
