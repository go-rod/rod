# Overview

[![GoDoc](https://godoc.org/github.com/ysmood/rod?status.svg)](https://pkg.go.dev/github.com/ysmood/rod?tab=doc)
[![codecov](https://codecov.io/gh/ysmood/rod/branch/master/graph/badge.svg)](https://codecov.io/gh/ysmood/rod)
[![goreport](https://goreportcard.com/badge/github.com/ysmood/rod)](https://goreportcard.com/report/github.com/ysmood/rod)
[![Gitter](https://badges.gitter.im/ysmood-rod/community.svg)](https://gitter.im/ysmood-rod/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

![logo](fixtures/rod.png)

test

Rod is a High-level Chrome Devtools driver directly based on [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).
It's designed for web automation and scraping. Rod also tries to expose low-level interfaces to users, so that whenever a function is missing users can easily send control requests to the browser directly.

## Features

- Fluent interface design to reduce verbose code
- Chained context design, intuitive to timeout or cancel the long-running task
- Debugging friendly, auto input tracing, remote monitoring headless browser
- Automatically find or download [browser](lib/launcher)
- No external dependencies, [CI](https://github.com/ysmood/rod/actions) tested on Linux, Mac, and Windows
- High-level helpers like WaitStable, WaitRequestIdle, GetDownloadFile, Resource
- Two-step WaitEvent design, never miss an event
- Correctly handles nested iframes
- No zombie chrome process after the crash ([how it works](https://github.com/ysmood/leakless))

## Examples

You can find examples from [here](examples_test.go) or [here](lib/examples).

For more detailed examples, please search the unit tests.
Such as the usage of method `HandleAuth`, search the all the `*_test.go` files that contain `HandleAuth` or `HandleAuthE`.
You can also search the github issues, they contain a lot of usage examples too.

If you have questions, please raise an issue or join the [gitter room](https://gitter.im/ysmood-rod/community?utm_source=share-link&utm_medium=link&utm_campaign=share-link).

## How it works

Here's the common start process of Rod:

1. Try to connect to a Chrome Devtools endpoint, if not found try to launch a local browser, if still not found try to download one, then connect again. The lib to handle it is [here](lib/launcher).

1. Use the JSON-RPC protocol to talk to the browser endpoint to control it. The lib to handle it is  [here](lib/cdp).

1. To control a specific page, Rod will first inject a js helper script to it. Rod uses it to query and manipulate the page content. The js lib is [here](lib/assets).

## FAQ

### Q: How to use Rod with docker

To let rod work with docker is very easy:

1. Run the Rod image `docker run -p 9222:9222 ysmood/rod`

2. Open another terminal and run a go program like this [example](lib/examples/remote-launch/main.go)

The [Rod image](https://hub.docker.com/repository/docker/ysmood/rod)
can dynamically launch a chrome for each remote driver with customizable chrome flags.
It's [tuned](lib/docker/Dockerfile) for screenshots and fonts among popular natural languages.
You can easily load balance requests to the cluster of this image, each container can create multiple browser instances at the same time.

### Q: Does it support other browsers like Firefox or Edge

Rod should work with any browser that supports [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).
For now, Firefox is [supporting](https://wiki.mozilla.org/Remote) this protocol, and Edge will adopt chromium as their backend, so it seems like most major browsers will support it in the future except for Safari.

### Q: Why is it called Rod

Rod is related to puppetry, see [Rod Puppet](https://en.wikipedia.org/wiki/Puppet#Rod_puppet).
So we are the puppeteer, Chrome is the puppet, we use the rod to control the puppet.
So in this sense, `puppeteer.js` sounds strange, we are controlling a puppeteer?

### Q: How to contribute

Please check this [file](.github/CONTRIBUTING.md).

### Q: How versioning is handled

[Semver](https://semver.org/) is used.

Before `v1.0.0` whenever the second section changed, such as `v0.1.0` to `v0.2.0`, there must be some public API changes, such as changes of function names or parameter types. If only the last section changed, no public API will be changed.

### Q: Why another puppeteer like lib

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
