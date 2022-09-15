// The .github/workflows/docker.yml uses it as an github action
// and run it like this:
//
//	DOCKER_TOKEN=$TOKEN go run ./lib/utils/docker $GITHUB_REF
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

	master := regexp.MustCompile(`^refs/heads/master$`).MatchString(event)
	m := regexp.MustCompile(`^refs/tags/(v[0-9]+\.[0-9]+\.[0-9]+)$`).FindStringSubmatch(event)
	ver := ""
	if len(m) > 1 {
		ver = m[1]
	}

	at := getArchType()

	if master {
		releaseLatest(at)
	} else if ver != "" {
		releaseWithVer(ver, at)
	} else {
		test(at)
	}
}

func releaseLatest(at archType) {
	login()
	test(at)

	utils.Exec("docker push", at.tag())
	utils.Exec("docker push", at.tagDev())
}

func releaseWithVer(ver string, at archType) {
	login()

	utils.Exec("docker manifest create", registry, archAmd.tag(), archArm.tag())
	utils.Exec("docker manifest push", registry)

	registryDev := registry + ":dev"
	utils.Exec("docker manifest create", registryDev, archAmd.tagDev(), archArm.tagDev())
	utils.Exec("docker manifest push", registryDev)

	verImage := registry + ":" + ver
	verImageDev := registry + ":" + ver + "-dev"

	utils.Exec("docker manifest create", verImage, archAmd.tag(), archArm.tag())
	utils.Exec("docker manifest push", verImage)

	utils.Exec("docker manifest create", verImageDev, archAmd.tagDev(), archArm.tagDev())
	utils.Exec("docker manifest push", verImageDev)
}

func test(at archType) {
	utils.Exec("docker build -f=lib/docker/Dockerfile", "--platform", at.platform(), "-t", at.tag(), description(false), ".")
	utils.Exec("docker build -f=lib/docker/dev.Dockerfile", "--platform", at.platform(), "-t", at.tagDev(), description(true), ".")

	utils.Exec("docker run", at.tag(), "rod-manager", "-h")

	wd, err := os.Getwd()
	utils.E(err)
	utils.Exec("docker run -w=/t -v", fmt.Sprintf("%s:/t", wd), at.tagDev(), "go", "test")
}

func login() {
	cmd := exec.Command("docker", "login", "-u=rod-robot", "-p", os.Getenv("DOCKER_TOKEN"), registry)
	out, err := cmd.CombinedOutput()
	utils.E(err)
	utils.E(os.Stdout.Write(out))
}

func description(dev bool) string {
	sha := strings.TrimSpace(utils.ExecLine(false, "git", "rev-parse", "HEAD"))

	f := "Dockerfile"
	if dev {
		f = "dev." + f
	}

	return `--label=org.opencontainers.image.description=https://github.com/go-rod/rod/blob/` + sha + "/lib/docker/" + f
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
