package git

import (
	"errors"
	"reflect"
	"testing"
)

type recordingExecutor struct {
	calls [][]string
}

func (e *recordingExecutor) CombinedOutput(name string, args ...string) (string, error) {
	e.calls = append(e.calls, append([]string{name}, args...))
	return "", nil
}

func (e *recordingExecutor) Interactive(string, ...string) error { return nil }

func (e *recordingExecutor) LookPath(name string) (string, error) {
	if name == "git" {
		return "git", nil
	}
	return "", errors.New("not found")
}

func TestResetHistoryUsesExpectedGitCommands(t *testing.T) {
	executor := &recordingExecutor{}
	client := New(executor)

	operations := []func() error{
		func() error { return client.CreateOrphanBranch("zyr-reset-history") },
		client.AddAllFiles,
		func() error { return client.CommitInitial("Initial commit") },
		func() error { return client.ReplaceCurrentBranch("main") },
		func() error { return client.ForcePush("main") },
	}
	for _, operation := range operations {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
	}

	expected := [][]string{
		{"git", "checkout", "--orphan", "zyr-reset-history"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", "Initial commit"},
		{"git", "branch", "-M", "main"},
		{"git", "push", "--force", "origin", "main"},
	}
	if !reflect.DeepEqual(executor.calls, expected) {
		t.Fatalf("unexpected Git commands:\nwant %v\n got %v", expected, executor.calls)
	}
}
