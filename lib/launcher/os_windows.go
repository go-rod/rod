// +build windows

package launcher

import (
	"os/exec"
	"syscall"
)

func killGroup(pid int) {
	terminateProcess(pid)
}

func (l *Launcher) osSetupCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func terminateProcess(pid int) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, true, uint32(pid))
	if err != nil {
		return
	}

	syscall.TerminateProcess(handle, 0)
	syscall.CloseHandle(handle)
}
