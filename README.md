# Overview

[![GoDoc](https://godoc.org/github.com/go-rod/rod?status.svg)](https://pkg.go.dev/github.com/go-rod/rod?tab=doc)
[![codecov](https://codecov.io/gh/go-rod/rod/branch/master/graph/badge.svg)](https://codecov.io/gh/go-rod/rod)
[![goreport](https://goreportcard.com/badge/github.com/go-rod/rod)](https://goreportcard.com/report/github.com/go-rod/rod)
[![Discord Chat](https://img.shields.io/discord/719933559456006165.svg)][discord room]

Rod is a High-level Devtools driver directly based on [DevTools Protocol][devtools protocol].
It's designed for web automation and scraping. rod also tries to expose low-level interfaces to users, so that whenever a function is missing users can easily send control requests to the browser directly.

## Features

- Chained context design, intuitive to timeout or cancel the long-running task
- Debugging friendly, auto input tracing, remote monitoring headless browser
- Thread-safe for all operations
- Automatically find or download [browser](lib/launcher)
- No external dependencies, [CI](https://github.com/go-rod/rod/actions) tested on Linux, Mac, and Windows
- High-level helpers like WaitStable, WaitRequestIdle, HijackRequests, GetDownloadFile, etc
- Two-step WaitEvent design, never miss an event ([how it works](https://github.com/ysmood/goob))
- Correctly handles nested iframes
- No zombie browser process after the crash ([how it works](https://github.com/ysmood/leakless))

## Examples

Please check the [examples_test.go](examples_test.go) file first, then check the [examples](lib/examples) folder.

For more detailed examples, please search the unit tests.
Such as the usage of method `HandleAuth`, you can search all the `*_test.go` files that contain `HandleAuth` or `HandleAuthE`,
for example, use Github online [search in repository](https://github.com/go-rod/rod/search?q=HandleAuth&unscoped_q=HandleAuth).
You can also search the GitHub issues, they contain a lot of usage examples too.

[Here](lib/examples/compare-chromedp) is a comparison of the examples between rod and [chromedp][chromedp].

If you have questions, please raise an issue or join the [chat room][discord room].

## How it works

Here's the common start process of rod:

1. Try to connect to a Devtools endpoint (WebSocket), if not found try to launch a local browser, if still not found try to download one, then connect again. The lib to handle it is [launcher](lib/launcher).

1. Use the JSON-RPC to talk to the Devtools endpoint to control the browser. The lib handles it is [cdp](lib/cdp).

1. The type definitions of the JSON-RPC are in lib [proto](lib/proto).

1. To control a specific page, rod will first inject a js helper script to it. rod uses it to query and manipulate the page content. The js lib is in [assets](lib/assets).

Object model:

![object model](fixtures/object-model.svg)

## API Documentation

We use the standard way to doc the project: [Go Doc](https://pkg.go.dev/github.com/go-rod/rod?tab=doc)

## FAQ

### Q: How to use rod with docker so that I don't have to install a browser

To let rod work with docker is very easy:

1. Run the rod image `docker run -p 9222:9222 rodorg/rod`

2. Open another terminal and run a go program like this [example](lib/examples/remote-launch/main.go)

The [rod image](https://hub.docker.com/repository/docker/rodorg/rod)
can dynamically launch a browser for each remote driver with customizable browser flags.
It's [tuned](lib/docker/Dockerfile) for screenshots and fonts among popular natural languages.
You can easily load balance requests to the cluster of this image, each container can create multiple browser instances at the same time.

### Q: Why there is always an "about:blank" page

It's an issue of the browser itself. If we enable the `--no-first-run` flag and we don't create a blank page, it will create a hello page which will consume more power.

### Q: Does it support other browsers like Firefox or Edge

Rod should work with any browser that supports [DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).

- Microsoft Edge can pass all the unit tests.
- Firefox is [supporting](https://wiki.mozilla.org/Remote) this protocol.
- Safari doesn't have any plan to support it yet.
- IE won't support it.

### Q: Why is it called rod

Rod is related to puppetry, see [rod Puppet](https://en.wikipedia.org/wiki/Puppet#rod_puppet).
So we are the puppeteer, the browser is the puppet, we use the rod to control the puppet.
So in this sense, `puppeteer.js` sounds strange, we are controlling a puppeteer?

### Q: How to contribute

Please check this [doc](.github/CONTRIBUTING.md).

### Q: How versioning is handled

[Semver](https://semver.org/) is used.

Before `v1.0.0` whenever the second section changed, such as `v0.1.0` to `v0.2.0`, there must be some public API changes, such as changes of function names or parameter types. If only the last section changed, no public API will be changed.

You can use the Github's release comparison to see the automated changelog, for example, [compare v0.44.2 with v0.44.0](https://github.com/go-rod/rod/compare/v0.44.0...v0.44.2).

### Q: Why another puppeteer like lib

There are a lot of great projects, but no one is perfect, choose the best one that fits your needs is important.

- [chromedp][chromedp]

  Theoretically, rod should perform faster and consume less memory than chromedp.

  Chromedp uses a [fix-sized buffer](https://github.com/chromedp/chromedp/blob/b56cd66/target.go#L69-L73) for events, it can cause dead-lock on high concurrency. Because chromedp uses single-event-loop, the slow event handlers will block each other. Rod doesn't have these issues because it's based on [goob](https://github.com/ysmood/goob).

  Chromedp will JSON decode every message from browser, rod is decode-on-demand, so rod performs better, especially for heavy network events.

  Chromedp uses third part WebSocket lib which has [1MB overhead](https://github.com/chromedp/chromedp/blob/b56cd66f9cebd6a1fa1283847bbf507409d48225/conn.go#L43-L54) for each cdp client, if you want to control thousands of remote browsers it can become a problem. Because of this limitation, if you evaluate a js script larger than 1MB, chromedp will crash.

  When a crash happens, chromedp will leave the zombie browser process on Windows and Mac.

  Rod is more configurable, such as you can even replace the WebSocket lib with the lib you like.

  For direct code comparison you can check [here](lib/examples/compare-chromedp). If you compare the example called `logic` between [rod](lib/examples/compare-chromedp/logic/main.go) and [chromedp](https://github.com/chromedp/examples/blob/master/logic/main.go), you will find out how much simpler rod is.

  With chromedp, you have to use their verbose DSL like tasks to handle the main logic, because chromedp uses several wrappers to handle execution with context and options which makes it very hard to understand their code when bugs happen. The heavily used interfaces make the static types useless when tracking issues. In contrast, rod uses as few interfaces as possible.

  Rod has less dependencies, simpler code structure, and better test coverage, you should find it's easier to contribute code to rod. Therefore compared with chromedp, rod has the potential to have more nice functions from the community in the future.

  Another problem of chromedp is their architecture is based on [DOM node id](https://chromedevtools.github.io/devtools-protocol/tot/DOM/#type-NodeId), but puppeteer and rod are based on [remote object id](https://chromedevtools.github.io/devtools-protocol/tot/Runtime/#type-RemoteObjectId). In consequence, it will prevent chromedp's maintainers from adding high-level functions that are coupled with runtime. For example, this [ticket](https://github.com/chromedp/chromedp/issues/72) had opened for 3 years. Even after it's closed, you still can't evaluate js express on the element inside an iframe.

- [puppeteer][puppeteer]

  Puppeteer can't guarantee event order, rod guarantees all event order.

  Puppeteer will JSON decode every message from browser, rod is decode-on-demand, so rod performs better, especially for heavy network events.

  With puppeteer, you have to handle promise/async/await a lot. End to end tests requires a lot of sync operations to simulate human inputs, because Puppeteer is based on Nodejs all IO operations are async calls, so usually, people end up typing tons of async/await. The overhead grows when your project grows.

  Rod is type-safe by default. It has type bindings with all the API of Devtools protocol.

  Rod will disable domain events whenever possible, puppeteer will always enable all the domains. It will consume a lot of resources when driving a remote browser.

  Rod supports cancellation and timeout better. For example, to simulate `click` we have to send serval cdp requests, with [Promise](https://stackoverflow.com/questions/29478751/cancel-a-vanilla-ecmascript-6-promise-chain) you can't achieve something like "only send half of the cdp requests", but with the [context](https://golang.org/pkg/context/) we can.

- [selenium](https://www.selenium.dev/)

  Selenium is based on [webdriver protocol](https://www.w3.org/TR/webdriver/) which has much less functions compare to [devtools protocol][devtools protocol]. Such as it can't handle [closed shadow DOM](https://github.com/sukgu/shadow-automation-selenium/issues/7#issuecomment-563062460). No way to save page as PDF. No support for tools like [Profiler](https://chromedevtools.github.io/devtools-protocol/tot/Profiler/) or [Performance](https://chromedevtools.github.io/devtools-protocol/tot/Performance/), etc.

  Harder to set up and maintain because of extra dependencies like a browser driver.

  Though selenium sells itself for better cross-browser support, it's usually very hard to make it work for all major browsers.

  There are plenty of articles about "selenium vs puppeteer", you can treat rod as the Golang version of puppeteer.

- [cypress](https://www.cypress.io/)

  Cypress is very limited, for closed shadow dom or cross-domain iframes it's almost unusable. Read their [limitation doc](https://docs.cypress.io/guides/references/trade-offs.html) for more details.

  If you want to cooperate with us to create a testing focused framework base on rod to overcome the limitation of cypress, please contact us.

[devtools protocol]: https://chromedevtools.github.io/devtools-protocol
[chromedp]: https://github.com/chromedp/chromedp
[puppeteer]: https://github.com/puppeteer/puppeteer
[discord room]: https://discord.gg/CpevuvY
