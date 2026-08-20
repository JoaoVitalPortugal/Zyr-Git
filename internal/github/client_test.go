package github

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type recordingExecutor struct {
	path             string
	output           string
	err              error
	interactiveError error
	calls            [][]string
	interactiveCalls [][]string
	lookups          []string
}

func (e *recordingExecutor) CombinedOutput(name string, args ...string) (string, error) {
	e.calls = append(e.calls, append([]string{name}, args...))
	return e.output, e.err
}
func (e *recordingExecutor) Interactive(name string, args ...string) error {
	e.interactiveCalls = append(e.interactiveCalls, append([]string{name}, args...))
	return e.interactiveError
}
func (e *recordingExecutor) LookPath(name string) (string, error) {
	e.lookups = append(e.lookups, name)
	if e.path == "" {
		return "", errors.New("not found")
	}
	return e.path, nil
}

func TestRepositoriesUsesAuthenticatedPaginatedAPIAndSortsResults(t *testing.T) {
	executor := &recordingExecutor{
		path:   "gh",
		output: `[[{"name":"repo","full_name":"org/repo","visibility":"internal","html_url":"https://github.com/org/repo","archived":true,"owner":{"login":"org","type":"Organization"},"permissions":{"admin":true}}],[{"name":"alpha","full_name":"ana/alpha","visibility":"private","html_url":"https://github.com/ana/alpha","archived":false,"owner":{"login":"ana","type":"User"},"permissions":{"admin":false}}]]`,
	}
	repositories, err := New(executor).Repositories()
	if err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 2 || repositories[0].FullName != "ana/alpha" || repositories[1].FullName != "org/repo" {
		t.Fatalf("unexpected repositories: %#v", repositories)
	}
	if !repositories[1].Permissions.Admin || repositories[1].Owner.Type != "Organization" {
		t.Fatalf("organization metadata was not preserved: %#v", repositories[1])
	}
	want := []string{"gh", "api", "--hostname", "github.com", "--paginate", "--slurp", "/user/repos?per_page=100&affiliation=owner%2Ccollaborator%2Corganization_member&sort=full_name&direction=asc"}
	if !reflect.DeepEqual(executor.calls[0], want) {
		t.Fatalf("unexpected gh call:\nwant %v\n got %v", want, executor.calls[0])
	}
}

func TestRepositoriesRejectsInconsistentAPIData(t *testing.T) {
	executor := &recordingExecutor{
		path:   "gh",
		output: `[[{"name":"other","full_name":"org/repo","visibility":"public","html_url":"https://github.com/org/repo","owner":{"login":"org","type":"Organization"},"permissions":{"admin":true}}]]`,
	}
	_, err := New(executor).Repositories()
	if err == nil || !strings.Contains(err.Error(), "inconsistentes") {
		t.Fatalf("expected inconsistent data error, got %v", err)
	}
}

func TestDeleteRepositoryUsesOnlyValidatedFullName(t *testing.T) {
	executor := &recordingExecutor{path: "gh"}
	if err := New(executor).DeleteRepository("owner/project"); err != nil {
		t.Fatal(err)
	}
	want := []string{"gh", "repo", "delete", "owner/project", "--yes"}
	if !reflect.DeepEqual(executor.calls[0], want) {
		t.Fatalf("unexpected delete call: %v", executor.calls[0])
	}
}

func TestDeleteRepositoryRejectsCommandLikeInputWithoutExecution(t *testing.T) {
	executor := &recordingExecutor{path: "gh"}
	err := New(executor).DeleteRepository("--repo/evil")
	if err == nil || len(executor.calls) != 0 {
		t.Fatalf("invalid identifier should not execute gh: err=%v calls=%v", err, executor.calls)
	}
}

func TestDeleteFailureIncludesPermissionRecoveryWithoutClaimingSuccess(t *testing.T) {
	executor := &recordingExecutor{path: "gh", err: errors.New("HTTP 403")}
	err := New(executor).DeleteRepository("owner/project")
	if err == nil || !strings.Contains(err.Error(), "gh auth refresh -s delete_repo") || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("unexpected deletion error: %v", err)
	}
}

func TestAuthenticationUsesActiveGitHubAccount(t *testing.T) {
	executor := &recordingExecutor{path: "gh"}
	if err := New(executor).Authenticated(); err != nil {
		t.Fatal(err)
	}
	want := []string{"gh", "auth", "status", "--active", "--hostname", "github.com"}
	if !reflect.DeepEqual(executor.calls[0], want) {
		t.Fatalf("unexpected auth call: %v", executor.calls[0])
	}
}

func TestLoginUsesOfficialBrowserFlow(t *testing.T) {
	executor := &recordingExecutor{path: "gh"}
	if err := New(executor).Login(); err != nil {
		t.Fatal(err)
	}
	want := []string{"gh", "auth", "login", "--hostname", "github.com", "--web"}
	if !reflect.DeepEqual(executor.interactiveCalls[0], want) {
		t.Fatalf("unexpected login call: %v", executor.interactiveCalls[0])
	}
}

func TestDeleteRepoScopeIsDetectedWithoutShowingToken(t *testing.T) {
	executor := &recordingExecutor{path: "gh", output: "Token scopes: 'gist', 'read:org', 'repo', 'delete_repo'"}
	hasScope, err := New(executor).HasDeleteRepoScope()
	if err != nil || !hasScope {
		t.Fatalf("scope should be detected: present=%v err=%v", hasScope, err)
	}
	want := []string{"gh", "auth", "status", "--active", "--hostname", "github.com"}
	if !reflect.DeepEqual(executor.calls[0], want) {
		t.Fatalf("unexpected scope check: %v", executor.calls[0])
	}
	for _, argument := range executor.calls[0] {
		if argument == "--show-token" {
			t.Fatal("scope check must never request the authentication token")
		}
	}
}

func TestMissingDeleteRepoScopeIsReported(t *testing.T) {
	executor := &recordingExecutor{path: "gh", output: "Token scopes: 'gist', 'read:org', 'repo'"}
	hasScope, err := New(executor).HasDeleteRepoScope()
	if err != nil || hasScope {
		t.Fatalf("scope should be missing: present=%v err=%v", hasScope, err)
	}
}

func TestDeleteRepoAuthorizationUsesOfficialRefreshFlow(t *testing.T) {
	executor := &recordingExecutor{path: "gh"}
	if err := New(executor).AuthorizeDeleteRepo(); err != nil {
		t.Fatal(err)
	}
	want := []string{"gh", "auth", "refresh", "--hostname", "github.com", "--scopes", "delete_repo"}
	if !reflect.DeepEqual(executor.interactiveCalls[0], want) {
		t.Fatalf("unexpected authorization call: %v", executor.interactiveCalls[0])
	}
}
