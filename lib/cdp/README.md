# Overview

This client is directly based on [doc](https://chromedevtools.github.io/devtools-protocol/)

You can treat it as a minimal example of how to use the DevTools Protocol, no complex abstraction.

The lib is thread-safe, and context first. The overhead of encoding API will never be the bottleneck as long as you use headless browser.

For a basic example, check this [file](example_test.go).

For a detailed example, check this [file](main_test.go).
