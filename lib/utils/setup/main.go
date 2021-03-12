package main

import (
	"log"
	"os/exec"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	log.Println("setup project...")

	goDeps()

	nodejsDeps()

	genDockerIgnore()
}

func goDeps() {
	utils.Exec("go", "install", "github.com/ysmood/golangci-lint@latest")
}

func nodejsDeps() {
	_, err := exec.LookPath("npm")
	if err != nil {
		log.Fatalln("please install Node.js: https://nodejs.org")
	}

	utils.Exec("npm", "i", "-q", "--no-audit", "--no-fund", "--silent", "eslint-plugin-html")
}

func genDockerIgnore() {
	s, err := utils.ReadString(".gitignore")
	utils.E(err)
	utils.E(utils.OutputFile(".dockerignore", s))
}
