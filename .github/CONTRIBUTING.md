# Contributing

Your help is more than welcome! Even just open an issue to ask a question may greatly help others.

We use [Github Projects](https://github.com/orgs/go-rod/projects/1) to manage tasks. You can see the priority and progress of the issues there.

You might want to learn the basics of [Go Testing](https://golang.org/pkg/testing), [Sub-tests](https://golang.org/pkg/testing), and [Test Suite](https://github.com/stretchr/testify#suite-package) first.

You can get started by reading the unit tests by their nature hierarchy: `Browser -> Page -> Element`.
So your reading order will be something like `browser_test.go -> page_test.go -> element_test.go`.
The test is intentionally being designed to be easily understandable.

Here's an example to run a single test case: `go test -v -run Test/TestClick`, `TestClick` is the function name you want to run.

We trade off code lines to reduce function call distance to the source code of Golang itself.
You may see redundant code everywhere to reduce the use of interfaces or dynamic tricks. We shall only use interfaces for IO and dependency injection.
So that everything should map to your brain like a tree, not a graph.
So that you can always jump from one definition to another in a uni-directional manner, the reverse search should be rare.

## Run tests

The entry point of all tests is the `setup_test.go` file.

### Example to run a single test

`go test -v -run Test/Click`, `Click` is the pattern to match the test function name.

### To disable headless mode

`rod=show go test -v -run Test/Click`.

### To lint the project

```bash
go run ./lib/utils/lint
```

### Code Coverage

If the code coverage is less than 100%, the CI will fail.

Learn the [basics](https://blog.golang.org/cover) first.

To cover the error branch of the code you can intercept cdp calls.
There are several helper functions in the [setup_test.go](../setup_test.go) for it:

- stubCounter
- stub
- stubErr

### To run inside docker

1. `docker build -t rod -f lib/docker/Dockerfile .`

2. `docker run --name rod -itp 9273:9273 -v $(pwd):/t -w /t rod sh`

3. `rod=monitor,blind go test -v -run Test/Click`

4. visit `http://[::]:9273` to monitor the tests

After you exit the container, you can reuse it with `docker start -i rod`.

### Convention of the git commit message

The commit message follows the rules [here](https://github.com/torvalds/subsurface-for-dirk/blame/a48494d2fbed58c751e9b7e8fbff88582f9b2d02/README#L88). We don't use rules like [Conventional Commits](https://www.conventionalcommits.org/) because it's hard for beginners to write correct commit messages. It will encourage reviewers to spend more time on high-level problems, not the details. We also want to reduce the overhead when reading the git-blame, for example, `fix: correct minor typos in code` is the same as `fix minor typos in code`, there's no need to repeat content in the title line.

## Become a maintainer

At the early stage of this project, we will use a simple model to promote contributors to maintainers.
Anybody who has contributed code or doc to the project can get write access to issues and PRs contributors.
Maintainers will have all the permissions of this project, only the first 2 maintainers are granted by the owner, then we will start to elect
new maintainers by voting in the public issue. If no one votes down and 2/3 votes up then one election will pass.
