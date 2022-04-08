// The .github/workflows/docker.yml uses it as an github action
// and run it like this:
//   DOCKER_TOKEN=$TOKEN go run ./lib/utils/docker $GITHUB_REF
package main

import (
	"fmt"
	"os"
	"os/exec"
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
	utils.Exec("docker", "push", image)
	utils.Exec("docker", "push", devImage)
}

func releaseWithVer(ver string) {
	login()

	verImage := image + ":" + ver

	utils.Exec("docker", "pull", image)
	utils.Exec("docker", "tag", image, verImage)
	utils.Exec("docker", "push", verImage)

	utils.Exec("docker", "pull", devImage)
	utils.Exec("docker", "tag", devImage, verImage+"-dev")
	utils.Exec("docker", "push", verImage+"-dev")
}

func test() {
	utils.Exec("docker", "build", "-t", image, description(false), "-f=lib/docker/Dockerfile", ".")
	utils.Exec("docker", "build", "-t", devImage, description(true), "-f=lib/docker/dev.Dockerfile", ".")

	wd, err := os.Getwd()
	utils.E(err)

	utils.Exec("docker", "run", image, "rod-manager", "-h")
	utils.Exec("docker", "run", "-v", fmt.Sprintf("%s:/t", wd), "-w=/t", devImage, "go", "test")
}

func login() {
	utils.Exec("docker", "login", registry, "-u=rod-robot", "-p="+token)
}

func description(dev bool) string {
	b, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	utils.E(err)

	sha := strings.TrimSpace(string(b))

	f := "Dockerfile"
	if dev {
		f = "dev." + f
	}

	return `--label=org.opencontainers.image.description=https://github.com/go-rod/rod/blob/` + sha + "/lib/docker/" + f
}
