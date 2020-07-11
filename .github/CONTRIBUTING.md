# Contributing

Your help is more than welcome! Even just open an issue to ask a question may greatly help others.

We use [Github Projects](https://github.com/go-rod/rod/projects/1) to manage tasks. You can see the priority and progress of the issues there.

You might want to learn the basics of [Go Testing](https://golang.org/pkg/testing), [Sub-tests](https://golang.org/pkg/testing), and [Test Suite](https://github.com/stretchr/testify#suite-package) first.

You can get started by reading the unit tests by their nature hierarchy: `Browser -> Page -> Element`.
So your reading order will be something like `browser_test.go -> page_test.go -> element_test.go`.
The test is intentionally being designed to be easily understandable.

Here's an example to run a single test case: `go test -v -run Test/TestClick`, `TestClick` is the function name you want to run.

We trade off code lines to reduce function call distance to the source code of Golang itself.
You may see redundant code everywhere to reduce the use of interfaces or dynamic tricks.
So that everything should map to your brain like a tree, not a graph.
So that you can always jump from one definition to another in a uni-directional manner, the reverse search should be rare.

## E suffix

If you read the function list, you will notice a lot of functions have two versions, such as `Screenshot` and `ScreenshotE`,
the `E` suffix means error. Functions end with E suffix will return an error as the last value. The non-E version is usually a wrapper
for the E version with fixed default options to make it easier to use, and it will panic if the error is not nil other than return the error.
Usually, The E function is a low-level version of the non-E functions with more options.

For example the source code of `Element.Click` and `Element.ClickE`. `Click` has no argument.
But with `ClickE` you can pass `button` argument to it to decide which button to click.
`Click` calls `ClickE` inside it and passes left-button to it.

All the non-E version functions should be inside the sugar files (files end with sugar.go). Since even we remove all those high-level functions the low-level version of them can still do the job, users just need to type a little bit more code.

When you adding a new function to this lib, please make two versions of it if possible.

The reason to have redundant functions is because of Golang is lacking the support of generics.
To support the fluent API design, we trade more code to make this lib more user friendly.

## Run tests

The entry point of all tests is the `setup_test.go` file.

### Example to run a single test

`go test -v -run Test/Click`, `Click` is the pattern to match the test function name.

### To disable headless mode

`rod=show go test -v -run Test/Click`.

### To lint the project

You can use [godev](https://github.com/ysmood/kit#godev) to lint the project on your local:

`godev lint`

We also use it in the CI to enforce the minimum test coverage.

### To run inside docker

1. `docker build -t rod -f lib/docker/test.Dockerfile .`

2. `docker run --name rod -itp 9273:9273 -v $(pwd):/t -w /t rod sh`

3. `rod=monitor,blind go test -v -run Test/Click`

4. visit `http://[::]:9273` to monitor the tests

After you exit the container, you can reuse it with `docker start -i rod`.

### Convention of the git commit message

The commit message follows the rules [here](https://github.com/torvalds/subsurface-for-dirk/blame/a48494d2fbed58c751e9b7e8fbff88582f9b2d02/README#L88). We don't use rules like [Conventional Commits](https://www.conventionalcommits.org/) because it's hard for beginners to write correct commit messages. It will encourage reviewers to spend more time on high-level problems, not the details. We also want to reduce the overhead when reading the git-blame, for example, `fix: correct minor typos in code` is the same as `fix minor typos in code`, there's no need to repeat content in the title line.

## Become a maintainer

Since this is a small project, we will use a very simple model to promote contributors to maintainers.
Only the first 2 maintainers will grant permission from me, then we will start to elect
new contributors by voting in the public issue. If no one votes down and 2/3 votes up then one election will pass.
