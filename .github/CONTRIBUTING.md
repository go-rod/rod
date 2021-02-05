# Contributing

Your help is more than welcome! Even just open an issue to ask a question may greatly help others.

Please read [How To Ask Questions The Smart Way](http://www.catb.org/~esr/faqs/smart-questions.html) before you ask questions.

We use Github Projects to manage tasks, you can see the priority and progress of the issues [here](https://github.com/orgs/go-rod/projects/1).

## Become a maintainer

Anyone has contributed code to the project can become a member of the project and have the write permission to issues and doc repositories.

At the early stage of this project, we will use a simple model to promote members to maintainers.
Maintainers will have all the permissions of this project, only the first 2 maintainers are granted by the owner, the standard is whether the member is good enough to review others' code, then we will start to elect
new maintainers by voting in the public issue. If no one votes down and 2/3 votes up then an election passes.

## Terminology

When we talk about type in the doc we use [gopls](https://github.com/golang/tools/tree/master/gopls) symbol query syntax. For example, when we say `rod.Page.PDF`, you can run the below to locate the file and line of it:

```bash
gopls workspace_symbol -matcher=fuzzy rod.Page.PDF$
```

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

### Specify browser binary

You can use the `-browser-bin` flag to specify a custom browser executable path:

For example:

```bash
go test -v --browser-bin=/path/to/browser
```

### Lint project

You can run all commands inside Docker so that you don't have to install all the development dependencies.
Check [Use Docker for development](#Use-Docker-for-development) for more info.

```bash
go generate # only required for first time
go run ./lib/utils/lint
```

### Code Coverage

If the code coverage is less than 100%, the CI will fail.

Learn the [basics](https://blog.golang.org/cover) first.

To visually see the coverage report you can run something like this:

```bash
go test -coverprofile=coverage.txt
go tool cover -html=coverage.txt
```

It will open a web page to tell you which line is not covered.

To cover the error branch of the code we usually intercept cdp calls.
There are several helper functions for it:

- `rod_test.MockClient.stubCounter`
- `rod_test.MockClient.stub`
- `rod_test.MockClient.stubErr`

### Use Docker for development

1. Build the test image: `docker build -t rod -f lib/docker/test.Dockerfile .`

1. Run a container with and mount the cache volume to it: `docker run -v $(pwd):/t --name rod -it rod bash`

1. Open another terminal, copy your global git-ignore file to the container: `docker cp ~/.gitignore_global rod:/root/`

1. Run lint in the container: `go run ./lib/utils/lint`

1. Run tests in the container: `go test -run /Click -v`

1. After you exit the container with `exit`, you can restart it by: `docker start -i rod`

### Detect goroutine leak

Because parallel execution will pollution the global goroutine stack, by default, the goroutine leak detection for each test will be disabled, but the detection for the whole test program will still work as well. To enable detection for each test, just use `go test -parallel=1`.

## Convention of the git commit message

The commit message follows the rules [here](https://github.com/torvalds/subsurface-for-dirk/blame/a48494d2fbed58c751e9b7e8fbff88582f9b2d02/README#L88). We don't use rules like [Conventional Commits](https://www.conventionalcommits.org/) because it's hard for beginners to write correct commit messages. It will encourage reviewers to spend more time on high-level problems, not the details. We also want to reduce the overhead when reading the git-blame, for example, `fix: correct minor typos in code` is the same as `fix minor typos in code`, there's no need to repeat content in the title line.
