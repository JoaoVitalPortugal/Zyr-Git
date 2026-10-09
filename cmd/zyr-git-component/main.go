package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/JoaoVitalPortugal/zyr-git/internal/app"
	"github.com/JoaoVitalPortugal/zyr-git/internal/command"
	gitclient "github.com/JoaoVitalPortugal/zyr-git/internal/git"
	githubclient "github.com/JoaoVitalPortugal/zyr-git/internal/github"
	"github.com/JoaoVitalPortugal/zyr-git/internal/gitignore"
	"github.com/JoaoVitalPortugal/zyr-git/internal/platform"
	"github.com/JoaoVitalPortugal/zyr-git/internal/state"
	"github.com/JoaoVitalPortugal/zyr-git/internal/terminal"
)

const componentProtocol = "zyr-component/git/1"

var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "--zyr-component-protocol" {
		fmt.Fprintln(os.Stdout, componentProtocol)
		return
	}
	if os.Getenv("ZYR_GIT_UPDATE_RESUME") != "1" {
		if code, handled := checkAndResume(args); handled {
			os.Exit(code)
		}
	}
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		fmt.Fprintf(os.Stdout, "Zyr Git %s\n", version)
		return
	}
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "help")) {
		printHelp()
		return
	}
	if len(args) != 1 || (args[0] != "commit" && args[0] != "reset-history" && args[0] != "add-repo" && args[0] != "delete-repo" && args[0] != "update-gh") {
		fmt.Fprintln(os.Stderr, "✕ Comando Git desconhecido.")
		printHelp()
		os.Exit(2)
	}

	ui := terminal.New(os.Stdin, os.Stdout)
	executor := command.OSExecutor{}
	git := gitclient.New(executor)
	var runError error
	switch args[0] {
	case "commit":
		application := app.Application{
			Git:       git,
			Installer: platform.NewGitInstaller(runtime.GOOS, executor),
			State:     state.NewGitState(git),
			Ignore:    gitignore.NewCurrentDirectoryManager(),
			UI:        ui,
		}
		runError = application.Run(context.Background())
	case "reset-history":
		runError = (app.ResetHistoryApplication{Git: git, UI: ui}).Run(context.Background())
	case "add-repo":
		runError = (app.AddRepoApplication{
			GitHub:    githubclient.New(executor),
			Installer: platform.NewGitHubCLIInstaller(runtime.GOOS, executor),
			UI:        ui,
		}).Run(context.Background())
	case "delete-repo":
		runError = (app.DeleteRepoApplication{
			GitHub:    githubclient.New(executor),
			Installer: platform.NewGitHubCLIInstaller(runtime.GOOS, executor),
			UI:        ui,
		}).Run(context.Background())
	case "update-gh":
		runError = (app.UpdateGHApplication{
			GitHub:    githubclient.New(executor),
			Installer: platform.NewGitHubCLIInstaller(runtime.GOOS, executor),
			UI:        ui,
		}).Run(context.Background())
	}
	if runError != nil {
		ui.Error(runError.Error())
		os.Exit(1)
	}
}

func checkAndResume(args []string) (int, bool) {
	self, err := os.Executable()
	if err != nil {
		return 0, false
	}
	uiPath := filepath.Join(filepath.Dir(self), "zyr-git-dashboard.exe")
	if _, err := os.Stat(uiPath); err != nil {
		return 0, false
	}
	check := exec.Command(uiPath, "--check-and-update")
	err = check.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 10 {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Aviso: não foi possível verificar ou instalar a atualização; continuando com a versão atual.")
		}
		return 0, false
	}
	home := os.Getenv("ZYR_CLI_HOME")
	if home == "" {
		home = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Zyr CLI")
	}
	data, err := os.ReadFile(filepath.Join(home, "components", "git.json"))
	if err != nil {
		return 0, false
	}
	var manifest struct {
		Executable string `json:"executable"`
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.Executable == "" || manifest.Executable == self {
		return 0, false
	}
	command := exec.Command(manifest.Executable, args...)
	command.Env = append(os.Environ(), "ZYR_GIT_UPDATE_RESUME=1")
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		if errors.As(err, &exit) {
			return exit.ExitCode(), true
		}
		fmt.Fprintln(os.Stderr, "Falha ao retomar o comando após a atualização:", err)
		return 1, true
	}
	return 0, true
}

func printHelp() {
	fmt.Fprintln(os.Stdout, "Uso: zyr git <comando>")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Comandos:")
	fmt.Fprintln(os.Stdout, "  commit          Adiciona alterações, cria um commit e faz push")
	fmt.Fprintln(os.Stdout, "  reset-history   Substitui o histórico por um novo commit inicial")
	fmt.Fprintln(os.Stdout, "  add-repo        Cria um novo repositório remoto no GitHub")
	fmt.Fprintln(os.Stdout, "  delete-repo     Exclui permanentemente um repositório remoto do GitHub")
	fmt.Fprintln(os.Stdout, "  update-gh       Atualiza o GitHub CLI (gh) no Windows")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Execute: zyr git <comando>")
}
