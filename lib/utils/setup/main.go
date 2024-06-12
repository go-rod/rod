// Package main ...
package main

import (
	"log"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	log.Println("setup project...")

	golangDeps()

	nodejsDeps()

	genDockerIgnore()
}

func golangDeps() {
	utils.Exec("go mod download")
	utils.Exec("go install mvdan.cc/gofumpt@latest")
}

func nodejsDeps() {
	utils.UseNode(true)

	utils.Exec("npm i -s eslint-plugin-html")
}

func genDockerIgnore() {
	s, err := utils.ReadString(".gitignore")
	utils.E(err)
	utils.E(utils.OutputFile(".dockerignore", s))
}
