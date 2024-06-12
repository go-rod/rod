//go:build !windows

package launcher

import (
	"os/exec"
	"syscall"

	"github.com/go-rod/rod/lib/launcher/flags"
)

func killGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

func (l *Launcher) osSetupCmd(cmd *exec.Cmd) {
	if flags, has := l.GetFlags(flags.XVFB); has {
		var command []string
		// flags must append before cmd.Args
		command = append(command, flags...)
		command = append(command, cmd.Args...)

		*cmd = *exec.Command("xvfb-run", command...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
