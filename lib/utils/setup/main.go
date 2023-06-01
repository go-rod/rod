// Package main ...
package main

import (
	"log"
	"os/exec"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	log.Println("setup project...")

	golangDeps()

	nodejsDeps()

	genDockerIgnore()
}

func golangDeps() {
	utils.Exec("go install mvdan.cc/gofumpt@latest")
}

func nodejsDeps() {
	_, err := exec.LookPath("npm")
	if err != nil {
		log.Fatalln("please install Node.js: https://nodejs.org")
	}

	utils.Exec("npm i -s eslint-plugin-html")
}

func genDockerIgnore() {
	s, err := utils.ReadString(".gitignore")
	utils.E(err)
	utils.E(utils.OutputFile(".dockerignore", s))
}
