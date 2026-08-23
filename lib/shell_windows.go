//go:build windows

package lib

import (
	"os/exec"
)

// setNewProcessGroup is a no-op placeholder on Windows; process-group
// termination there would require CREATE_NEW_PROCESS_GROUP + job objects,
// which is out of scope here. Killing the direct process plus WaitDelay's
// forced pipe closure still prevents an indefinite hang.
func setNewProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills just the process itself on Windows.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
