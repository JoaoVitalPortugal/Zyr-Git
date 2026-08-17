package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/launcher"
)

func TestLauncherIsPrioritizedWithoutRemovingOtherZyr(t *testing.T) {
	launcher := `C:\Users\Ana\AppData\Local\Programs\Zyr CLI\bin`
	oldComponent := `C:\Users\Ana\AppData\Local\Programs\Zyr Git Commit`
	otherZyr := `C:\Users\Ana\AppData\Local\Programs\Zyr`
	original := stringsJoin(`C:\Windows`, otherZyr, oldComponent, `C:\Program Files\Git\cmd`)
	updated := prioritizePathEntry(original, launcher, oldComponent)
	expected := stringsJoin(launcher, `C:\Windows`, otherZyr, `C:\Program Files\Git\cmd`)
	if updated != expected {
		t.Fatalf("unexpected prioritized PATH:\nwant %q\n got %q", expected, updated)
	}
	if restored := removePathEntry(updated, launcher); restored != stringsJoin(`C:\Windows`, otherZyr, `C:\Program Files\Git\cmd`) {
		t.Fatalf("other Zyr path was not preserved: %q", restored)
	}
}

func stringsJoin(parts ...string) string {
	result := ""
	for index, part := range parts {
		if index > 0 {
			result += ";"
		}
		result += part
	}
	return result
}

func TestParseWhereOutputKeepsMultipleDeterministicCandidates(t *testing.T) {
	output := "C:\\Apps\\ZyrOld\\zyr.exe\r\nC:\\Apps\\ZyrNew\\zyr.exe\r\nC:\\Apps\\ZyrOld\\zyr.exe\r\n"
	paths := parseWhereOutput(output)
	expected := []string{`C:\Apps\ZyrOld\zyr.exe`, `C:\Apps\ZyrNew\zyr.exe`}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected where.exe parse: %v", paths)
	}
}

func TestUnknownExecutableAtSharedLauncherIsNeverOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zyr.exe")
	if err := os.WriteFile(path, []byte("not a launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := verifyOwnedExecutable(path, "--zyr-launcher-protocol", launcher.Protocol); err == nil {
		t.Fatal("unrecognized executable should block installation")
	}
}

func TestComponentUpdatePreservesOtherManifest(t *testing.T) {
	directory := t.TempDir()
	gitPath := filepath.Join(directory, "git.json")
	initPath := filepath.Join(directory, "init.json")
	writeManifest := func(path string, manifest launcher.Component) {
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest(initPath, launcher.Component{SchemaVersion: 1, Command: "init", Executable: `C:\Apps\init.exe`})
	if err := ensureNoGitManifestConflict(directory, gitPath); err != nil {
		t.Fatal(err)
	}
	writeManifest(gitPath, launcher.Component{SchemaVersion: 1, Command: "git", Version: "1.0.0", Executable: `C:\Apps\git.exe`})
	writeManifest(gitPath, launcher.Component{SchemaVersion: 1, Command: "git", Version: "2.0.0", Executable: `C:\Apps\git.exe`})
	if count, err := componentManifestCount(directory); err != nil || count != 2 {
		t.Fatalf("component update removed another component: count=%d err=%v", count, err)
	}
	if _, err := os.Stat(initPath); err != nil {
		t.Fatal("other component manifest was removed")
	}
}

func TestRemovingGitManifestLeavesOtherComponent(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "git.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "init.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(directory, "git.json")); err != nil {
		t.Fatal(err)
	}
	if count, err := componentManifestCount(directory); err != nil || count != 1 {
		t.Fatalf("other component should keep launcher installed: count=%d err=%v", count, err)
	}
}

func TestDetectSetupState(t *testing.T) {
	directory := t.TempDir()
	manifestPath := filepath.Join(directory, "git.json")

	state, err := detectSetupState(manifestPath, false, "2.0.0")
	if err != nil || state.Mode != modeInstall {
		t.Fatalf("expected installation, got %+v err=%v", state, err)
	}
	state, err = detectSetupState(manifestPath, true, "2.0.0")
	if err != nil || state.Mode != modeUpdate || state.InstalledVersion != "versão monolítica anterior" {
		t.Fatalf("expected monolith update, got %+v err=%v", state, err)
	}

	manifest := launcher.Component{SchemaVersion: 1, Command: "git", Version: "1.5.0", Owner: productName}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	state, err = detectSetupState(manifestPath, false, "2.0.0")
	if err != nil || state.Mode != modeUpdate || state.InstalledVersion != "1.5.0" {
		t.Fatalf("expected update, got %+v err=%v", state, err)
	}
	state, err = detectSetupState(manifestPath, false, "1.5.0")
	if err != nil || state.Mode != modeRepair {
		t.Fatalf("expected repair, got %+v err=%v", state, err)
	}
}

func TestSetupNoticesRequireExplicitConfirmation(t *testing.T) {
	options := setupOptions{installDir: `C:\Apps\Zyr Git Commit`, sharedHome: `C:\Apps\Zyr CLI`}
	tests := []struct {
		name      string
		state     setupState
		answer    string
		confirmed bool
		expected  string
	}{
		{name: "install", state: setupState{Mode: modeInstall, TargetVersion: "2.0.0"}, answer: "s\n", confirmed: true, expected: "INSTALAÇÃO"},
		{name: "update", state: setupState{Mode: modeUpdate, InstalledVersion: "1.0.0", TargetVersion: "2.0.0"}, answer: "sim\n", confirmed: true, expected: "ATUALIZAÇÃO"},
		{name: "repair cancelled", state: setupState{Mode: modeRepair, InstalledVersion: "2.0.0", TargetVersion: "2.0.0"}, answer: "n\n", confirmed: false, expected: "REPARO"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			confirmed, err := showSetupNotice(options, test.state, strings.NewReader(test.answer), &output)
			if err != nil {
				t.Fatal(err)
			}
			if confirmed != test.confirmed || !strings.Contains(output.String(), test.expected) || !strings.Contains(output.String(), "Outros executáveis") {
				t.Fatalf("unexpected notice: confirmed=%v output=%s", confirmed, output.String())
			}
			if !confirmed && !strings.Contains(output.String(), "Nenhum arquivo foi alterado") {
				t.Fatal("cancellation must explicitly guarantee no changes")
			}
		})
	}
}

func TestDelayedDeleteRemovesValidatedTemporaryFile(t *testing.T) {
	file, err := os.CreateTemp("", "zyr-git-commit-uninstall-*.exe")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if !launchDelayedSelfDelete(path) {
		t.Fatal("failed to launch delayed delete")
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("temporary file was not removed: %s", path)
}
