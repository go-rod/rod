package main

import (
	"path/filepath"
	"strings"
)

var slash = filepath.FromSlash

func main() {
	helper()
}

// not using encoding like base64 or gzip because of they will make git diff every large for small change
func encode(s string) string {
	return strings.ReplaceAll(s, "`", "` + \"`\" + `")
}
