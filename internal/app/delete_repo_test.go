package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	githubclient "github.com/JoaoVitalPortugal/zyr-git-commit/internal/github"
)

type fakeGitHub struct {
	missing            bool
	authError          error
	loginError         error
	scopeError         error
	authorizeError     error
	missingDeleteScope bool
	keepAuthError      bool
	keepScopeMissing   bool
	listError          error
	deleteError        error
	repositories       []githubclient.Repository
	versionCalls       int
	authCalls          int
	loginCalls         int
	scopeCalls         int
	authorizeCalls     int
	listCalls          int
	deleted            []string
}

func (g *fakeGitHub) Version() (string, error) {
	g.versionCalls++
	if g.missing {
		return "", errors.New("missing")
	}
	return "gh version 2.80.0\nhttps://github.com/cli/cli/releases", nil
}
func (g *fakeGitHub) Authenticated() error {
	g.authCalls++
	return g.authError
}
func (g *fakeGitHub) Login() error {
	g.loginCalls++
	if g.loginError == nil && !g.keepAuthError {
		g.authError = nil
	}
	return g.loginError
}
func (g *fakeGitHub) HasDeleteRepoScope() (bool, error) {
	g.scopeCalls++
	return !g.missingDeleteScope, g.scopeError
}
func (g *fakeGitHub) AuthorizeDeleteRepo() error {
	g.authorizeCalls++
	if g.authorizeError == nil && !g.keepScopeMissing {
		g.missingDeleteScope = false
	}
	return g.authorizeError
}
func (g *fakeGitHub) Repositories() ([]githubclient.Repository, error) {
	g.listCalls++
	return g.repositories, g.listError
}
func (g *fakeGitHub) DeleteRepository(fullName string) error {
	g.deleted = append(g.deleted, fullName)
	return g.deleteError
}

type fakeGitHubInstaller struct {
	github *fakeGitHub
	called bool
	err    error
}

func (i *fakeGitHubInstaller) InstallGitHubCLI() error {
	i.called = true
	if i.err == nil {
		i.github.missing = false
	}
	return i.err
}

func repository(fullName string, admin bool) githubclient.Repository {
	parts := strings.Split(fullName, "/")
	result := githubclient.Repository{
		Name:       parts[1],
		FullName:   fullName,
		Visibility: "private",
		URL:        "https://github.com/" + fullName,
	}
	result.Owner.Login = parts[0]
	result.Owner.Type = "User"
	result.Permissions.Admin = admin
	return result
}

func runDeleteRepo(t *testing.T, github *fakeGitHub, installer *fakeGitHubInstaller, ui *fakeUI) error {
	t.Helper()
	return (DeleteRepoApplication{GitHub: github, Installer: installer, UI: ui}).Run(context.Background())
}

func TestDeleteRepoMissingCLIRefusalDoesNothing(t *testing.T) {
	github := &fakeGitHub{missing: true}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{false}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if installer.called || github.listCalls != 0 || len(github.deleted) != 0 {
		t.Fatalf("refusal changed state: installer=%v list=%d deleted=%v", installer.called, github.listCalls, github.deleted)
	}
}

func TestDeleteRepoInstallsCLIOnlyAfterConsentAndRechecksIt(t *testing.T) {
	github := &fakeGitHub{missing: true}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if !installer.called || github.versionCalls != 2 {
		t.Fatalf("expected consented install and recheck: installer=%v versions=%d", installer.called, github.versionCalls)
	}
}

func TestDeleteRepoUnauthenticatedLoginRefusalDoesNothing(t *testing.T) {
	github := &fakeGitHub{authError: errors.New("not logged in")}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{false}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if github.loginCalls != 0 || github.listCalls != 0 || len(github.deleted) != 0 {
		t.Fatalf("unexpected unauthenticated flow: list=%d deleted=%v messages=%v", github.listCalls, github.deleted, ui.messages)
	}
}

func TestDeleteRepoLogsInOnlyAfterConsentAndRechecksAuthentication(t *testing.T) {
	github := &fakeGitHub{authError: errors.New("not logged in")}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if github.loginCalls != 1 || github.authCalls != 2 || github.listCalls != 1 {
		t.Fatalf("expected login and recheck: login=%d auth=%d list=%d", github.loginCalls, github.authCalls, github.listCalls)
	}
}

func TestDeleteRepoAbortsWhenLoginCannotBeConfirmed(t *testing.T) {
	github := &fakeGitHub{authError: errors.New("not logged in"), keepAuthError: true}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	err := runDeleteRepo(t, github, installer, ui)
	if err == nil || !strings.Contains(err.Error(), "autenticação não pôde ser confirmada") || github.listCalls != 0 {
		t.Fatalf("unexpected unconfirmed login result: err=%v list=%d", err, github.listCalls)
	}
}

func TestDeleteRepoReturnsLoginFlowErrorWithoutListing(t *testing.T) {
	github := &fakeGitHub{authError: errors.New("not logged in"), loginError: errors.New("browser flow failed")}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	err := runDeleteRepo(t, github, installer, ui)
	if err == nil || !strings.Contains(err.Error(), "browser flow failed") || github.listCalls != 0 {
		t.Fatalf("unexpected login failure: err=%v list=%d", err, github.listCalls)
	}
}

func TestDeleteRepoMissingScopeRefusalDoesNothing(t *testing.T) {
	github := &fakeGitHub{missingDeleteScope: true}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{false}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if github.authorizeCalls != 0 || github.listCalls != 0 || len(github.deleted) != 0 {
		t.Fatalf("scope refusal changed state: authorize=%d list=%d deleted=%v", github.authorizeCalls, github.listCalls, github.deleted)
	}
}

func TestDeleteRepoAuthorizesMissingScopeAndRechecksIt(t *testing.T) {
	github := &fakeGitHub{missingDeleteScope: true}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if github.authorizeCalls != 1 || github.scopeCalls != 2 || github.listCalls != 1 {
		t.Fatalf("expected scope authorization and recheck: authorize=%d scope=%d list=%d", github.authorizeCalls, github.scopeCalls, github.listCalls)
	}
}

func TestDeleteRepoAbortsWhenScopeWasNotGranted(t *testing.T) {
	github := &fakeGitHub{missingDeleteScope: true, keepScopeMissing: true}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	err := runDeleteRepo(t, github, installer, ui)
	if err == nil || !strings.Contains(err.Error(), "não foi concedida") || github.listCalls != 0 {
		t.Fatalf("unexpected missing scope result: err=%v list=%d", err, github.listCalls)
	}
}

func TestDeleteRepoReturnsAuthorizationFlowErrorWithoutListing(t *testing.T) {
	github := &fakeGitHub{missingDeleteScope: true, authorizeError: errors.New("authorization failed")}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{confirmations: []bool{true}}
	err := runDeleteRepo(t, github, installer, ui)
	if err == nil || !strings.Contains(err.Error(), "authorization failed") || github.listCalls != 0 {
		t.Fatalf("unexpected authorization failure: err=%v list=%d", err, github.listCalls)
	}
}

func TestDeleteRepoZeroSelectionCancelsWithoutDeletion(t *testing.T) {
	github := &fakeGitHub{repositories: []githubclient.Repository{repository("owner/project", true)}}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{prompts: []string{"0"}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.deleted) != 0 {
		t.Fatalf("cancelled selection deleted repositories: %v", github.deleted)
	}
}

func TestDeleteRepoWithoutAdminPermissionNeverAsksForFinalConfirmation(t *testing.T) {
	github := &fakeGitHub{repositories: []githubclient.Repository{repository("org/project", false)}}
	github.repositories[0].Owner.Type = "Organization"
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{prompts: []string{"1"}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.deleted) != 0 || len(ui.prompts) != 0 {
		t.Fatalf("non-admin flow should stop before confirmation: deleted=%v prompts=%v", github.deleted, ui.prompts)
	}
}

func TestDeleteRepoWrongExactConfirmationDoesNothing(t *testing.T) {
	github := &fakeGitHub{repositories: []githubclient.Repository{repository("owner/project", true)}}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{prompts: []string{"1", "project"}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.deleted) != 0 {
		t.Fatalf("wrong confirmation deleted repositories: %v", github.deleted)
	}
}

func TestDeleteRepoConfirmationWithExtraWhitespaceDoesNothing(t *testing.T) {
	github := &fakeGitHub{repositories: []githubclient.Repository{repository("owner/project", true)}}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{prompts: []string{"1", " owner/project "}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.deleted) != 0 {
		t.Fatalf("non-exact confirmation deleted repositories: %v", github.deleted)
	}
}

func TestDeleteRepoInvalidIndexThenExactConfirmationDeletesOnlySelection(t *testing.T) {
	github := &fakeGitHub{repositories: []githubclient.Repository{
		repository("owner/first", true),
		repository("org/second", true),
	}}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{prompts: []string{"abc", "3", "2", "org/second"}}
	if err := runDeleteRepo(t, github, installer, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.deleted) != 1 || github.deleted[0] != "org/second" {
		t.Fatalf("unexpected deleted repositories: %v", github.deleted)
	}
}

func TestDeleteRepoFailureReturnsErrorAndDoesNotPrintSuccess(t *testing.T) {
	github := &fakeGitHub{
		repositories: []githubclient.Repository{repository("owner/project", true)},
		deleteError:  errors.New("denied"),
	}
	installer := &fakeGitHubInstaller{github: github}
	ui := &fakeUI{prompts: []string{"1", "owner/project"}}
	err := runDeleteRepo(t, github, installer, ui)
	if err == nil || !strings.Contains(err.Error(), "denied") || messagesContain(ui.messages, "Repositório remoto excluído") {
		t.Fatalf("unexpected failure result: err=%v messages=%v", err, ui.messages)
	}
}

func messagesContain(messages []string, expected string) bool {
	for _, message := range messages {
		if strings.Contains(message, expected) {
			return true
		}
	}
	return false
}
