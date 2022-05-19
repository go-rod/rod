package rod_test

import (
	"testing"
)

// This is the template to demonstrate how to test Rod.
func TestLab(t *testing.T) {
	g := setup(t)
	browser, page := g.browser, g.page

	// You can use the pre-launched g.browser for testing
	g.Eq(browser.MustVersion().ProtocolVersion, "1.3")

	// You can also use the pre-created g.page for testing
	page.MustNavigate(g.blank())
	g.Has(page.MustInfo().URL, "blank.html")
}
