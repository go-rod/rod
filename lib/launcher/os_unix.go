// +build !windows

package launcher

import (
	"os/exec"
	"syscall"
)

func killGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

func (l *Launcher) osSetupCmd(cmd *exec.Cmd) {
	if _, has := l.Get(flagXVFB); has {
		*cmd = *exec.Command("xvfb-run", cmd.Args...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
