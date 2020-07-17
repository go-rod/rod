package main

import (
	"fmt"

	"github.com/go-rod/rod"
)

func main() {
	page := rod.New().Connect().Page("https://teslainventory.teslastats.no/")

	// Disable all alerts by making window.alert no-op.
	page.Eval(`window.alert = () => {}`)

	// Navigate through country, model options whitout worrying about alert messages.
	country := page.Element("#car_list_country")
	model := page.Element("#car_list_model")

	for _, c := range page.Elements("#car_list_country option") {
		country.Select(c.Text())

		for _, m := range page.Elements("#car_list_model option") {
			model.Select(m.Text())

			fmt.Println(c.Text(), m.Text(), "selected.")
		}
	}
}
