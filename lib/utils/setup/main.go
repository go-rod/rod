package main

import (
	"log"
	"os/exec"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	log.Println("setup project...")

	nodejsDeps()
	golangDeps()

	genDockerIgnore()
}

func nodejsDeps() {
	_, err := exec.LookPath("npm")
	if err != nil {
		log.Fatalln("please install Node.js: https://nodejs.org")
	}

	utils.Exec("npm", "i", "-q", "--no-audit", "--no-fund", "--silent", "eslint-plugin-html")
}

func golangDeps() {
	_, err := exec.Command("golangci-lint", "--version").CombinedOutput()
	if err != nil {
		log.Fatal("please install golangci-lint: https://golangci-lint.run")
	}
}

func genDockerIgnore() {
	s, err := utils.ReadString(".gitignore")
	utils.E(err)
	utils.E(utils.OutputFile(".dockerignore", s))
}
