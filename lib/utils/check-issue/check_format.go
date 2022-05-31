package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"os/exec"
	"regexp"
	"strings"
)

func checkMarkdown(body string) error {
	cmd := strings.Split("npx -ys -- markdownlint-cli@0.31.1 -s --disable MD041 MD034 MD013 MD047 MD010", " ")
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin = bytes.NewBufferString(body)
	b, err := c.CombinedOutput()
	if err == nil {
		return nil
	}

	b = regexp.MustCompile(`(?m)^stdin:`).ReplaceAll(b, []byte{})

	return fmt.Errorf("Please fix the format of your markdown:\n\n```txt\n%s```", b)
}

func checkGoCode(body string) error {
	reg := regexp.MustCompile("(?s)```go\r?\n(.+?)```")

	errs := []string{}
	i := 0
	for _, m := range reg.FindAllStringSubmatch(body, -1) {
		_, err := parser.ParseFile(token.NewFileSet(), "", m[1], parser.AllErrors)
		if list, ok := err.(scanner.ErrorList); ok {
			i++
			errs = append(errs, fmt.Sprintf("@@ golang markdown block %d @@", i))
			for _, err := range list {
				errs = append(errs, err.Error())
			}
		}
	}

	if len(errs) != 0 {
		return errors.New("Please fix the golang code in your markdown:\n\n```txt\n" + strings.Join(errs, "\n") + "\n```")
	}

	return nil
}
