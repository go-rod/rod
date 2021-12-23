package main

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	utils.ExecLine("npx -yq -- eslint@7.16.0 --config=lib/utils/lint/eslint.yml --ext=.js,.html --fix --ignore-path=.gitignore .")

	utils.ExecLine("npx -yq -- prettier@2.2.1 --loglevel=error --config=lib/utils/lint/prettier.yml --write --ignore-path=.gitignore .")

	utils.ExecLine(filepath.Join(build.Default.GOPATH, "bin", "golangci-lint"))

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
