# Contributing

Anyone has contributed code to the project can become a member of the project and have the write permission to issues and doc repositories.

At the early stage of this project, we will use a simple model to promote members to maintainers.
Maintainers will have all the permissions of this project, only the first 2 maintainers are granted by the owner, the standard is whether the member is good enough to review others' code, then we will start to elect
new maintainers by voting in the public issue. If no one votes down and 2/3 votes up then an election passes.

## Contribute Doc

Check [here](https://github.com/go-rod/go-rod.github.io/blob/main/contribute-doc.md).

## Terminology

When we talk about type in the doc we use [gopls](https://github.com/golang/tools/tree/master/gopls) symbol query syntax. For example, when we say `rod.Page.PDF`, you can run the below to locate the file and line of it:

```bash
gopls workspace_symbol -matcher=fuzzy rod.Page.PDF$
```

- `cdp`: It's short for Chrome Devtools Protocol

## How it works

Here's the common start process of rod:

1. Try to connect to a Devtools endpoint (WebSocket), if not found try to launch a local browser, if still not found try to download one, then connect again. The lib to handle it is [launcher](lib/launcher).

1. Use the JSON-RPC to talk to the Devtools endpoint to control the browser. The lib handles it is [cdp](lib/cdp).

1. Use the type definitions of the JSON-RPC to perform high-level actions. The lib handles it is [proto](lib/proto).

Object model:

![object model](../fixtures/object-model.svg)

## Run tests

First, launch a test shell for rod:

```bash
go run ./lib/utils/shell
```

Then, no magic, just `go test`. Read the test template [rod_test.go](../rod_test.go) to get started.

The entry point of tests is [setup_test.go](../setup_test.go). All the test helpers are defined in it.

The `cdp` requests of each test will be recorded and output to folder `tmp/cdp-log`, the CI will store them as
[artifacts](https://docs.github.com/en/actions/guides/storing-workflow-data-as-artifacts) so that we can download
them for debugging.

### Disable headless mode

```bash
rod=show,trace,slow=2s go test
```

Check type `defaults.ResetWithEnv` for how it works.

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
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

It will open a web page to tell you which line is not covered.

To cover the error branch of the code we usually intercept cdp calls.
There are several helper functions for it:

- `rod_test.MockClient.stubCounter`
- `rod_test.MockClient.stub`
- `rod_test.MockClient.stubErr`

### Use Docker for development

1. Build the test image: `docker build -t rod -f lib/docker/dev.Dockerfile .`

1. Run a container with and mount the cache volume to it: `docker run -v $(pwd):/t --name rod -it rod bash`

1. Open another terminal, copy your global git-ignore file to the container: `docker cp ~/.gitignore_global rod:/root/`

1. Run lint in the container: `go run ./lib/utils/lint`

1. Run tests in the container: `go test`

1. After you exit the container with `exit`, you can restart it by: `docker start -i rod`

### Deployment of docker images

We use `.github/workflows/docker.yml` to automate it.

### Detect goroutine leak

Because parallel execution will pollution the global goroutine stack, by default, the goroutine leak detection for each test will be disabled, but the detection for the whole test program will still work as well. To enable detection for each test, just use `go test -parallel=1`.

### Debug dependency libs

Run `go mod vendor` to create a local mirror of dependencies.
The Golang compiler will use the libs under `vendor` folder as a priority.
For example, we can modify file `./vendor/github.com/ysmood/goob/goob.go` to debug, such as add some extra logs.

## Comments

All conversations in Github issues, PRs, etc. should be summarized into code comments so that this project is not deep coupled with Github service.

## Convention of the git commit message

The commit message follows the rules [here](https://github.com/torvalds/subsurface-for-dirk/blame/a48494d2fbed58c751e9b7e8fbff88582f9b2d02/README#L88). We don't use rules like [Conventional Commits](https://www.conventionalcommits.org/) because it's hard for beginners to write correct commit messages. It will encourage reviewers to spend more time on high-level problems, not the details. We also want to reduce the overhead when reading the git-blame, for example, `fix: correct minor typos in code` is the same as `fix minor typos in code`, there's no need to repeat content in the title line.
