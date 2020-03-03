# Overview

[![GoDoc](https://godoc.org/github.com/ysmood/rod?status.svg)](https://pkg.go.dev/github.com/ysmood/rod?tab=doc)
[![codecov](https://codecov.io/gh/ysmood/rod/branch/master/graph/badge.svg)](https://codecov.io/gh/ysmood/rod)
[![goreport](https://goreportcard.com/badge/github.com/ysmood/rod)](https://goreportcard.com/report/github.com/ysmood/rod)

Rod is a High-level chrome devtools controller directly based on [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).

For example, compared with [puppeteer](https://github.com/puppeteer/puppeteer) or [chromedp](https://github.com/chromedp/chromedp),
it's pretty verbose to use them, with puppeteer you have to handle promise/async/await a lot.
With chromedp it's painful to deal with iframes. When crash happens, chromedp will leave zombie chrome process on Windows and Mac.

Rod also tries to expose low-level interfaces to users, so that whenever a function is missing users can easily send control requests to the browser directly. Here's the example of how to call chrome API directly [lib/cdp](lib/cdp).

## Features

- Fluent interface design to reduce verbose code
- Context first, such as timeout inheritance, cancel long-running task
- Debug friendly, auto input trace, and screenshots
- Automatically find or download chrome
- No external dependencies, CI tested on Linux, Mac, and Windows
- High-level helpers like WaitStable, WaitRequestIdle, GetDownloadFile, Resource
- Two-step WaitEvent design, never miss an event
- Correctly handles nested iframes
- No zombie chrome process after crash ([how it works](https://github.com/ysmood/leakless))

## Examples

[Basic examples](./examples_test.go)

For detailed examples, please read the unit tests.

## Development

See the Github Actions CI config.
