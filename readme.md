# Overview

[![GoDoc](https://godoc.org/github.com/ysmood/rod?status.svg)](https://pkg.go.dev/github.com/ysmood/rod?tab=doc)
[![codecov](https://codecov.io/gh/ysmood/rod/branch/master/graph/badge.svg)](https://codecov.io/gh/ysmood/rod)
[![goreport](https://goreportcard.com/badge/github.com/ysmood/rod)](https://goreportcard.com/report/github.com/ysmood/rod)

Rod is a High-level chrome devtools controller directly based on [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/).

Rod also tries to expose low-level interfaces to users, so that whenever a function is missing users can easily send control requests to the browser directly ([example](https://github.com/ysmood/rod/blob/c788570429a63fb76933e109470928add504adad/examples_test.go#L141)).

## Features

- Fluent interface design to reduce verbose code
- Chained context design, intuitive to timeout or cancel the long-running task
- Debugging friendly, auto input tracing, and [screenshots](https://youtu.be/JJlPNU9n_gU)
- Automatically find or download [chrome](lib/launcher)
- No external dependencies, [CI](https://github.com/ysmood/rod/actions) tested on Linux, Mac, and Windows
- High-level helpers like WaitStable, WaitRequestIdle, GetDownloadFile, Resource
- Two-step WaitEvent design, never miss an event
- Correctly handles nested iframes
- No zombie chrome process after the crash ([how it works](https://github.com/ysmood/leakless))

## Examples

[Basic examples](./examples_test.go)

For detailed examples, please read the unit tests.

Here's the example to use Rod with docker, we choose [zenika/alpine-chrome](https://github.com/Zenika/alpine-chrome)
as an example:

```bash
# git clone and cd to the root dir of rod repo
# build the test binary for linux, it will generate the rod.test executable file
GOOS=linux go test -c

# nothing special, just run the binary inside container
docker run --rm --cap-add=SYS_ADMIN --entrypoint='' \
    -v $(pwd):/rod -w=/rod \
    zenika/alpine-chrome \
    ./rod.test -test.v
```

## Development

See the Github Actions CI config.

## FAQ

> Why another puppeteer like lib?

There are a lot of great projects, but no one is perfect, choose the best one that fits your needs is important.

### [puppeteer](https://github.com/puppeteer/puppeteer)

With puppeteer, you have to handle promise/async/await a lot. It requires a deep understanding of how promises works which are usually painful
for QA to write automation tests. End to end tests usually requires a lot of sync operations to simulate human inputs, because puppeteer is based on
Nodejs all control signals it sends to chrome will be async calls, so it's unfriendly for QA from the beginning.

### [chromedp](https://github.com/chromedp/chromedp)

With chromedp, you have to use their verbose DSL like tasks to handle the main logic and it's painful to deal with iframes.
Because chromedp uses several wrappers to handle execution with context and options which makes it very hard to understand their code when bugs happen.
When a crash happens, chromedp will leave the zombie chrome process on Windows and Mac.

### [cypress](https://www.cypress.io/)

Cypress is very limited, to test cross-domain iframes such as payments,
it's almost unusable. Read their [limitation doc](https://docs.cypress.io/guides/references/trade-offs.html).
