package main

import (
	"testing"

	"github.com/ysmood/got"
)

func TestBasic(t *testing.T) {
	g := got.T(t)

	body := g.Read(g.Open(false, "body-invalid.txt")).String()

	g.Eq(check(body), ""+
		"Please add a valid `**Rod Version:** v0.0.0` to your issue. Current version is <nil>\n"+
		"\n"+
		"Please fix the format of your markdown:\n"+
		"\n"+
		"```txt\n"+
		"stdin:5 MD040/fenced-code-language Fenced code blocks should have a language specified [Context: \"```\"]\n"+
		"stdin:26:24 MD009/no-trailing-spaces Trailing spaces [Expected: 0 or 2; Actual: 1]\n"+
		"```\n"+
		"\n"+
		"Please fix the golang code in your markdown:\n"+
		"\n"+
		"```@@ go markdown error @@\n"+
		"4:5: invalid import path: \"testing (and 1 more errors)\n"+
		"```")

	body = g.Read(g.Open(false, "body.txt")).String()
	g.Zero(check(body))
}
