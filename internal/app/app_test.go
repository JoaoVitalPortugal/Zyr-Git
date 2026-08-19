package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeGit struct {
	version           string
	versionMissing    bool
	config            map[string]string
	repository        bool
	origin            string
	hasOrigin         bool
	changes           bool
	staged            bool
	branch            string
	repositoryName    string
	upstream          bool
	pushError         error
	initCalled        bool
	addOrigin         string
	addCalled         bool
	commitMessage     string
	pushCalled        bool
	setUpstreamCalled bool
	destructive       []string
}

func defaultFakeGit() *fakeGit {
	return &fakeGit{
		version:        "git version 2.50.0",
		config:         map[string]string{"user.name": "Ana", "user.email": "ana@example.com"},
		repository:     true,
		origin:         "https://github.com/example/project.git",
		hasOrigin:      true,
		changes:        true,
		staged:         true,
		branch:         "main",
		repositoryName: "project",
		upstream:       true,
	}
}

func (g *fakeGit) Version() (string, error) {
	if g.versionMissing {
		return "", errors.New("Git não encontrado")
	}
	return g.version, nil
}
func (g *fakeGit) GlobalConfig(key string) (string, error) { return g.config[key], nil }
func (g *fakeGit) SetGlobalConfig(key, value string) error { g.config[key] = value; return nil }
func (g *fakeGit) IsRepository() (bool, error)             { return g.repository, nil }
func (g *fakeGit) Init() error                             { g.repository = true; g.initCalled = true; return nil }
func (g *fakeGit) Origin() (string, bool, error)           { return g.origin, g.hasOrigin, nil }
func (g *fakeGit) AddOrigin(value string) error {
	g.origin, g.hasOrigin, g.addOrigin = value, true, value
	return nil
}
func (g *fakeGit) HasChanges() (bool, error)              { return g.changes, nil }
func (g *fakeGit) AddAll() error                          { g.addCalled = true; return nil }
func (g *fakeGit) HasStagedChanges() (bool, error)        { return g.staged, nil }
func (g *fakeGit) Commit(message string) error            { g.commitMessage = message; return nil }
func (g *fakeGit) CurrentBranch() (string, error)         { return g.branch, nil }
func (g *fakeGit) HasOriginUpstream(string) (bool, error) { return g.upstream, nil }
func (g *fakeGit) Push(_ string, setUpstream bool) error {
	g.pushCalled, g.setUpstreamCalled = true, setUpstream
	return g.pushError
}
func (g *fakeGit) RepositoryName() (string, error) { return g.repositoryName, nil }
func (g *fakeGit) CreateOrphanBranch(name string) error {
	g.destructive = append(g.destructive, "orphan:"+name)
	return nil
}
func (g *fakeGit) AddAllFiles() error {
	g.destructive = append(g.destructive, "add-all")
	return nil
}
func (g *fakeGit) CommitInitial(message string) error {
	g.destructive = append(g.destructive, "commit:"+message)
	return nil
}
func (g *fakeGit) ReplaceCurrentBranch(name string) error {
	g.destructive = append(g.destructive, "replace:"+name)
	return nil
}
func (g *fakeGit) ForcePush(branch string) error {
	g.destructive = append(g.destructive, "force-push:"+branch)
	return nil
}

type fakeInstaller struct {
	git    *fakeGit
	called bool
}

func (i *fakeInstaller) InstallGit() error {
	i.called = true
	i.git.versionMissing = false
	return nil
}

type fakeState struct {
	confirmed bool
	saved     bool
}

func (s *fakeState) AddAllConfirmed() (bool, error) { return s.confirmed, nil }
func (s *fakeState) ConfirmAddAll() error           { s.confirmed, s.saved = true, true; return nil }

type fakeIgnore struct {
	created bool
	called  bool
}

func (i *fakeIgnore) Ensure() (bool, error) { i.called = true; return i.created, nil }

type fakeUI struct {
	prompts       []string
	confirmations []bool
	messages      []string
}

func (u *fakeUI) Banner()          { u.messages = append(u.messages, "banner") }
func (u *fakeUI) Println(v string) { u.messages = append(u.messages, v) }
func (u *fakeUI) Success(v string) { u.messages = append(u.messages, "ok:"+v) }
func (u *fakeUI) Warning(v string) { u.messages = append(u.messages, "warn:"+v) }
func (u *fakeUI) Prompt(string) (string, error) {
	if len(u.prompts) == 0 {
		return "", errors.New("teste sem resposta de prompt")
	}
	answer := u.prompts[0]
	u.prompts = u.prompts[1:]
	return answer, nil
}
func (u *fakeUI) Confirm(string) (bool, error) {
	if len(u.confirmations) == 0 {
		return false, errors.New("teste sem resposta de confirmação")
	}
	answer := u.confirmations[0]
	u.confirmations = u.confirmations[1:]
	return answer, nil
}
func (u *fakeUI) ConfirmExplicit(string) (bool, error) {
	if len(u.confirmations) == 0 {
		return false, errors.New("teste sem resposta de confirmação explícita")
	}
	answer := u.confirmations[0]
	u.confirmations = u.confirmations[1:]
	return answer, nil
}

type fixture struct {
	app       Application
	git       *fakeGit
	installer *fakeInstaller
	state     *fakeState
	ignore    *fakeIgnore
	ui        *fakeUI
}

func newFixture() fixture {
	git := defaultFakeGit()
	installer := &fakeInstaller{git: git}
	state := &fakeState{confirmed: true}
	ignore := &fakeIgnore{}
	ui := &fakeUI{prompts: []string{"commit de teste"}}
	return fixture{
		app:       Application{Git: git, Installer: installer, State: state, Ignore: ignore, UI: ui},
		git:       git,
		installer: installer,
		state:     state,
		ignore:    ignore,
		ui:        ui,
	}
}

func runFixture(t *testing.T, fixture fixture) error {
	t.Helper()
	return fixture.app.Run(context.Background())
}

// Scenario 1: Git installed and project already configured.
func TestConfiguredProject(t *testing.T) {
	f := newFixture()
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if f.git.initCalled || f.git.addOrigin == "" && !f.git.pushCalled {
		t.Fatal("configured repository should commit and push without reconfiguration")
	}
}

// Scenario 2: Git installed without .git.
func TestInitializesMissingRepository(t *testing.T) {
	f := newFixture()
	f.git.repository = false
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if !f.git.initCalled {
		t.Fatal("expected git init")
	}
}

// Scenario 3: Git installed without origin.
func TestConfiguresMissingRemote(t *testing.T) {
	f := newFixture()
	f.git.hasOrigin = false
	f.ui.prompts = []string{"https://github.com/zyr/example.git", "commit"}
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if f.git.addOrigin != "https://github.com/zyr/example.git" {
		t.Fatalf("unexpected origin: %q", f.git.addOrigin)
	}
}

// Scenario 4: Git installed without user.name.
func TestConfiguresMissingUserNameGlobally(t *testing.T) {
	f := newFixture()
	f.git.config["user.name"] = ""
	f.ui.prompts = []string{"João", "commit"}
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if f.git.config["user.name"] != "João" {
		t.Fatal("user.name was not configured")
	}
}

// Scenario 5: Git installed without user.email.
func TestConfiguresMissingUserEmailGlobally(t *testing.T) {
	f := newFixture()
	f.git.config["user.email"] = ""
	f.ui.prompts = []string{"joao@example.com", "commit"}
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if f.git.config["user.email"] != "joao@example.com" {
		t.Fatal("user.email was not configured")
	}
}

// Scenario 6: Git missing.
func TestInstallsMissingGitAndRechecks(t *testing.T) {
	f := newFixture()
	f.git.versionMissing = true
	f.ui.confirmations = []bool{true}
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if !f.installer.called || f.git.versionMissing {
		t.Fatal("Git installer was not called and rechecked")
	}
}

// Scenario 7: missing .gitignore.
func TestCreatesMissingGitignore(t *testing.T) {
	f := newFixture()
	f.ignore.created = true
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if !f.ignore.called || !containsMessage(f.ui.messages, ".gitignore criado") {
		t.Fatal("missing .gitignore was not created/reported")
	}
}

// Scenario 8: no changes.
func TestStopsWhenThereAreNoChanges(t *testing.T) {
	f := newFixture()
	f.git.changes = false
	f.ui.prompts = nil
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if f.git.addCalled || f.git.commitMessage != "" || f.git.pushCalled {
		t.Fatal("no-change flow must not add, commit, or push")
	}
}

// Scenario 9: normal commit.
func TestNormalCommitUsesMessageAndExistingUpstream(t *testing.T) {
	f := newFixture()
	f.ui.prompts = []string{"corrigi o sistema de tickets"}
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if f.git.commitMessage != "corrigi o sistema de tickets" || f.git.setUpstreamCalled {
		t.Fatal("normal commit did not preserve its message/upstream")
	}
}

// Scenario 10: push failure must be real and observable after commit.
func TestPushFailureIsReturned(t *testing.T) {
	f := newFixture()
	f.git.pushError = errors.New("remote rejected")
	err := runFixture(t, f)
	if err == nil || !strings.Contains(err.Error(), "remote rejected") {
		t.Fatalf("expected real push error, got %v", err)
	}
	if f.git.commitMessage == "" {
		t.Fatal("commit should already exist when push fails")
	}
}

func TestFirstPushSetsOriginUpstream(t *testing.T) {
	f := newFixture()
	f.git.upstream = false
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if !f.git.setUpstreamCalled {
		t.Fatal("first push should configure origin upstream")
	}
}

func TestRejectsCredentialInHTTPSRemote(t *testing.T) {
	f := newFixture()
	f.git.hasOrigin = false
	f.ui.prompts = []string{"https://token@github.com/zyr/example.git", "https://github.com/zyr/example.git", "commit"}
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	if f.git.addOrigin != "https://github.com/zyr/example.git" {
		t.Fatal("credential-bearing remote should have been rejected")
	}
}

func TestExistingCredentialIsRedactedFromTerminal(t *testing.T) {
	f := newFixture()
	f.git.origin = "https://user:secret@github.com/zyr/example.git"
	if err := runFixture(t, f); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(f.ui.messages, "\n")
	if strings.Contains(joined, "secret") || !strings.Contains(joined, "REDACTED") {
		t.Fatal("existing credentials must never be printed")
	}
}

func containsMessage(messages []string, part string) bool {
	for _, message := range messages {
		if strings.Contains(message, part) {
			return true
		}
	}
	return false
}
