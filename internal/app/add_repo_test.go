package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	githubclient "github.com/JoaoVitalPortugal/zyr-git-commit/internal/github"
)

type fakeRepositoryCreator struct {
	missing        bool
	authError      error
	loginError     error
	owner          string
	templates      []string
	templatesError error
	createError    error
	created        []githubclient.CreateRepositoryOptions
}

func (g *fakeRepositoryCreator) Version() (string, error) {
	if g.missing {
		return "", errors.New("missing")
	}
	return "gh version 2.97.0", nil
}
func (g *fakeRepositoryCreator) Authenticated() error { return g.authError }
func (g *fakeRepositoryCreator) Login() error {
	if g.loginError == nil {
		g.authError = nil
	}
	return g.loginError
}
func (g *fakeRepositoryCreator) CurrentUser() (string, error) { return g.owner, nil }
func (g *fakeRepositoryCreator) GitignoreTemplates() ([]string, error) {
	return g.templates, g.templatesError
}
func (g *fakeRepositoryCreator) CreateRepository(options githubclient.CreateRepositoryOptions) (string, error) {
	g.created = append(g.created, options)
	if g.createError != nil {
		return "", g.createError
	}
	return "https://github.com/" + options.Owner + "/" + options.Name, nil
}

type fakeAddRepoInstaller struct {
	github *fakeRepositoryCreator
	called bool
}

func (i *fakeAddRepoInstaller) InstallGitHubCLI() error {
	i.called = true
	i.github.missing = false
	return nil
}

func runAddRepo(t *testing.T, github *fakeRepositoryCreator, ui *fakeUI) error {
	t.Helper()
	installer := &fakeAddRepoInstaller{github: github}
	return (AddRepoApplication{GitHub: github, Installer: installer, UI: ui}).Run(context.Background())
}

func TestAddRepoCreatesConfiguredRepository(t *testing.T) {
	github := &fakeRepositoryCreator{owner: "JoaoVitalPortugal", templates: []string{"Go", "Node", "Python"}}
	ui := &fakeUI{
		prompts:       []string{"new-project", "Descrição do projeto", "1", "go"},
		confirmations: []bool{true, true, true},
	}
	if err := runAddRepo(t, github, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.created) != 1 {
		t.Fatalf("expected one repository, got %d", len(github.created))
	}
	want := githubclient.CreateRepositoryOptions{
		Owner:       "JoaoVitalPortugal",
		Name:        "new-project",
		Description: "Descrição do projeto",
		Visibility:  "public",
		AddReadme:   true,
		Gitignore:   "Go",
	}
	if github.created[0] != want {
		t.Fatalf("unexpected options: want %#v, got %#v", want, github.created[0])
	}
	if !messagesContain(ui.messages, "Nenhum arquivo ou repositório local foi alterado") {
		t.Fatalf("local safety message missing: %v", ui.messages)
	}
}

func TestAddRepoSupportsPrivateEmptyRepository(t *testing.T) {
	github := &fakeRepositoryCreator{owner: "owner"}
	ui := &fakeUI{
		prompts:       []string{"empty-repo", "", "2"},
		confirmations: []bool{false, false, true},
	}
	if err := runAddRepo(t, github, ui); err != nil {
		t.Fatal(err)
	}
	created := github.created[0]
	if created.Visibility != "private" || created.Description != "" || created.AddReadme || created.Gitignore != "" {
		t.Fatalf("unexpected empty repository options: %#v", created)
	}
}

func TestAddRepoValidatesNameDescriptionAndVisibility(t *testing.T) {
	github := &fakeRepositoryCreator{owner: "owner"}
	ui := &fakeUI{
		prompts: []string{
			"invalid name", "valid-name",
			strings.Repeat("á", 351), "Descrição válida",
			"3", "2",
		},
		confirmations: []bool{false, false, true},
	}
	if err := runAddRepo(t, github, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.created) != 1 || github.created[0].Name != "valid-name" || github.created[0].Description != "Descrição válida" || github.created[0].Visibility != "private" {
		t.Fatalf("unexpected validated options: %#v", github.created)
	}
}

func TestAddRepoListsAndValidatesGitignoreTemplate(t *testing.T) {
	github := &fakeRepositoryCreator{owner: "owner", templates: []string{"Go", "Python"}}
	ui := &fakeUI{
		prompts:       []string{"project", "", "1", "?", "unknown", "python"},
		confirmations: []bool{false, true, true},
	}
	if err := runAddRepo(t, github, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.created) != 1 || github.created[0].Gitignore != "Python" || !messagesContain(ui.messages, "Templates disponíveis") {
		t.Fatalf("unexpected template selection: created=%#v messages=%v", github.created, ui.messages)
	}
}

func TestAddRepoFinalCancellationDoesNotCreateRepository(t *testing.T) {
	github := &fakeRepositoryCreator{owner: "owner"}
	ui := &fakeUI{
		prompts:       []string{"project", "", "1"},
		confirmations: []bool{false, false, false},
	}
	if err := runAddRepo(t, github, ui); err != nil {
		t.Fatal(err)
	}
	if len(github.created) != 0 {
		t.Fatalf("cancelled flow created repositories: %#v", github.created)
	}
}

func TestAddRepoCreateFailureDoesNotPrintSuccess(t *testing.T) {
	github := &fakeRepositoryCreator{owner: "owner", createError: errors.New("already exists")}
	ui := &fakeUI{
		prompts:       []string{"project", "", "1"},
		confirmations: []bool{false, false, true},
	}
	err := runAddRepo(t, github, ui)
	if err == nil || !strings.Contains(err.Error(), "already exists") || messagesContain(ui.messages, "Repositório criado com sucesso") {
		t.Fatalf("unexpected create failure: err=%v messages=%v", err, ui.messages)
	}
}
