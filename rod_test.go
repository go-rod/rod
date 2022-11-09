package rod_test

import (
	"testing"
)

// This is the template to demonstrate how to test Rod.
func TestRod(t *testing.T) {
	g := setup(t)
	g.cancelTimeout() // Cancel timeout protection

	// You can use the pre-launched g.browser or g.page for testing
	browser, page := g.browser, g.page

	// You can also use the g.html to serve static html content
	page.MustNavigate(g.html(doc)).MustWaitLoad()

	g.Eq(browser.MustVersion().ProtocolVersion, "1.3")
	g.Has(page.MustElement("body").MustText(), "ok")
}

const doc = `
<html>
  <body>ok</body>
</html>
`
