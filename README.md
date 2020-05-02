# Overview

[![GoDoc](https://godoc.org/github.com/ysmood/rod?status.svg)](https://pkg.go.dev/github.com/ysmood/rod?tab=doc)
[![codecov](https://codecov.io/gh/ysmood/rod/branch/master/graph/badge.svg)](https://codecov.io/gh/ysmood/rod)
[![goreport](https://goreportcard.com/badge/github.com/ysmood/rod)](https://goreportcard.com/report/github.com/ysmood/rod)

Rod is a High-level Chrome Devtools driver directly based on [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).
It's designed for web automation and scraping. Rod also tries to expose low-level interfaces to users, so that whenever a function is missing users can easily send control requests to the browser directly.

## Features

- Fluent interface design to reduce verbose code
- Chained context design, intuitive to timeout or cancel the long-running task
- Debugging friendly, auto input tracing, remote monitoring headless browser
- Automatically find or download [chrome](lib/launcher)
- No external dependencies, [CI](https://github.com/ysmood/rod/actions) tested on Linux, Mac, and Windows
- High-level helpers like WaitStable, WaitRequestIdle, GetDownloadFile, Resource
- Two-step WaitEvent design, never miss an event
- Correctly handles nested iframes
- No zombie chrome process after the crash ([how it works](https://github.com/ysmood/leakless))

## Examples

You can find examples from [here](examples_test.go) or [here](lib/examples).
For more details, please read the unit tests.

## How it works

Here's the common start process of Rod:

1. Try to connect to a Chrome Devtools endpoint, if not found try to launch a local browser, if still not found try to download one, then connect again. The lib to handle it is [here](lib/launcher).

1. Use the JSON-RPC protocol to talk to the browser endpoint to control it. The lib to handle it is  [here](lib/cdp).

1. To control a specific page, Rod will first inject a js helper script to it. Rod uses it to query and manipulate the page content. The js lib is [here](lib/assets).

## FAQ

### How to use Rod inside a docker container

To let rod work with docker is very easy:

1. Run the Rod image `docker run -p 9222:9222 ysmood/rod`

2. Open another terminal and run a go program like this [example](lib/examples/remote-launch/main.go)

The [Rod image](https://hub.docker.com/repository/docker/ysmood/rod)
can dynamically launch a chrome for each remote driver with customizable chrome flags.
It's [tuned](lib/docker/Dockerfile) for screenshots and fonts for popular languages.

### Does it support other browsers like Firefox or Edge

Rod should work with any browser that supports [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).
For now, Firefox is [supporting](https://wiki.mozilla.org/Remote) this protocol, and Edge will adopt chromium as their backend, so it seems like most major browsers will support it in the future except for Safari.

### Why is it called Rod

Rod is related to puppetry, see [Rod Puppet](https://en.wikipedia.org/wiki/Puppet#Rod_puppet).
So we are the puppeteer, Chrome is the puppet, we use the rod to control the puppet.
So in this sense, `puppeteer.js` sounds strange, we are controlling a puppeteer?

### How to contribute

Your help is more than welcome! Even just open an issue to ask a question may greatly help others.

You might want to learn the basics of [Go Testing](https://golang.org/pkg/testing), [Sub-tests](https://golang.org/pkg/testing), and [Test Suite](https://github.com/stretchr/testify#suite-package) first.

You can get started by reading the unit tests by their nature hierarchy: `Browser -> Page -> Element`.
So you read order will be something like `browser_test.go -> page_test.go -> element_test.go`.
The test is intentionally being designed to be easily understandable.

Here an example to run a single test case: `go test -v -run Test/TestClick`, `TestClick` is the function name you want to run.

We trade off code lines to reduce function call distance to the source code of Golang itself.
You may see redundant code everywhere to reduce the use of interfaces or dynamic tricks.
So that everything should map to your brain like a tree, not a graph.
So that you can always jump from one definition to another in a uni-directional manner, the reverse search should be rare.

### Why another puppeteer like lib

There are a lot of great projects, but no one is perfect, choose the best one that fits your needs is important.

- [selenium](https://www.selenium.dev/)

  It's slower by design because it encourages the use of hard-coded sleep. When work with Rod, you generally don't use sleep at all.
  Therefore it's more buggy to use selenium if the network is unstable.
  It's harder to setup and maintain because of extra dependencies like a browser driver.

- [puppeteer](https://github.com/puppeteer/puppeteer)

  With Puppeteer, you have to handle promise/async/await a lot. It requires a deep understanding of how promises works which are usually painful for QA to write automation tests. End to end tests usually requires a lot of sync operations to simulate human inputs, because Puppeteer is based on Nodejs all control signals it sends to chrome will be async calls, so it's unfriendly for QA from the beginning.

- [chromedp](https://github.com/chromedp/chromedp)

  With Chromedp, you have to use their verbose DSL like tasks to handle the main logic, because Chromedp uses several wrappers to handle execution with context and options which makes it very hard to understand their code when bugs happen. The DSL like wrapper also make the Go type useless when tracking issues.

  It's painful to use Chromedp to deal with iframes, this [ticket](https://github.com/chromedp/chromedp/issues/72) is still open after years.

  When a crash happens, Chromedp will leave the zombie chrome process on Windows and Mac.

- [cypress](https://www.cypress.io/)

  Cypress is very limited, for closed shadow dom or cross-domain iframes it's almost unusable. Read their [limitation doc](https://docs.cypress.io/guides/references/trade-offs.html) for more details.
