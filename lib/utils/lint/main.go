package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	utils.ExecLine("npx -yq -- eslint@8.7.0 --config=lib/utils/lint/eslint.yml --ext=.js,.html --fix --ignore-path=.gitignore .")

	utils.ExecLine("npx -yq -- prettier@2.5.1 --loglevel=error --config=lib/utils/lint/prettier.yml --write --ignore-path=.gitignore .")

	utils.ExecLine("go run github.com/ysmood/golangci-lint@v0.5.0")

	lintMustPrefix()

	checkGitClean()
}

func checkGitClean() {
	b, err := exec.Command("git", "status", "--porcelain").CombinedOutput()
	utils.E(err)

	out := string(b)
	if out != "" {
		panic("Please run \"go generate\" on local and git commit the changes:\n" + out)
	}
}
