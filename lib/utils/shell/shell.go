package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strings"
)

// Shell https://github.com/riywo/loginshell/blob/master/loginshell.go
func Shell() (string, error) {
	switch runtime.GOOS {
	case "plan9":
		return plan9Shell()
	case "linux":
		return nixShell()
	case "openbsd":
		return nixShell()
	case "freebsd":
		return nixShell()
	case "android":
		return androidShell()
	case "darwin":
		return darwinShell()
	case "windows":
		return windowsShell()
	}

	return "", errors.New("Undefined GOOS: " + runtime.GOOS)
}

func plan9Shell() (string, error) {
	if _, err := os.Stat("/dev/osversion"); err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", errors.New("/dev/osversion check failed")
	}

	return "/bin/rc", nil
}

func nixShell() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}

	out, err := exec.Command("getent", "passwd", user.Uid).Output()
	if err != nil {
		return "", err
	}

	ent := strings.Split(strings.TrimSuffix(string(out), "\n"), ":")
	return ent[6], nil
}

func androidShell() (string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "", errors.New("shell not defined in android")
	}
	return shell, nil
}

func darwinShell() (string, error) {
	dir := "Local/Default/Users/" + os.Getenv("USER")
	out, err := exec.Command("dscl", "localhost", "-read", dir, "UserShell").Output()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("UserShell: (/[^ ]+)\n")
	matched := re.FindStringSubmatch(string(out))
	shell := matched[1]
	if shell == "" {
		return "", fmt.Errorf("Invalid output: %s", string(out))
	}

	return shell, nil
}

func windowsShell() (string, error) {
	consoleApp := os.Getenv("COMSPEC")
	if consoleApp == "" {
		consoleApp = "cmd.exe"
	}

	return consoleApp, nil
}
