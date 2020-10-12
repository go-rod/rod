# Contributing

Your help is more than welcome! Even just open an issue to ask a question may greatly help others.

We use Github Projects to manage tasks, you can see the priority and progress of the issues [here](https://github.com/orgs/go-rod/projects/1).

## Terminology

When we talk about type in the doc we use [gopls](https://github.com/golang/tools/tree/master/gopls) symbol query syntax. For example, when we say `rod.Page.PDF`, you can run:

```bash
gopls workspace_symbol -matcher=fuzzy rod.Page.PDF$
```

to locate the file and line of it.

## Run tests

No magic, just `go test`.

We use type `rod_test.T` to hold all the tests.
For more details check doc [here](https://github.com/ysmood/got).

Use regex to match and run a single test: `go test -v -run /^Click$`.

### Disable headless mode

```bash
rod=show,trace,slow=2s go test -v -run /Click
```

Check type `defaults.ResetWithEnv` for how it works.

### Lint project

```bash
go generate # only required for first time
go run ./lib/utils/lint
```

### Code Coverage

If the code coverage is less than 100%, the CI will fail.

Learn the [basics](https://blog.golang.org/cover) first.

To cover the error branch of the code we usually intercept cdp calls.
There are several helper functions for it:

- `rod_test.MockClient.stubCounter`
- `rod_test.MockClient.stub`
- `rod_test.MockClient.stubErr`

### To run inside docker

1. `docker build -t rod -f lib/docker/test.Dockerfile .`

1. `docker volume create rod`

1. `docker run --rm -v rod:/root -v $(pwd):/t rod go test -v -run /Click`

### Detect goroutine leak

Because parallel execution will pollution the global goroutine stack. By default, the goroutine leak detection for each test will be disabled, but the detection for the whole test program will still work as well. To enable detection for each test, just let the `go test -parallel=1`.

## Convention of the git commit message

The commit message follows the rules [here](https://github.com/torvalds/subsurface-for-dirk/blame/a48494d2fbed58c751e9b7e8fbff88582f9b2d02/README#L88). We don't use rules like [Conventional Commits](https://www.conventionalcommits.org/) because it's hard for beginners to write correct commit messages. It will encourage reviewers to spend more time on high-level problems, not the details. We also want to reduce the overhead when reading the git-blame, for example, `fix: correct minor typos in code` is the same as `fix minor typos in code`, there's no need to repeat content in the title line.

## Become a maintainer

At the early stage of this project, we will use a simple model to promote contributors to maintainers.
Anybody who has contributed code or doc to the project can get write access to issues and PRs contributors.
Maintainers will have all the permissions of this project, only the first 2 maintainers are granted by the owner, then we will start to elect
new maintainers by voting in the public issue. If no one votes down and 2/3 votes up then one election will pass.
