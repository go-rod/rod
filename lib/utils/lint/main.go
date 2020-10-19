package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"strings"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	exec("npx -q eslint --config=lib/utils/lint/eslint.yml --ext=.js,.html --fix --ignore-path=.gitignore .")

	exec("npx -q prettier --loglevel=error --config=lib/utils/lint/prettier.yml --write --ignore-path=.gitignore .")

	exec("godev lint")

	lintMustPrefix()
}

func exec(cmd string) {
	log.Println(cmd)
	args := strings.Split(cmd, " ")
	utils.Exec(args[0], args[1:]...)
}

func lintMustPrefix() {
	log.Println("lint 'Must' prefix")

	paths, err := filepath.Glob("*.go")
	utils.E(err)
	lintErr := false

	for _, p := range paths {
		name := filepath.Base(p)
		if name == "must.go" || strings.HasSuffix(name, "_test.go") {
			continue
		}

		src, err := utils.ReadString(p)
		utils.E(err)

		list := token.NewFileSet()
		f, err := parser.ParseFile(list, p, src, 0)
		if err != nil {
			panic(err)
		}

		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if ok && strings.HasPrefix(fd.Name.Name, "Must") {
				log.Printf("%s %s\n", list.Position(fd.Name.Pos()), fd.Name.Name)
				lintErr = true
			}
		}
		break
	}

	if lintErr {
		log.Fatalln("'Must' prefixed function should be declared in file 'must.go'")
	}
}
