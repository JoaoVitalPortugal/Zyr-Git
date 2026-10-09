package gitignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCreatesGenericTemplateOnlyOnce(t *testing.T) {
	directory := t.TempDir()
	manager := NewManagerAt(directory)
	created, err := manager.Ensure()
	if err != nil || !created {
		t.Fatalf("first ensure failed: created=%v err=%v", created, err)
	}
	data, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{".vscode/", ".vscode-test/", ".history/", "node_modules/", "__pycache__/", ".gradle/", "CMakeFiles/", ".terraform/"} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("generic template is missing %q", expected)
		}
	}
	if strings.Contains(string(data), "!.vscode/") {
		t.Fatal("generated template must ignore all VS Code settings")
	}
	created, err = manager.Ensure()
	if err != nil || created {
		t.Fatalf("second ensure should preserve file: created=%v err=%v", created, err)
	}
}
