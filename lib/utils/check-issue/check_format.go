package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os/exec"
	"regexp"
	"strings"
)

func checkMarkdown(body string) error {
	cmd := strings.Split("npx -ys -- markdownlint-cli@0.31.1 -s --disable MD041 MD034 MD013", " ")
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin = bytes.NewBufferString(body)
	b, err := c.CombinedOutput()
	if err == nil {
		return nil
	}

	return fmt.Errorf("Please fix the format of your markdown:\n\n```txt\n%s```", b)
}

func checkGoCode(body string) error {
	reg := regexp.MustCompile("(?s)```go\r?\n(.+)```")

	errs := []string{}
	for _, m := range reg.FindAllStringSubmatch(body, -1) {
		_, err := parser.ParseFile(token.NewFileSet(), "", m[1], parser.AllErrors)
		if err != nil {
			errs = append(errs, "@@ go markdown error @@\n"+err.Error())
		}
	}

	if len(errs) != 0 {
		return errors.New("Please fix the golang code in your markdown:\n\n```" + strings.Join(errs, "\n\n") + "\n```")
	}

	return nil
}
