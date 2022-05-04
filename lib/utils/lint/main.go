package main

import (
	"fmt"
	"os"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	utils.Exec("npx -ys -- eslint@8.7.0 --config=lib/utils/lint/eslint.yml --ext=.js,.html --fix --ignore-path=.gitignore .")

	utils.Exec("npx -ys -- prettier@2.5.1 --loglevel=error --config=lib/utils/lint/prettier.yml --write --ignore-path=.gitignore .")

	utils.Exec("go run github.com/ysmood/golangci-lint@latest")

	lintMustPrefix()

	checkGitClean()
}

func checkGitClean() {
	out := utils.ExecLine(false, "git status --porcelain")
	if out != "" {
		panic("Please run \"go generate\" on local and git commit the changes:\n" + out)
	}
}
