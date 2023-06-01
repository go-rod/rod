package proto

import (
	"regexp"
	"strings"
)

var (
	regAsterisk  = regexp.MustCompile(`([^\\])\*`)
	regBackSlash = regexp.MustCompile(`([^\\])\?`)
)

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
