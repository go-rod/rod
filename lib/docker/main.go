// The .github/workflows/docker.yml uses it as an github action
// and run it like this:
//
//	DOCKER_TOKEN=$TOKEN go run ./lib/utils/docker $GITHUB_REF
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/go-rod/rod/lib/utils"
)

const registry = "ghcr.io"
const image = registry + "/go-rod/rod"
const imageDev = image + ":dev"
const imageAMD = image + ":amd64"
const imageAMDDev = image + ":amd64-dev"
const imageARM = image + ":arm64"
const imageARMDev = image + ":arm64-dev"

var token = os.Getenv("DOCKER_TOKEN")

func main() {
	event := os.Args[1]

	fmt.Println("Event:", event)

	master := regexp.MustCompile(`^refs/heads/master$`).MatchString(event)
	m := regexp.MustCompile(`^refs/tags/(v[0-9]+\.[0-9]+\.[0-9]+)$`).FindStringSubmatch(event)
	ver := ""
	if len(m) > 1 {
		ver = m[1]
	}

	if master {
		releaseLatest()
	} else if ver != "" {
		releaseWithVer(ver)
	} else {
		test()
	}
}

func releaseLatest() {
	login()
	test()

	utils.Exec("docker push", imageAMD)
	utils.Exec("docker push", imageAMDDev)

	utils.Exec("docker push", imageARM)
	utils.Exec("docker push", imageARMDev)

	// create manifest
	utils.Exec("docker manifest create", image, imageAMD, imageARM)
	utils.Exec("docker manifest push", image)
	utils.Exec("docker manifest create", imageDev, imageAMDDev, imageARMDev)
	utils.Exec("docker manifest push", imageDev)
}

func releaseWithVer(ver string) {
	login()

	verImage := image + ":" + ver
	verImageDev := image + ":" + ver + "-dev"

	utils.Exec("docker manifest create", verImage, imageAMD, imageARM)
	utils.Exec("docker manifest push", verImage)

	utils.Exec("docker manifest create", verImageDev, imageAMDDev, imageARMDev)
	utils.Exec("docker manifest push", verImageDev)
}

func test() {
	// build amd64 images
	// utils.Exec("docker build --platform linux/amd64 -f=lib/docker/Dockerfile -t", imageAMD, description(false), ".")
	// utils.Exec("docker build --platform linux/amd64 -f=lib/docker/dev.Dockerfile -t", imageAMDDev, description(true), ".")

	// build arm64 images
	utils.Exec("docker build --platform linux/arm64 -f=lib/docker/Dockerfile -t", imageARM, description(false), ".")
	utils.Exec("docker build --platform linux/arm64 -f=lib/docker/dev.Dockerfile -t", imageARMDev, description(true), ".")

	// utils.Exec("docker run", imageAMD, "rod-manager", "-h")
	utils.Exec("docker run", imageARM, "rod-manager", "-h")

	// wd, err := os.Getwd()
	// utils.E(err)
	// utils.Exec("docker run -w=/t -v", fmt.Sprintf("%s:/t", wd), imageAMDDev, "go", "test")
}

func login() {
	utils.Exec("docker login -u=rod-robot", "-p", token, registry)
}

func description(dev bool) string {
	sha := strings.TrimSpace(utils.ExecLine(false, "git", "rev-parse", "HEAD"))

	f := "Dockerfile"
	if dev {
		f = "dev." + f
	}

	return `--label=org.opencontainers.image.description=https://github.com/go-rod/rod/blob/` + sha + "/lib/docker/" + f
}
