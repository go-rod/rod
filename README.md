# Overview

[![Go Reference](https://pkg.go.dev/badge/github.com/go-rod/rod.svg)](https://pkg.go.dev/github.com/go-rod/rod)
[![Discord Chat](https://img.shields.io/discord/719933559456006165.svg)][discord room]

## [Documentation](https://go-rod.github.io/) | [API reference](https://pkg.go.dev/github.com/go-rod/rod?tab=doc) | [FAQ](https://go-rod.github.io/#/faq/README)

Rod is a high-level driver directly based on [DevTools Protocol](https://chromedevtools.github.io/devtools-protocol).
It's designed for web automation and scraping for both high-level and low-level use, senior developers can use the low-level packages and functions to easily
customize or build up their own version of Rod, the high-level functions are just examples to build a default version of Rod.

[中文 API 文档](https://pkg.go.dev/github.com/go-rod/go-rod-chinese)

## Features

- Chained context design, intuitive to timeout or cancel the long-running task
- Auto-wait elements to be ready
- Debugging friendly, auto input tracing, remote monitoring headless browser
- Thread-safe for all operations
- Automatically find or download [browser](lib/launcher)
- High-level helpers like WaitStable, WaitRequestIdle, HijackRequests, WaitDownload, etc
- Two-step WaitEvent design, never miss an event ([how it works](https://github.com/ysmood/goob))
- Correctly handles nested iframes or shadow DOMs
- No zombie browser process after the crash ([how it works](https://github.com/ysmood/leakless))
- [CI](https://github.com/go-rod/rod/actions) enforced 100% test coverage

## Examples

Please check the [examples_test.go](examples_test.go) file first, then check the [examples](lib/examples) folder.

For more detailed examples, please search the unit tests.
Such as the usage of method `HandleAuth`, you can search all the `*_test.go` files that contain `HandleAuth`,
for example, use Github online [search in repository](https://github.com/go-rod/rod/search?q=HandleAuth&unscoped_q=HandleAuth).
You can also search the GitHub [issues](https://github.com/go-rod/rod/issues) or [discussions](https://github.com/go-rod/rod/discussions),
a lot of usage examples are recorded there.

[Here](lib/examples/compare-chromedp) is a comparison of the examples between rod and Chromedp.

If you have questions, please raise an [issues](https://github.com/go-rod/rod/issues)/[discussions](https://github.com/go-rod/rod/discussions) or join the [chat room][discord room].

## Join us

Your help is more than welcome! Even just open an issue to ask a question may greatly help others.

Please read [How To Ask Questions The Smart Way](http://www.catb.org/~esr/faqs/smart-questions.html) before you ask questions.

We use Github Projects to manage tasks, you can see the priority and progress of the issues [here](https://github.com/go-rod/rod/projects).

If you want to contribute please read the [Contributor Guide](.github/CONTRIBUTING.md).

[discord room]: https://discord.gg/CpevuvY
