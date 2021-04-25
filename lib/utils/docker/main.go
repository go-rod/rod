// The .github/workflows/docker.yml will use it as an github action
// To test it locally, you can generate a personal github token: https://github.com/settings/tokens
// Then run this:
//   ROD_GITHUB_ROBOT=your_token GITHUB_REF=refs/heads/master go run ./lib/utils/docker
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

var ref = os.Getenv("GITHUB_REF")
var token = os.Getenv("ROD_GITHUB_ROBOT")

func main() {
	fmt.Println("GITHUB_REF:", ref)

	master := regexp.MustCompile(`^refs/heads/master$`).MatchString(ref)
	m := regexp.MustCompile(`^refs/tags/(v[0-9]+\.[0-9]+\.[0-9]+)$`).FindStringSubmatch(ref)
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

	utils.Exec("docker", "run", image, "rod-launcher", "-h")
	utils.Exec("docker", "run", "-v", fmt.Sprintf("%s:/t", wd), "-w=/t", "dev", "go", "test")
}

func login() {
	utils.Exec("docker", "login", registry, "-u=rod-robot", "-p="+token)
}

func description() string {
	b, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	utils.E(err)

	sha := strings.TrimSpace(string(b))

	return `--label=org.opencontainers.image.description=https://github.com/go-rod/rod/commit/` + sha
}
