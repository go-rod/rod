# Rod comparison with chromedp

chromedp is one of the most popular libraries available for Go which controls the Chrome Devtools Protocol. 

We have emulated the examples they provide using our own library so you can compare which one you prefer.

Occasionally, some of these examples may break if the Rod API gets updated or if the specific websites these examples use get updated.
We suggest you create an [issue](https://github.com/ysmood/rod/issues/new/choose).

You can build and run these examples in the usual Go way:

```sh
# retrieve examples
$ go get -u -d github.com/ysmood/rod/

# run example <prog>
$ go run $GOPATH/src/github.com/ysmood/rod/lib/examples/compare-chromedp/<prog>/main.go

# build example <prog>
$ go build -o <prog> github.com/ysmood/rod/lib/examples/compare-chromedp/<prog>/ && ./<prog>
```

| Example                   | chromedp Example                                              | Description                                                                  |
|---------------------------|---------------------------------------------------------------|------------------------------------------------------------------------------|
| [click](./click)           | [click](https://github.com/chromedp/examples/blob/master/click)           | use a selector to click on an element                                        |
| [cookie](./cookie)         | [cookie](https://github.com/chromedp/examples/blob/master/cookie)         | set a HTTP cookie on requests                                                |
| [emulate](./emulate)       | [emulate](https://github.com/chromedp/examples/blob/master/emulate)       | emulate a specific device such as an iPhone                                  |
| [eval](./eval)             | [eval](https://github.com/chromedp/examples/blob/master/eval)             | evaluate javascript and retrieve the result                                  |
| [headers](./headers)       | [headers](https://github.com/chromedp/examples/blob/master/headers)       | set a HTTP header on requests                                                |
| [keys](./keys)             | [keys](https://github.com/chromedp/examples/blob/master/keys)             | send key events to an element                                                |
| [logic](./logic)           | [logic](https://github.com/chromedp/examples/blob/master/logic)           | more complex logic beyond simple actions                                     |
| [remote](./remote)         | [remote](https://github.com/chromedp/examples/blob/master/remote)         | connect to an existing Chrome DevTools instance using a remote WebSocket URL |
| [screenshot](./screenshot) | [screenshot](https://github.com/chromedp/examples/blob/master/screenshot) | take a screenshot of a specific element and of the entire browser viewport   |
| [submit](./submit)         | [submit](https://github.com/chromedp/examples/blob/master/submit)         | fill out and submit a form                                                   |
| [text](./text)             | [text](https://github.com/chromedp/examples/blob/master/text)             | extract text from a specific element                                         |
| [upload](./upload)         | [upload](https://github.com/chromedp/examples/blob/master/upload)         | upload a file on a form                                                      |
| [visible](./visible)       | [visible](https://github.com/chromedp/examples/blob/master/visible)       | wait until an element is visible                                             |
