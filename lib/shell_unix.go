//go:build !windows

package lib

import (
	"os/exec"
	"syscall"
)

// setNewProcessGroup configures cmd to run in its own process group so that
// any children it spawns (including backgrounded/orphaned processes such as
// `some_daemon &`) can be terminated together with it, rather than being
// left running and holding the stdout/stderr pipes open forever.
func setNewProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGKILL to the entire process group of cmd, not
// just the direct child, so orphaned/background descendants are cleaned up
// too. Falls back to killing just the process if the group isn't available.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		// Negative pid targets the whole process group.
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return nil
	}
	return cmd.Process.Kill()
}
