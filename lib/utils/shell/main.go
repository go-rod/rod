// Package main It helps to launcher a transparent shell under the current shell with
// some extra environment variables that are required by rod testing.
package main

import (
	"os"
	"os/exec"

	"github.com/go-rod/rod/lib/utils"
)

func main() {
	list := []string{}
	for k, v := range utils.TestEnvs {
		list = append(list, k+"="+v)
	}

	bin, err := Shell()
	utils.E(err)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), list...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	_ = cmd.Run()
	os.Exit(cmd.ProcessState.ExitCode())
}
