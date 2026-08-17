package main

import (
	"fmt"
	"os"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/launcher"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--zyr-launcher-protocol" {
		fmt.Fprintln(os.Stdout, launcher.Protocol)
		return
	}

	home, err := launcher.HomeFromExecutable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "✕ "+err.Error())
		os.Exit(1)
	}
	dispatcher := launcher.Dispatcher{
		Home:         home,
		LauncherPath: os.Args[0],
		Version:      version,
		Out:          os.Stdout,
		Err:          os.Stderr,
		Runner:       launcher.OSRunner{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr},
	}
	os.Exit(dispatcher.Run(os.Args[1:]))
}
