// The .github/workflows/docker.yml uses it as an github action
// and run it like this:
//   DOCKER_TOKEN=$TOKEN go run ./lib/utils/docker $GITHUB_REF
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
const devImage = image + ":dev"

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
	utils.Exec("docker push", image)
	utils.Exec("docker push", devImage)
}

func releaseWithVer(ver string) {
	login()

	verImage := image + ":" + ver

	utils.Exec("docker pull", image)
	utils.Exec("docker tag", image, verImage)
	utils.Exec("docker push", verImage)

	utils.Exec("docker pull", devImage)
	utils.Exec("docker tag", devImage, verImage+"-dev")
	utils.Exec("docker push", verImage+"-dev")
}

func test() {
	utils.Exec("docker build -f=lib/docker/Dockerfile -t", image, description(false), ".")
	utils.Exec("docker build -f=lib/docker/dev.Dockerfile -t", devImage, description(true), ".")

	wd, err := os.Getwd()
	utils.E(err)

	utils.Exec("docker run", image, "rod-manager", "-h")
	utils.Exec("docker run -w=/t -v", fmt.Sprintf("%s:/t", wd), devImage, "go", "test")
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
