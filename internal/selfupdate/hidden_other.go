//go:build !windows

package selfupdate

import "os/exec"

func configureHidden(command *exec.Cmd) {}
