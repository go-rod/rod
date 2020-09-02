package main

import (
	"log"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	log.Println("npx eslint --ext .js,.html .")
	utils.Exec("npx", "eslint", "--ext", ".js,.html", ".")

	log.Println("npx prettier --loglevel error --write .")
	utils.Exec("npx", "prettier", "--loglevel", "error", "--write", ".")

	log.Println("godev lint")
	utils.Exec("godev", "lint")
}
