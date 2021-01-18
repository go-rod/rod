package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	run("npx -yq -- eslint@7.16.0 --config=lib/utils/lint/eslint.yml --ext=.js,.html --fix --ignore-path=.gitignore .")

	run("npx -yq -- prettier@2.2.1 --loglevel=error --config=lib/utils/lint/prettier.yml --write --ignore-path=.gitignore .")

	run("go run github.com/ysmood/golangci-lint")

	lintMustPrefix()

	run("go mod tidy")

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
		panic("Please run \"go generate\" on local and git commit the changes:\n" + out)
	}
}
