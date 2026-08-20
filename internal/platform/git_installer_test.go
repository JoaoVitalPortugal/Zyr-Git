package platform

import (
	"errors"
	"reflect"
	"testing"
)

type fakeExecutor struct {
	paths       map[string]string
	id          string
	interactive []string
	args        [][]string
}

func (e *fakeExecutor) CombinedOutput(name string, args ...string) (string, error) {
	if name == "id" {
		return e.id, nil
	}
	return "", nil
}
func (e *fakeExecutor) Interactive(name string, args ...string) error {
	e.interactive = append(e.interactive, name)
	e.args = append(e.args, args)
	return nil
}
func (e *fakeExecutor) LookPath(name string) (string, error) {
	if path, found := e.paths[name]; found {
		return path, nil
	}
	return "", errors.New("not found")
}

// Scenario 11: Windows chooses an available Windows package manager.
func TestWindowsGitInstallationUsesWinget(t *testing.T) {
	executor := &fakeExecutor{paths: map[string]string{"winget": `C:\winget.exe`}}
	if err := NewGitInstaller("windows", executor).InstallGit(); err != nil {
		t.Fatal(err)
	}
	if executor.interactive[0] != `C:\winget.exe` || !reflect.DeepEqual(executor.args[0][:4], []string{"install", "--id", "Git.Git", "-e"}) {
		t.Fatalf("unexpected Windows command: %v %v", executor.interactive, executor.args)
	}
}

// Scenario 12: Linux detects the available manager instead of assuming apt.
func TestLinuxGitInstallationDetectsDNF(t *testing.T) {
	executor := &fakeExecutor{paths: map[string]string{"dnf": "/usr/bin/dnf"}, id: "0"}
	if err := NewGitInstaller("linux", executor).InstallGit(); err != nil {
		t.Fatal(err)
	}
	if executor.interactive[0] != "/usr/bin/dnf" || !reflect.DeepEqual(executor.args[0], []string{"install", "-y", "git"}) {
		t.Fatalf("unexpected Linux command: %v %v", executor.interactive, executor.args)
	}
}

// Scenario 13: macOS uses Homebrew when detected.
func TestMacOSGitInstallationUsesHomebrew(t *testing.T) {
	executor := &fakeExecutor{paths: map[string]string{"brew": "/opt/homebrew/bin/brew"}}
	if err := NewGitInstaller("darwin", executor).InstallGit(); err != nil {
		t.Fatal(err)
	}
	if executor.interactive[0] != "/opt/homebrew/bin/brew" || !reflect.DeepEqual(executor.args[0], []string{"install", "git"}) {
		t.Fatalf("unexpected macOS command: %v %v", executor.interactive, executor.args)
	}
}

func TestWindowsGitHubCLIInstallationUsesWinget(t *testing.T) {
	executor := &fakeExecutor{paths: map[string]string{"winget": `C:\winget.exe`}}
	if err := NewGitHubCLIInstaller("windows", executor).InstallGitHubCLI(); err != nil {
		t.Fatal(err)
	}
	want := []string{"install", "--id", "GitHub.cli", "-e", "--source", "winget", "--accept-package-agreements", "--accept-source-agreements"}
	if executor.interactive[0] != `C:\winget.exe` || !reflect.DeepEqual(executor.args[0], want) {
		t.Fatalf("unexpected GitHub CLI command: %v %v", executor.interactive, executor.args)
	}
}

func TestWindowsGitHubCLIInstallationFallsBackToChocolatey(t *testing.T) {
	executor := &fakeExecutor{paths: map[string]string{"choco": `C:\choco.exe`}}
	if err := NewGitHubCLIInstaller("windows", executor).InstallGitHubCLI(); err != nil {
		t.Fatal(err)
	}
	if executor.interactive[0] != `C:\choco.exe` || !reflect.DeepEqual(executor.args[0], []string{"install", "gh", "-y"}) {
		t.Fatalf("unexpected GitHub CLI command: %v %v", executor.interactive, executor.args)
	}
}

func TestGitHubCLIAutomaticInstallationIsWindowsOnly(t *testing.T) {
	executor := &fakeExecutor{paths: map[string]string{"brew": "/opt/homebrew/bin/brew"}}
	err := NewGitHubCLIInstaller("darwin", executor).InstallGitHubCLI()
	if err == nil || len(executor.interactive) != 0 {
		t.Fatalf("non-Windows installation should be manual: err=%v commands=%v", err, executor.interactive)
	}
}
