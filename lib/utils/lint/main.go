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

	run("go mod tidy")

	run("golangci-lint run --fix ./...")

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
		panic("Changes of \"go generate\", \"lint auto fix\", etc are not git committed:\n" + out)
	}
}
