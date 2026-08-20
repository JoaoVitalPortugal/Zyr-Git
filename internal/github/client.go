package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/command"
)

var (
	ownerPattern       = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
	repositoryPattern  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gitignorePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+._-]*$`)
	deleteScopePattern = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])delete_repo([^A-Za-z0-9_]|$)`)
)

type CreateRepositoryOptions struct {
	Owner       string
	Name        string
	Description string
	Visibility  string
	AddReadme   bool
	Gitignore   string
}

type Repository struct {
	Name       string `json:"name"`
	FullName   string `json:"full_name"`
	Visibility string `json:"visibility"`
	URL        string `json:"html_url"`
	Archived   bool   `json:"archived"`
	Owner      struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"owner"`
	Permissions struct {
		Admin bool `json:"admin"`
	} `json:"permissions"`
}

type Client struct {
	executor command.Executor
}

func New(executor command.Executor) *Client {
	return &Client{executor: executor}
}

func (c *Client) Version() (string, error) {
	return c.run("verificar o GitHub CLI", "--version")
}

func (c *Client) Authenticated() error {
	_, err := c.run("verificar a autenticação do GitHub CLI", "auth", "status", "--active", "--hostname", "github.com")
	return err
}

func (c *Client) Login() error {
	return c.runInteractive("fazer login no GitHub CLI", "auth", "login", "--hostname", "github.com", "--web")
}

func (c *Client) HasDeleteRepoScope() (bool, error) {
	output, err := c.run("verificar a permissão delete_repo", "auth", "status", "--active", "--hostname", "github.com")
	if err != nil {
		return false, err
	}
	return deleteScopePattern.MatchString(output), nil
}

func (c *Client) AuthorizeDeleteRepo() error {
	return c.runInteractive("autorizar a permissão delete_repo", "auth", "refresh", "--hostname", "github.com", "--scopes", "delete_repo")
}

func (c *Client) CurrentUser() (string, error) {
	login, err := c.run("identificar a conta do GitHub", "api", "--hostname", "github.com", "user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	if !ownerPattern.MatchString(login) {
		return "", fmt.Errorf("o GitHub CLI retornou um nome de usuário inválido")
	}
	return login, nil
}

func (c *Client) GitignoreTemplates() ([]string, error) {
	output, err := c.run("listar os templates de .gitignore", "repo", "gitignore", "list")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	templates := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if !gitignorePattern.MatchString(name) {
			return nil, fmt.Errorf("o GitHub CLI retornou um template de .gitignore inválido")
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		templates = append(templates, name)
	}
	sort.Slice(templates, func(i, j int) bool {
		return strings.ToLower(templates[i]) < strings.ToLower(templates[j])
	})
	return templates, nil
}

func (c *Client) CreateRepository(options CreateRepositoryOptions) (string, error) {
	if !ownerPattern.MatchString(options.Owner) || !ValidRepositoryName(options.Name) {
		return "", fmt.Errorf("nome de repositório inválido")
	}
	if utf8.RuneCountInString(options.Description) > 350 || strings.ContainsRune(options.Description, '\x00') {
		return "", fmt.Errorf("a descrição do repositório deve ter no máximo 350 caracteres")
	}
	if options.Visibility != "public" && options.Visibility != "private" {
		return "", fmt.Errorf("visibilidade de repositório inválida")
	}
	if options.Gitignore != "" && !gitignorePattern.MatchString(options.Gitignore) {
		return "", fmt.Errorf("template de .gitignore inválido")
	}

	fullName := options.Owner + "/" + options.Name
	args := []string{"repo", "create", fullName, "--" + options.Visibility}
	if options.Description != "" {
		args = append(args, "--description", options.Description)
	}
	if options.AddReadme {
		args = append(args, "--add-readme")
	}
	if options.Gitignore != "" {
		args = append(args, "--gitignore", options.Gitignore)
	}
	if _, err := c.run("criar o repositório no GitHub", args...); err != nil {
		return "", err
	}
	return "https://github.com/" + fullName, nil
}

func (c *Client) Repositories() ([]Repository, error) {
	output, err := c.run(
		"listar os repositórios do GitHub",
		"api", "--hostname", "github.com", "--paginate", "--slurp",
		"/user/repos?per_page=100&affiliation=owner%2Ccollaborator%2Corganization_member&sort=full_name&direction=asc",
	)
	if err != nil {
		return nil, err
	}

	var pages [][]Repository
	if err := json.Unmarshal([]byte(output), &pages); err != nil {
		return nil, fmt.Errorf("a resposta do GitHub CLI não pôde ser interpretada: %w", err)
	}

	repositories := make([]Repository, 0)
	seen := make(map[string]struct{})
	for _, page := range pages {
		for _, repository := range page {
			if err := validateRepository(repository); err != nil {
				return nil, err
			}
			key := strings.ToLower(repository.FullName)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			repositories = append(repositories, repository)
		}
	}
	sort.Slice(repositories, func(i, j int) bool {
		return strings.ToLower(repositories[i].FullName) < strings.ToLower(repositories[j].FullName)
	})
	return repositories, nil
}

func (c *Client) DeleteRepository(fullName string) error {
	if !validFullName(fullName) {
		return fmt.Errorf("identificador de repositório inválido")
	}
	if _, err := c.run("excluir o repositório do GitHub", "repo", "delete", fullName, "--yes"); err != nil {
		return fmt.Errorf("%w; confirme sua permissão administrativa e, se necessário, execute 'gh auth refresh -s delete_repo'", err)
	}
	return nil
}

func (c *Client) run(operation string, args ...string) (string, error) {
	executable, err := c.executor.LookPath("gh")
	if err != nil {
		return "", fmt.Errorf("GitHub CLI não encontrado")
	}
	output, err := c.executor.CombinedOutput(executable, args...)
	if err != nil {
		return output, fmt.Errorf("%s: %w", operation, err)
	}
	return strings.TrimSpace(output), nil
}

func (c *Client) runInteractive(operation string, args ...string) error {
	executable, err := c.executor.LookPath("gh")
	if err != nil {
		return fmt.Errorf("GitHub CLI não encontrado")
	}
	if err := c.executor.Interactive(executable, args...); err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}

func validateRepository(repository Repository) error {
	if !validFullName(repository.FullName) || repository.Name == "" || repository.Owner.Login == "" {
		return fmt.Errorf("o GitHub CLI retornou um repositório com identificador inválido")
	}
	expected := repository.Owner.Login + "/" + repository.Name
	if !strings.EqualFold(repository.FullName, expected) {
		return fmt.Errorf("o GitHub CLI retornou dados inconsistentes para o repositório %q", repository.FullName)
	}
	switch repository.Visibility {
	case "public", "private", "internal":
	default:
		return fmt.Errorf("o GitHub CLI retornou visibilidade inválida para %q", repository.FullName)
	}
	parsed, err := url.Parse(repository.URL)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "github.com") || !strings.EqualFold(strings.Trim(parsed.Path, "/"), repository.FullName) {
		return fmt.Errorf("o GitHub CLI retornou uma URL inválida para %q", repository.FullName)
	}
	return nil
}

func validFullName(fullName string) bool {
	parts := strings.Split(fullName, "/")
	return len(parts) == 2 && ownerPattern.MatchString(parts[0]) && ValidRepositoryName(parts[1])
}

func ValidRepositoryName(name string) bool {
	return len(name) <= 100 && name != "." && name != ".." && repositoryPattern.MatchString(name)
}
