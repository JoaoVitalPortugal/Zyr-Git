package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/app"
	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/command"
	gitclient "github.com/JoaoVitalPortugal/zyr-git-commit/internal/git"
	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/gitignore"
	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/platform"
	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/state"
	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/terminal"
)

const componentProtocol = "zyr-component/git/1"

var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "--zyr-component-protocol" {
		fmt.Fprintln(os.Stdout, componentProtocol)
		return
	}
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		fmt.Fprintf(os.Stdout, "Zyr Git Commit %s\n", version)
		return
	}
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "help")) {
		printHelp()
		return
	}
	if len(args) != 1 || args[0] != "commit" {
		fmt.Fprintln(os.Stderr, "✕ Comando Git desconhecido.")
		printHelp()
		os.Exit(2)
	}

	ui := terminal.New(os.Stdin, os.Stdout)
	executor := command.OSExecutor{}
	git := gitclient.New(executor)
	application := app.Application{
		Git:       git,
		Installer: platform.NewGitInstaller(runtime.GOOS, executor),
		State:     state.NewGitState(git),
		Ignore:    gitignore.NewCurrentDirectoryManager(),
		UI:        ui,
	}
	if err := application.Run(context.Background()); err != nil {
		ui.Error(err.Error())
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Fprintln(os.Stdout, "Uso: zyr git <comando>")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Comandos:")
	fmt.Fprintln(os.Stdout, "  commit    Adiciona alterações, cria um commit e faz push")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Execute: zyr git commit")
}
