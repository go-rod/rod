// The .github/workflows/docker.yml will use it as an github action
// Then run this:
//   DOCKER_TOKEN=your_token go run ./lib/utils/docker refs/heads/master
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
}

func releaseWithVer(ver string) {
	login()
	utils.Exec("docker", "pull", image)
	utils.Exec("docker", "tag", image, image+":"+ver)
	utils.Exec("docker", "push", image+":"+ver)
}

func test() {
	utils.Exec("docker", "build", "-t", image, description(), "-f=lib/docker/Dockerfile", ".")
	utils.Exec("docker", "build", "-t=dev", "-f=lib/docker/dev.Dockerfile", ".")

	wd, err := os.Getwd()
	utils.E(err)

	utils.Exec("docker", "run", image, "rod-manager", "-h")
	utils.Exec("docker", "run", "-v", fmt.Sprintf("%s:/t", wd), "-w=/t", "dev", "go", "test")
}

func login() {
	utils.Exec("docker", "login", registry, "-u=rod-robot", "-p="+token)
}

func description() string {
	b, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	utils.E(err)

	sha := strings.TrimSpace(string(b))

	return `--label=org.opencontainers.image.description=https://github.com/go-rod/rod/blob/` + sha + "/lib/docker/Dockerfile"
}
