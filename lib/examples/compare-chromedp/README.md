# Rod comparison with chromedp

chromedp is one of the most popular drivers for Devtools Protocol.

To help developers who are familiar with chromedp to understand rod better we created side by side examples between rod and chromedp.

To run an example:

1. clone rod
2. cd to the folder of an example, such as `cd lib/examples/compare-chromedp/click`
3. run `go run .`

| rod                                | chromedp                                                                          | Description                                                                |
| ---------------------------------- | --------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| [click](./click)                   | [click](https://github.com/chromedp/examples/blob/master/click)                   | use a selector to click on an element                                      |
| [cookie](./cookie)                 | [cookie](https://github.com/chromedp/examples/blob/master/cookie)                 | set a HTTP cookie on requests                                              |
| [download_file](./download_file)   | [download_file](https://github.com/chromedp/examples/tree/master/download_file)   | do headless file downloads                                                 |
| [download_image](./download_image) | [download_image](https://github.com/chromedp/examples/tree/master/download_image) | do headless image downloads                                                |
| [emulate](./emulate)               | [emulate](https://github.com/chromedp/examples/blob/master/emulate)               | emulate a specific device such as an iPhone                                |
| [eval](./eval)                     | [eval](https://github.com/chromedp/examples/blob/master/eval)                     | evaluate javascript and retrieve the result                                |
| [headers](./headers)               | [headers](https://github.com/chromedp/examples/blob/master/headers)               | set a HTTP header on requests                                              |
| [keys](./keys)                     | [keys](https://github.com/chromedp/examples/blob/master/keys)                     | send key events to an element                                              |
| [logic](./logic)                   | [logic](https://github.com/chromedp/examples/blob/master/logic)                   | more complex logic beyond simple actions                                   |
| [pdf](./pdf)                       | [pdf](https://github.com/chromedp/examples/tree/master/pdf)                       | capture a pdf of a page                                                    |
| [proxy](./proxy)                   | [proxy](https://github.com/chromedp/examples/tree/master/proxy)                   | authenticate a proxy server which requires authentication                  |
| [remote](./remote)                 | [remote](https://github.com/chromedp/examples/blob/master/remote)                 | connect to an existing DevTools instance using a remote WebSocket URL      |
| [screenshot](./screenshot)         | [screenshot](https://github.com/chromedp/examples/blob/master/screenshot)         | take a screenshot of a specific element and of the entire browser viewport |
| [submit](./submit)                 | [submit](https://github.com/chromedp/examples/blob/master/submit)                 | fill out and submit a form                                                 |
| [subtree](./subtree)               | [subtree](https://github.com/chromedp/examples/tree/master/subtree)               | populate and travel a subtree of the DOM                                   |
| [text](./text)                     | [text](https://github.com/chromedp/examples/blob/master/text)                     | extract text from a specific element                                       |
| [upload](./upload)                 | [upload](https://github.com/chromedp/examples/blob/master/upload)                 | upload a file on a form                                                    |
| [visible](./visible)               | [visible](https://github.com/chromedp/examples/blob/master/visible)               | wait until an element is visible                                           |

Occasionally, some of these examples may break if the specific websites these examples use get updated.
We suggest you create an [issue](https://github.com/go-rod/rod/issues/new/choose).
