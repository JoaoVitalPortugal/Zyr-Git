package launcher

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeRunner struct {
	executable string
	args       []string
	exitCode   int
	called     bool
}

func (r *fakeRunner) Run(executable string, args []string) (int, error) {
	r.called = true
	r.executable = executable
	r.args = append([]string(nil), args...)
	return r.exitCode, nil
}

func testDispatcher(t *testing.T, home string, runner *fakeRunner) (Dispatcher, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var output bytes.Buffer
	var errors bytes.Buffer
	return Dispatcher{
		Home:         home,
		LauncherPath: filepath.Join(home, "bin", "zyr.exe"),
		Version:      "test",
		Out:          &output,
		Err:          &errors,
		Runner:       runner,
	}, &output, &errors
}

func createExecutable(t *testing.T, directory, name string) string {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTestJSON(t *testing.T, path string, value interface{}) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func registerGitComponent(t *testing.T, home string) string {
	t.Helper()
	executable := createExecutable(t, filepath.Join(home, "installed"), "zyr-git-commit.exe")
	writeTestJSON(t, filepath.Join(home, "components", "git.json"), Component{
		SchemaVersion: 1,
		Command:       "git",
		Executable:    executable,
		Version:       "1.0.0",
		Description:   "Componente Git",
		Owner:         "test",
	})
	return executable
}

// Scenarios 1, 3 and 7: Git-only and coexistence always route git commit to
// the registered component, independent of legacy PATH order.
func TestGitCommitRoutesToRegisteredComponent(t *testing.T) {
	home := t.TempDir()
	component := registerGitComponent(t, home)
	legacy := createExecutable(t, filepath.Join(home, "legacy"), "zyr.exe")
	writeTestJSON(t, filepath.Join(home, "legacy.json"), LegacyConfig{SchemaVersion: 1, Executables: []string{legacy}})
	runner := &fakeRunner{}
	dispatcher, _, _ := testDispatcher(t, home, runner)
	if code := dispatcher.Run([]string{"git", "commit"}); code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if runner.executable != component || !reflect.DeepEqual(runner.args, []string{"commit"}) {
		t.Fatalf("wrong dispatch: %s %v", runner.executable, runner.args)
	}
}

// Scenario 5: root help lists the Git component.
func TestRootHelpListsGit(t *testing.T) {
	home := t.TempDir()
	registerGitComponent(t, home)
	dispatcher, output, _ := testDispatcher(t, home, &fakeRunner{})
	if code := dispatcher.Run([]string{"--help"}); code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if !strings.Contains(output.String(), "git") || !strings.Contains(output.String(), "Componente Git") {
		t.Fatalf("help does not list git: %s", output.String())
	}
}

// Scenario 6: zyr git --help is forwarded as component help.
func TestGitHelpIsForwarded(t *testing.T) {
	home := t.TempDir()
	component := registerGitComponent(t, home)
	runner := &fakeRunner{}
	dispatcher, _, _ := testDispatcher(t, home, runner)
	if code := dispatcher.Run([]string{"git", "--help"}); code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if runner.executable != component || !reflect.DeepEqual(runner.args, []string{"--help"}) {
		t.Fatalf("wrong help dispatch: %s %v", runner.executable, runner.args)
	}
}

// Scenario 2: a single old Zyr remains available for unrelated commands.
func TestSingleLegacyZyrReceivesUnknownRootCommand(t *testing.T) {
	home := t.TempDir()
	legacy := createExecutable(t, filepath.Join(home, "legacy"), "zyr.exe")
	writeTestJSON(t, filepath.Join(home, "legacy.json"), LegacyConfig{SchemaVersion: 1, Executables: []string{legacy}})
	runner := &fakeRunner{}
	dispatcher, _, _ := testDispatcher(t, home, runner)
	if code := dispatcher.Run([]string{"init", "--force"}); code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if runner.executable != legacy || !reflect.DeepEqual(runner.args, []string{"init", "--force"}) {
		t.Fatalf("legacy command was not preserved: %s %v", runner.executable, runner.args)
	}
}

// Required where.exe multi-result behavior: never select a legacy executable
// based on incidental order.
func TestMultipleLegacyZyrExecutablesAreRejectedAsAmbiguous(t *testing.T) {
	home := t.TempDir()
	first := createExecutable(t, filepath.Join(home, "legacy-a"), "zyr.exe")
	second := createExecutable(t, filepath.Join(home, "legacy-b"), "zyr.exe")
	writeTestJSON(t, filepath.Join(home, "legacy.json"), LegacyConfig{SchemaVersion: 1, Executables: []string{second, first}})
	runner := &fakeRunner{}
	dispatcher, _, errorOutput := testDispatcher(t, home, runner)
	if code := dispatcher.Run([]string{"init"}); code == 0 {
		t.Fatal("ambiguous legacy command should fail")
	}
	if runner.called || !strings.Contains(errorOutput.String(), first) || !strings.Contains(errorOutput.String(), second) {
		t.Fatalf("ambiguity was not reported safely: %s", errorOutput.String())
	}
}

func TestDuplicateComponentCommandIsRejected(t *testing.T) {
	home := t.TempDir()
	first := createExecutable(t, filepath.Join(home, "one"), "git-one.exe")
	second := createExecutable(t, filepath.Join(home, "two"), "git-two.exe")
	writeTestJSON(t, filepath.Join(home, "components", "one.json"), Component{SchemaVersion: 1, Command: "git", Executable: first})
	writeTestJSON(t, filepath.Join(home, "components", "two.json"), Component{SchemaVersion: 1, Command: "git", Executable: second})
	dispatcher, _, errorOutput := testDispatcher(t, home, &fakeRunner{})
	if code := dispatcher.Run([]string{"git", "commit"}); code == 0 {
		t.Fatal("duplicate command must fail")
	}
	if !strings.Contains(errorOutput.String(), "duplicados") {
		t.Fatalf("duplicate was not explained: %s", errorOutput.String())
	}
}
