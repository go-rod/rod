package main

import (
	"github.com/go-rod/rod"
)

func main() {

	browser := rod.New().Connect()

	page := browser.Pages()[0]

	page.SetUserAgent(nil)

	page.EvalOnNewDocument(`

// https://github.com/chromedp/chromedp/issues/396#issuecomment-503351342

(function(w, n, wn) {
  // Pass the Webdriver Test.
  Object.defineProperty(n, 'webdriver', {
    get: () => false,
  });

  // Pass the Plugins Length Test.
  // Overwrite the plugins property to use a custom getter.
  Object.defineProperty(n, 'plugins', {
    // This just needs to have length > 0 for the current test,
    // but we could mock the plugins too if necessary.
    get: () => [1, 2, 3, 4, 5],
  });

  // Pass the Languages Test.
  // Overwrite the plugins property to use a custom getter.
  Object.defineProperty(n, 'languages', {
    get: () => ['en-US', 'en'],
  });

  // Pass the Chrome Test.
  // We can mock this in as much depth as we need for the test.
  w.chrome = {
    runtime: {},
  };

  // Pass the Permissions Test.
  const originalQuery = wn.permissions.query;
  return wn.permissions.query = (parameters) => (
    parameters.name === 'notifications' ?
      Promise.resolve({ state: Notification.permission }) :
      originalQuery(parameters)
  );

})(window, navigator, window.navigator);

`)

	page.Navigate("https://intoli.com/blog/not-possible-to-block-chrome-headless/chrome-headless-test.html")

	wait := page.WaitRequestIdle()
	wait()

	page.Screenshot("chrome-headless-test.jpg")

}
