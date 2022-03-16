package main

import "github.com/go-rod/rod/lib/utils"

func main() {
	utils.ExecLine("go run github.com/ysmood/golangci-lint@v0.5.0")

	utils.ExecLine("go test -coverprofile=coverage.out ./lib/launcher")
	utils.ExecLine("go run ./lib/utils/check-cov")

	utils.ExecLine("go test -coverprofile=coverage.out")
	utils.ExecLine("go run ./lib/utils/check-cov")
}
