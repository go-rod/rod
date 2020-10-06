package proto

import (
	"reflect"
	"regexp"
	"strings"
)

var regAsterisk = regexp.MustCompile(`([^\\])\*`)
var regBackSlash = regexp.MustCompile(`([^\\])\?`)

// PatternToReg FetchRequestPattern.URLPattern to regular expression
func PatternToReg(pattern string) string {
	if pattern == "" {
		return ""
	}

	pattern = " " + pattern
	pattern = regAsterisk.ReplaceAllString(pattern, "$1.*")
	pattern = regBackSlash.ReplaceAllString(pattern, "$1.")

	return `\A` + strings.TrimSpace(pattern) + `\z`
}

// assign each fields from src to dst
func assign(src, dst interface{}) {
	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst).Elem()

	l := srcVal.NumField()
	for i := 0; i < l; i++ {
		dstVal.Field(i).Set(srcVal.Field(i))
	}
}
