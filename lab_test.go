package rod_test

import (
	"testing"
)

// This is the template to demonstrate how to test Rod.
func TestLab(t *testing.T) {
	g := setup(t)
	g.cancelTimeout() // Cancel timeout protection

	browser, page := g.browser, g.page

	// You can use the pre-launched g.browser for testing
	g.Eq(browser.MustVersion().ProtocolVersion, "1.3")

	// You can also use the pre-created g.page for testing
	page.MustNavigate(g.blank()).MustWaitLoad()
	g.Has(page.MustInfo().URL, "blank.html")
}
