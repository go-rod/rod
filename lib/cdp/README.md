# Overview

This client is directly based on [doc](https://chromedevtools.github.io/devtools-protocol/)

You can treat it as a minimal example of how to use the Chrome DevTools Protocol, no
complex abstraction. The test coverage for this lib is 100%.

The lib is thread-safe, and context first. Chrome already does a good job of API type check,
so this lib won't do it again. The overhead of encoding API will never be the bottleneck as long as you use chrome headless.

For a basic example, check this [file](example_test.go).

For a detailed example, check this [file](main_test.go).
