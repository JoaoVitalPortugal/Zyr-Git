package app_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/app"
	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/command"
	gitclient "github.com/JoaoVitalPortugal/zyr-git-commit/internal/git"
	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/terminal"
)

func TestResetHistoryDeclinedLeavesRealRepositoryUntouched(t *testing.T) {
	work, remote := createResetHistoryRepository(t)
	beforeHead := runGit(t, work, "rev-parse", "HEAD")
	beforeRemote := runGitDir(t, remote, "rev-parse", "main")
	beforeStatus := runGit(t, work, "status", "--porcelain")

	runResetHistory(t, work, "n\n")

	if after := runGit(t, work, "rev-parse", "HEAD"); after != beforeHead {
		t.Fatalf("declined reset changed local HEAD: %s -> %s", beforeHead, after)
	}
	if after := runGitDir(t, remote, "rev-parse", "main"); after != beforeRemote {
		t.Fatalf("declined reset changed remote HEAD: %s -> %s", beforeRemote, after)
	}
	if after := runGit(t, work, "status", "--porcelain"); after != beforeStatus {
		t.Fatalf("declined reset changed working tree:\n%s\n%s", beforeStatus, after)
	}
}

func TestResetHistoryConfirmedReplacesRealLocalAndRemoteHistory(t *testing.T) {
	work, remote := createResetHistoryRepository(t)

	runResetHistory(t, work, "S\n")

	if count := runGit(t, work, "rev-list", "--count", "HEAD"); count != "1" {
		t.Fatalf("local history should contain one commit, got %s", count)
	}
	if count := runGitDir(t, remote, "rev-list", "--count", "main"); count != "1" {
		t.Fatalf("remote history should contain one commit, got %s", count)
	}
	if branch := runGit(t, work, "branch", "--show-current"); branch != "main" {
		t.Fatalf("original branch was not restored: %s", branch)
	}
	if _, err := os.Stat(filepath.Join(work, "current.txt")); err != nil {
		t.Fatal("current working tree was not preserved")
	}
}

func createResetHistoryRepository(t *testing.T) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git não está disponível para o teste de integração")
	}
	root := t.TempDir()
	work := filepath.Join(root, "project")
	remote := filepath.Join(root, "remote.git")
	runGitAt(t, root, "init", "--bare", remote)
	runGitAt(t, root, "init", "-b", "main", work)
	runGit(t, work, "config", "user.name", "Zyr Test")
	runGit(t, work, "config", "user.email", "zyr-test@example.invalid")

	writeTestFile(t, filepath.Join(work, "project.txt"), "primeira versão\n")
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "-m", "primeiro commit")
	runGit(t, work, "remote", "add", "origin", remote)
	runGit(t, work, "push", "-u", "origin", "main")

	writeTestFile(t, filepath.Join(work, "project.txt"), "segunda versão\n")
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "-m", "segundo commit")
	runGit(t, work, "push", "origin", "main")
	writeTestFile(t, filepath.Join(work, "current.txt"), "estado atual\n")
	return work, remote
}

func runResetHistory(t *testing.T, work, answer string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("não foi possível restaurar o diretório do teste: %v", err)
		}
	}()

	var output bytes.Buffer
	client := gitclient.New(command.OSExecutor{})
	application := app.ResetHistoryApplication{
		Git: client,
		UI:  terminal.New(strings.NewReader(answer), &output),
	}
	if err := application.Run(context.Background()); err != nil {
		t.Fatalf("reset-history falhou: %v\n%s", err, output.String())
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	return runGitAt(t, directory, append([]string{"-C", directory}, args...)...)
}

func runGitDir(t *testing.T, directory string, args ...string) string {
	t.Helper()
	return runGitAt(t, filepath.Dir(directory), append([]string{"--git-dir", directory}, args...)...)
}

func runGitAt(t *testing.T, directory string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = directory
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v falhou: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
