package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func checkVersion(body string) error {
	m := regexp.MustCompile(`\*\*Rod Version:\*\* v[0-9.]+`).FindString(body)
	if m == "" || m == "**Rod Version:** v0.0.0" {
		return fmt.Errorf(
			"Please add a valid `**Rod Version:** v0.0.0` to your issue. Current version is %s",
			currentVer(),
		)
	}

	return nil
}

func currentVer() string {
	q := req("/repos/go-rod/rod/tags?per_page=1")
	res, err := http.DefaultClient.Do(q)
	utils.E(err)
	defer func() { _ = res.Body.Close() }()

	data, err := ioutil.ReadAll(res.Body)
	utils.E(err)

	currentVer := gson.New(data).Get("0.name").Str()

	return currentVer
}
