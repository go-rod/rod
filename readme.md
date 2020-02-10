# Overview

[![GoDoc](https://godoc.org/github.com/ysmood/rod?status.svg)](http://godoc.org/github.com/ysmood/rod)
[![codecov](https://codecov.io/gh/ysmood/rod/branch/master/graph/badge.svg)](https://codecov.io/gh/ysmood/rod)
[![goreport](https://goreportcard.com/badge/github.com/ysmood/rod)](https://goreportcard.com/report/github.com/ysmood/rod)

Rod is a High-level chrome devtools controller directly based on [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).

For example, compared with [puppeteer](https://github.com/puppeteer/puppeteer) or [chromedp](https://github.com/chromedp/chromedp),
it's pretty verbose to use them, with puppeteer you have to handle promise/async/await a lot, with chromedp it's even more verbose.

Rod also tries to expose low-level interfaces to users, so that whenever a function is missing users can easily send control requests to the browser directly. Here's the example of how to call chrome API directly [lib/cdp](lib/cdp).

## Examples

[Basic examples](./examples_test.go)

For detailed examples, please read the unit tests.

## Development

See the Github Actions CI config.
