package main

import (
	"log"
	"os/exec"
	"strings"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	run("npx -yq -- eslint --config=lib/utils/lint/eslint.yml --ext=.js,.html --fix --ignore-path=.gitignore .")

	run("npx -yq -- prettier --loglevel=error --config=lib/utils/lint/prettier.yml --write --ignore-path=.gitignore .")

	run("go run github.com/ysmood/golangci-lint/lint -- run --fix")

	run("go mod tidy")

	lintMustPrefix()

	checkGitClean()
}

func run(cmd string) {
	log.Println("[lint]", cmd)
	args := strings.Split(cmd, " ")
	utils.Exec(args[0], args[1:]...)
}

func checkGitClean() {
	b, err := exec.Command("git", "status", "--porcelain").CombinedOutput()
	utils.E(err)

	out := string(b)
	if out != "" {
		panic("Please run \"go generate\" on local and git commit the changes:" + out)
	}
}
