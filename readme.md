# Overview

Rod is a High-level chrome devtools controller.

This lib is not designed for performance, if you want performance use the [cdp](lib/cdp) lib directly.

For example, compared with [puppeteer](https://github.com/puppeteer/puppeteer) or [chromedp](https://github.com/chromedp/chromedp),
they don't have high-level abstraction for iframes, it's a pain to use them to deal contents inside complex iframes.
Besides, it's pretty verbose to use them, with puppeteer you have to type async/await a lot, with chromedp it's even more verbose.
Rod also tries not to hide low-level interface from user, so that whenever there's a missing functionality from the high-level interface
you can easily send control request to browser directly.

## Examples

[Basic examples](./examples_test.go)

For detailed examples, please read the unit tests.
