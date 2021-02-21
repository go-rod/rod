# Overview

[![Go Reference](https://pkg.go.dev/badge/github.com/go-rod/rod.svg)](https://pkg.go.dev/github.com/go-rod/rod)
[![Discord Chat](https://img.shields.io/discord/719933559456006165.svg)][discord room]

## [Documentation](https://go-rod.github.io/) | [API reference](https://pkg.go.dev/github.com/go-rod/rod?tab=doc)

Rod is a high-level driver directly based on [DevTools Protocol](https://chromedevtools.github.io/devtools-protocol).
It's designed for web automation and scraping.
Rod is designed for both high-level and low-level use, senior programmers can use the low-level packages and functions to easily
customize or build up their own version of Rod, the high-level functions are just examples to build a default version of Rod.

## Features

- Chained context design, intuitive to timeout or cancel the long-running task
- Auto-wait elements to be ready
- Debugging friendly, auto input tracing, remote monitoring headless browser
- Thread-safe for all operations
- Automatically find or download [browser](lib/launcher)
- Lightweight, no third-party dependencies, [CI](https://github.com/go-rod/rod/actions) tested on Linux, Mac, and Windows
- High-level helpers like WaitStable, WaitRequestIdle, HijackRequests, WaitDownload, etc
- Two-step WaitEvent design, never miss an event ([how it works](https://github.com/ysmood/goob))
- Correctly handles nested iframes or shadow DOMs
- No zombie browser process after the crash ([how it works](https://github.com/ysmood/leakless))

## Examples

Please check the [examples_test.go](examples_test.go) file first, then check the [examples](lib/examples) folder.

For more detailed examples, please search the unit tests.
Such as the usage of method `HandleAuth`, you can search all the `*_test.go` files that contain `HandleAuth`,
for example, use Github online [search in repository](https://github.com/go-rod/rod/search?q=HandleAuth&unscoped_q=HandleAuth).
You can also search the GitHub issues, they contain a lot of usage examples too.

[Here](lib/examples/compare-chromedp) is a comparison of the examples between rod and Chromedp.

If you have questions, please raise an issue or join the [chat room][discord room].

## How it works

Here's the common start process of rod:

1. Try to connect to a Devtools endpoint (WebSocket), if not found try to launch a local browser, if still not found try to download one, then connect again. The lib to handle it is [launcher](lib/launcher).

1. Use the JSON-RPC to talk to the Devtools endpoint to control the browser. The lib handles it is [cdp](lib/cdp).

1. Use the type definitions of the JSON-RPC to perform high-level actions. The lib handles it is [proto](lib/proto).

Object model:

![object model](fixtures/object-model.svg)

## Become a maintainer

Please check this [doc](.github/CONTRIBUTING.md).

[discord room]: https://discord.gg/CpevuvY
