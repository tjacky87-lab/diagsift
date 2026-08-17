//go:build windows

package collector

import "os/exec"

func configureProcess(*exec.Cmd) {}

func terminateProcess(command *exec.Cmd) {
	if command.Process != nil {
		_ = command.Process.Kill()
	}
}
