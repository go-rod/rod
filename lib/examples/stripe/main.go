// Package main ...
package main

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/go-rod/rod"
	"github.com/ysmood/gson"
)

// An example to handle stripe 3DS callback.
func main() {
	page := rod.New().MustConnect().MustPage(getRedirectURL())

	// Get the button from the nested iframes
	frame01 := page.MustElement("div iframe").MustFrame()
	frame02 := frame01.MustElement("#challengeFrame").MustFrame()
	btn := frame02.MustElementR("button", "COMPLETE").MustWaitStable()

	wait := frame02.MustWaitRequestIdle()
	btn.MustClick()
	wait()
}

// Create a card payment that requires Visa's confirmation
func getRedirectURL() string {
	token := post(
		"/tokens", "card[number]=4000000000003220&card[exp_month]=7&card[exp_year]=2025&card[cvc]=314",
	).Get("id").Str()

	return post(
		"/payment_intents",
		"amount=100&currency=usd&payment_method_data[type]=card&confirm=true&return_url=https%3A%2F%2Fmdn.dev"+ // cSpell:ignore Fmdn
			"&payment_method_data[card][token]="+token,
	).Get("next_action.redirect_to_url.url").Str()
}

func post(path, body string) gson.JSON {
	req, _ := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1"+path, bytes.NewBufferString(body))
	req.Header.Add("Authorization", "Bearer sk_test_4eC39HqLyjWDarjtT1zdp7dc") // cSpell:ignore Darjt
	res, _ := http.DefaultClient.Do(req)
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}
	data, _ := ioutil.ReadAll(res.Body)
	return gson.New(data)
}
