package command

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLookPathFindsGitHubCLIAfterWindowsInstallerChangesPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific fallback")
	}
	programFiles := t.TempDir()
	executable := filepath.Join(programFiles, "GitHub CLI", "gh.exe")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	t.Setenv("ProgramFiles", programFiles)
	t.Setenv("ProgramFiles(x86)", "")
	t.Setenv("ProgramData", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("LOCALAPPDATA", "")

	found, err := (OSExecutor{}).LookPath("gh")
	if err != nil {
		t.Fatal(err)
	}
	if !stringsEqualPath(found, executable) {
		t.Fatalf("unexpected executable: want %q, got %q", executable, found)
	}
}

func stringsEqualPath(left, right string) bool {
	left, leftErr := filepath.Abs(left)
	right, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(left) == filepath.Clean(right)
}
