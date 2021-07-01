package main

import "github.com/go-rod/rod/lib/utils"

func main() {
	utils.ExecLine("go install github.com/ysmood/golangci-lint@latest")
	utils.ExecLine("golangci-lint")

	utils.ExecLine("go test -coverprofile=coverage.out ./lib/launcher")
	utils.ExecLine("go run ./lib/utils/check-cov")

	utils.ExecLine("go test -coverprofile=coverage.out")
	utils.ExecLine("go run ./lib/utils/check-cov")
}
