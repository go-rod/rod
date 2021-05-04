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
	if flags, has := l.GetFlags(flagXVFB); has {
		var command []string
		// flags must append before cmd.Args
		for _, flag := range flags {
			command=append(command,flag)
		}
		for _, arg := range cmd.Args {
			command=append(command,arg)
		}
		*cmd = *exec.Command("xvfb-run", command...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
