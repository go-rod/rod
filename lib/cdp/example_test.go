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
	// launch a browser
	url := launcher.New().Headless(false).MustLaunch()

	// create a controller
	client := cdp.New(url).MustConnect()

	// Such as call this endpoint on the api doc:
	// https://chromedevtools.github.io/devtools-protocol/tot/Page#method-navigate
	// This will create a new tab and navigate to the test.com
	res, err := client.Call(context.Background(), "", "Target.createTarget", map[string]string{
		"url": "https://google.com",
	})
	utils.E(err)

	fmt.Println(gjson.ParseBytes(res).Get("targetId").String())

	utils.Pause()

	// Skip
	// Output: id
}
