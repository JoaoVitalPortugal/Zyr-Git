package app

import (
	"context"
	"errors"
	"testing"
)

type fakeGHVersion struct {
	version string
	missing bool
}

func (g *fakeGHVersion) Version() (string, error) {
	if g.missing {
		return "", errors.New("gh não encontrado")
	}
	return g.version, nil
}

type fakeGHUpdater struct {
	github    *fakeGHVersion
	updated   bool
	installed bool
}

func (u *fakeGHUpdater) InstallGitHubCLI() error {
	u.installed = true
	u.github.missing = false
	u.github.version = "gh version 2.99.0"
	return nil
}

func (u *fakeGHUpdater) UpdateGitHubCLI() error {
	u.updated = true
	u.github.version = "gh version 2.99.0"
	return nil
}

func TestUpdateGHUpdatesExistingCLIOnlyAfterConfirmation(t *testing.T) {
	github := &fakeGHVersion{version: "gh version 2.98.0"}
	installer := &fakeGHUpdater{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	if err := (UpdateGHApplication{GitHub: github, Installer: installer, UI: ui}).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !installer.updated || installer.installed || !containsMessage(ui.messages, "gh version 2.99.0") {
		t.Fatalf("update result was not reported correctly: %+v %v", installer, ui.messages)
	}
}

func TestUpdateGHDeclineDoesNotChangeCLI(t *testing.T) {
	github := &fakeGHVersion{version: "gh version 2.98.0"}
	installer := &fakeGHUpdater{github: github}
	ui := &fakeUI{confirmations: []bool{false}}
	if err := (UpdateGHApplication{GitHub: github, Installer: installer, UI: ui}).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if installer.updated || installer.installed {
		t.Fatal("declined update changed the CLI")
	}
}

func TestUpdateGHCanInstallMissingCLI(t *testing.T) {
	github := &fakeGHVersion{missing: true}
	installer := &fakeGHUpdater{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	if err := (UpdateGHApplication{GitHub: github, Installer: installer, UI: ui}).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !installer.installed || installer.updated {
		t.Fatal("missing CLI was not installed")
	}
}
