package main

import (
	"os"
	"testing"

	"github.com/ysmood/got"
)

func TestBasic(t *testing.T) {
	g := got.T(t)

	_ = os.Setenv("ROD_GITHUB_ROBOT", "1234")

	body := g.Read(g.Open(false, "body-invalid.txt")).String()

	g.Eq(check(body), ""+
		"Please add a valid `Rod Version: v0.0.0` to your issue. Current version is <nil>\n"+
		"\n"+
		"Please fix the format of your markdown:\n"+
		"\n"+
		"```txt\n"+
		"5 MD040/fenced-code-language Fenced code blocks should have a language specified [Context: \"```\"]\n"+
		"20:24 MD009/no-trailing-spaces Trailing spaces [Expected: 0 or 2; Actual: 1]\n"+
		"```\n"+
		"\n"+
		"Please fix the golang code in your markdown:\n"+
		"\n"+
		"```txt\n"+
		"@@ golang markdown block 1 @@\n"+
		"4:15: expected ';', found 'EOF'\n"+
		"4:15: expected '}', found 'EOF'\n"+
		"```")

	body = g.Read(g.Open(false, "body.txt")).String()
	g.Zero(check(body))
}
