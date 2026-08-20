package platform

import (
	"fmt"
	"strings"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/command"
)

type GitInstaller struct {
	osName   string
	executor command.Executor
}

type GitHubCLIInstaller struct {
	osName   string
	executor command.Executor
}

func NewGitInstaller(osName string, executor command.Executor) *GitInstaller {
	return &GitInstaller{osName: osName, executor: executor}
}

func NewGitHubCLIInstaller(osName string, executor command.Executor) *GitHubCLIInstaller {
	return &GitHubCLIInstaller{osName: osName, executor: executor}
}

func (i *GitHubCLIInstaller) InstallGitHubCLI() error {
	if i.osName != "windows" {
		return fmt.Errorf("a instalação automática do GitHub CLI está disponível apenas no Windows; consulte https://cli.github.com")
	}
	if path, ok := find(i.executor, "winget"); ok {
		return run(i.executor, path, "install", "--id", "GitHub.cli", "-e", "--source", "winget", "--accept-package-agreements", "--accept-source-agreements")
	}
	if path, ok := find(i.executor, "choco"); ok {
		return run(i.executor, path, "install", "gh", "-y")
	}
	if path, ok := find(i.executor, "scoop"); ok {
		return run(i.executor, path, "install", "gh")
	}
	return fmt.Errorf("nenhum instalador compatível foi encontrado (winget, Chocolatey ou Scoop); instale o GitHub CLI por https://cli.github.com")
}

func find(executor command.Executor, name string) (string, bool) {
	path, err := executor.LookPath(name)
	return path, err == nil
}

func run(executor command.Executor, name string, args ...string) error {
	if err := executor.Interactive(name, args...); err != nil {
		return fmt.Errorf("o comando de instalação falhou: %w", err)
	}
	return nil
}

func (i *GitInstaller) InstallGit() error {
	switch i.osName {
	case "windows":
		return i.installWindows()
	case "linux":
		return i.installLinux()
	case "darwin":
		return i.installMacOS()
	default:
		return fmt.Errorf("sistema operacional não suportado para instalação automática: %s", i.osName)
	}
}

func (i *GitInstaller) installWindows() error {
	if path, ok := i.find("winget"); ok {
		return i.run(path, "install", "--id", "Git.Git", "-e", "--source", "winget", "--accept-package-agreements", "--accept-source-agreements")
	}
	if path, ok := i.find("choco"); ok {
		return i.run(path, "install", "git", "-y")
	}
	if path, ok := i.find("scoop"); ok {
		return i.run(path, "install", "git")
	}
	return fmt.Errorf("nenhum instalador compatível foi encontrado (winget, Chocolatey ou Scoop); instale o Git por https://git-scm.com/download/win")
}

func (i *GitInstaller) installLinux() error {
	type manager struct {
		name string
		args []string
	}
	managers := []manager{
		{name: "apt-get", args: []string{"install", "-y", "git"}},
		{name: "dnf", args: []string{"install", "-y", "git"}},
		{name: "yum", args: []string{"install", "-y", "git"}},
		{name: "pacman", args: []string{"-Sy", "--noconfirm", "git"}},
		{name: "zypper", args: []string{"--non-interactive", "install", "git"}},
		{name: "apk", args: []string{"add", "git"}},
		{name: "emerge", args: []string{"dev-vcs/git"}},
	}
	for _, candidate := range managers {
		if path, ok := i.find(candidate.name); ok {
			return i.runElevated(path, candidate.args...)
		}
	}
	return fmt.Errorf("nenhum gerenciador de pacotes compatível foi encontrado (apt-get, dnf, yum, pacman, zypper, apk ou emerge)")
}

func (i *GitInstaller) installMacOS() error {
	if path, ok := i.find("brew"); ok {
		return i.run(path, "install", "git")
	}
	if path, ok := i.find("port"); ok {
		return i.runElevated(path, "install", "git")
	}
	return fmt.Errorf("Homebrew ou MacPorts não foi encontrado; instale as Command Line Tools com 'xcode-select --install' e tente novamente")
}

func (i *GitInstaller) runElevated(name string, args ...string) error {
	if id, err := i.executor.CombinedOutput("id", "-u"); err == nil && strings.TrimSpace(id) == "0" {
		return i.run(name, args...)
	}
	if sudo, ok := i.find("sudo"); ok {
		return i.run(sudo, append([]string{name}, args...)...)
	}
	return fmt.Errorf("a instalação requer privilégios administrativos, mas o comando sudo não foi encontrado")
}

func (i *GitInstaller) run(name string, args ...string) error {
	return run(i.executor, name, args...)
}

func (i *GitInstaller) find(name string) (string, bool) {
	return find(i.executor, name)
}
