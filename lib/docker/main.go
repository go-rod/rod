// The .github/workflows/docker.yml uses it as an github action
// and run it like this:
//
//	GITHUB_TOKEN=$TOKEN go run ./lib/utils/docker $GITHUB_REF
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	event := os.Args[1]

	fmt.Println("Event:", event)

	isMain := regexp.MustCompile(`^refs/heads/main$`).MatchString(event)
	m := regexp.MustCompile(`^refs/tags/(v[0-9]+\.[0-9]+\.[0-9]+)$`).FindStringSubmatch(event)
	ver := ""
	if len(m) > 1 {
		ver = m[1]
	}

	at := getArchType()

	if isMain {
		releaseLatest(at)
	} else if ver != "" {
		releaseWithVer(ver)
	} else {
		test(at)
	}
}

func releaseLatest(at archType) {
	login()
	test(at)

	utils.Exec("docker push", at.tagDev())
	utils.Exec("docker push", at.tag())
}

func releaseWithVer(ver string) {
	login()

	verImageDev := registry + ":" + ver + "-dev"
	utils.Exec("docker manifest create", verImageDev, archAmd.tagDev(), archArm.tagDev())
	utils.Exec("docker manifest push", verImageDev)

	verImage := registry + ":" + ver
	utils.Exec("docker manifest create", verImage, archAmd.tag(), archArm.tag())
	utils.Exec("docker manifest push", verImage)

	registryDev := registry + ":dev"
	utils.Exec("docker manifest create", registryDev, archAmd.tagDev(), archArm.tagDev())
	utils.Exec("docker manifest push", registryDev)

	utils.Exec("docker manifest create", registry, archAmd.tag(), archArm.tag())
	utils.Exec("docker manifest push", registry)
}

func test(at archType) {
	utils.Exec("docker build -f=lib/docker/Dockerfile", "--platform", at.platform(), "-t", at.tag(), description(false), ".")
	utils.Exec("docker build -f=lib/docker/dev.Dockerfile",
		"--platform", at.platform(),
		"--build-arg", "golang="+at.golang(),
		"--build-arg", "nodejs="+at.nodejs(),
		"-t", at.tagDev(),
		description(true), ".",
	)

	utils.Exec("docker run", at.tag(), "rod-manager", "-h")

	// TODO: arm cross execution for chromium doesn't work well on github actions.
	if at != archArm {
		wd, err := os.Getwd()
		utils.E(err)
		utils.Exec("docker run -w=/t -v", fmt.Sprintf("%s:/t", wd), at.tagDev(), "go", "run", "./lib/utils/ci-test")
	}
}

func login() {
	cmd := exec.Command("docker", "login", "-u=rod-robot", "-p", os.Getenv("GITHUB_TOKEN"), registry)
	out, err := cmd.CombinedOutput()
	utils.E(err)
	utils.E(os.Stdout.Write(out))
}

var headSha = strings.TrimSpace(utils.ExecLine(false, "git", "rev-parse", "HEAD"))

func description(dev bool) string {
	f := "Dockerfile"
	if dev {
		f = "dev." + f
	}

	return `--label=org.opencontainers.image.description=https://github.com/go-rod/rod/blob/` + headSha + "/lib/docker/" + f
}

const registry = "ghcr.io/go-rod/rod"

type archType int

const (
	archAmd archType = iota
	archArm
)

func getArchType() archType {
	arch := os.Getenv("ARCH")
	switch arch {
	case "arm":
		return archArm
	default:
		return archAmd
	}
}

func (at archType) platform() string {
	switch at {
	case archArm:
		return "linux/arm64"
	default:
		return "linux/amd64"
	}
}

func (at archType) tag() string {
	switch at {
	case archArm:
		return registry + ":arm"
	default:
		return registry + ":amd"
	}
}

func (at archType) tagDev() string {
	switch at {
	case archArm:
		return registry + ":arm-dev"
	default:
		return registry + ":amd-dev"
	}
}

func (at archType) golang() string {
	switch at {
	case archArm:
		return "https://go.dev/dl/go1.19.1.linux-arm64.tar.gz"
	default:
		return "https://go.dev/dl/go1.19.1.linux-amd64.tar.gz"
	}
}

func (at archType) nodejs() string {
	switch at {
	case archArm:
		return "https://nodejs.org/dist/v16.17.0/node-v16.17.0-linux-arm64.tar.xz"
	default:
		return "https://nodejs.org/dist/v16.17.0/node-v16.17.0-linux-x64.tar.xz"
	}
}
