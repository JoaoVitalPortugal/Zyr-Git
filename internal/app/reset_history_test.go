package app

import (
	"context"
	"reflect"
	"testing"
)

func TestResetHistoryRefusalDoesNotRunDestructiveGitOperation(t *testing.T) {
	f := newFixture()
	f.ui.confirmations = []bool{false}

	err := (ResetHistoryApplication{Git: f.git, UI: f.ui}).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.git.destructive) != 0 {
		t.Fatalf("refusal executed destructive operations: %v", f.git.destructive)
	}
	if !containsMessage(f.ui.messages, "Nenhuma alteração foi realizada") {
		t.Fatal("cancellation was not reported")
	}
}

func TestResetHistoryConfirmationStartsCompleteResetFlow(t *testing.T) {
	f := newFixture()
	f.ui.confirmations = []bool{true}

	err := (ResetHistoryApplication{Git: f.git, UI: f.ui}).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"orphan:" + resetHistoryBranch,
		"add-all",
		"commit:Initial commit",
		"replace:main",
		"force-push:main",
	}
	if !reflect.DeepEqual(f.git.destructive, expected) {
		t.Fatalf("unexpected reset flow:\nwant %v\n got %v", expected, f.git.destructive)
	}
	if !containsMessage(f.ui.messages, "Histórico Git resetado com sucesso") {
		t.Fatal("successful reset was not reported")
	}
}

func TestResetHistoryOutsideRepositoryStopsBeforeConfirmation(t *testing.T) {
	f := newFixture()
	f.git.repository = false

	err := (ResetHistoryApplication{Git: f.git, UI: f.ui}).Run(context.Background())
	if err == nil {
		t.Fatal("expected a non-repository error")
	}
	if len(f.git.destructive) != 0 {
		t.Fatalf("non-repository flow executed destructive operations: %v", f.git.destructive)
	}
}
