# Rod comparison with Puppeteer(in updating)

Puppeteer is a Node library which provides a high-level API to control Chrome or Chromium over the DevTools Protocol.

To help developers who are familiar with puppeteer to understand rod better we created side by side examples between rod and chromedp.

To run an example:

1. clone rod
2. cd to the folder of an example, such as `cd lib/examples/compare-puppeteer/pdf`
3. run `go run .`

| rod                        | Puppeteer                                                                 | Description                                                                |
| -------------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| [pdf](./pdf)               | [pdf](https://github.com/puppeteer/puppeteer/blob/main/examples/pdf.js)   | save webpage to pdf                                                        |


Occasionally, some of these examples may break if the specific websites these examples use get updated.
We suggest you create an [issue](https://github.com/go-rod/rod/issues/new/choose).
