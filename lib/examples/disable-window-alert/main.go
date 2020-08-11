package main

import (
	"fmt"

	"github.com/go-rod/rod"
)

func main() {
	page := rod.New().MustConnect().MustPage("https://teslainventory.teslastats.no/")

	// Disable all alerts by making window.alert no-op.
	page.MustEval(`window.alert = () => {}`)

	// Navigate through country, model options whitout worrying about alert messages.
	country := page.MustElement("#car_list_country")
	model := page.MustElement("#car_list_model")

	for _, c := range page.MustElements("#car_list_country option") {
		country.MustSelect(c.MustText())

		for _, m := range page.MustElements("#car_list_model option") {
			model.MustSelect(m.MustText())

			fmt.Println(c.MustText(), m.MustText(), "selected.")
		}
	}
}
