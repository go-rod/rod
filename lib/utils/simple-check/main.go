package main

import "github.com/go-rod/rod/lib/utils"

func main() {
	utils.Exec("go run github.com/ysmood/golangci-lint@latest")

	utils.Exec("go test -coverprofile=coverage.out ./lib/launcher")
	utils.Exec("go run ./lib/utils/check-cov")

	utils.Exec("go test -coverprofile=coverage.out")
	utils.Exec("go run ./lib/utils/check-cov")
}
