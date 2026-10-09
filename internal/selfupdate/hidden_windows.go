package selfupdate

import (
	"os/exec"
	"syscall"
)

func configureHidden(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
