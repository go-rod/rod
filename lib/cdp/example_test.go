package cdp_test

import (
	"context"
	"fmt"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/tidwall/gjson"
)

func ExampleClient() {
	ctx := context.Background()

	// launch a browser
	url := launcher.New().MustLaunch()

	// create a controller
	client := cdp.New(url).MustConnect(ctx)

	go func() {
		for range client.Event() {
			// you must consume the events
		}
	}()

	// Such as call this endpoint on the api doc:
	// https://chromedevtools.github.io/devtools-protocol/tot/Page#method-navigate
	// This will create a new tab and navigate to the test.com
	res, err := client.Call(ctx, "", "Target.createTarget", map[string]string{
		"url": "http://test.com",
	})
	utils.E(err)

	fmt.Println(len(gjson.ParseBytes(res).Get("targetId").Str))

	// close browser
	_, err = client.Call(ctx, "", "Browser.close", nil)
	utils.E(err)

	// Output: 32
}
