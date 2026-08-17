package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/command"
)

type Client struct {
	executor command.Executor
}

func New(executor command.Executor) *Client {
	return &Client{executor: executor}
}

func (c *Client) Version() (string, error) {
	return c.run("verificar o Git", "--version")
}

func (c *Client) GlobalConfig(key string) (string, error) {
	out, err := c.run("ler a configuração global "+key, "config", "--global", "--get", key)
	if code, ok := command.ExitCode(err); ok && code == 1 {
		return "", nil
	}
	return strings.TrimSpace(out), err
}

func (c *Client) SetGlobalConfig(key, value string) error {
	_, err := c.run("salvar a configuração global "+key, "config", "--global", key, value)
	return err
}

func (c *Client) IsRepository() (bool, error) {
	out, err := c.run("verificar o repositório", "rev-parse", "--is-inside-work-tree")
	if err == nil {
		return strings.EqualFold(strings.TrimSpace(out), "true"), nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "not a git repository") || strings.Contains(message, "não é um repositório git") {
		return false, nil
	}
	return false, err
}

func (c *Client) Init() error {
	_, err := c.run("inicializar o repositório", "init")
	return err
}

func (c *Client) Origin() (string, bool, error) {
	out, err := c.run("listar remotes", "remote")
	if err != nil {
		return "", false, err
	}
	found := false
	for _, name := range strings.Fields(out) {
		if name == "origin" {
			found = true
			break
		}
	}
	if !found {
		return "", false, nil
	}

	url, err := c.run("ler o remote origin", "remote", "get-url", "origin")
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(url), true, nil
}

func (c *Client) AddOrigin(url string) error {
	_, err := c.run("adicionar o remote origin", "remote", "add", "origin", url)
	return err
}

func (c *Client) HasChanges() (bool, error) {
	out, err := c.run("verificar alterações", "status", "--porcelain=v1", "--untracked-files=all", "--", ".")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (c *Client) AddAll() error {
	_, err := c.run("adicionar arquivos", "add", ".")
	return err
}

func (c *Client) HasStagedChanges() (bool, error) {
	_, err := c.run("verificar arquivos adicionados", "diff", "--cached", "--quiet", "--", ".")
	if err == nil {
		return false, nil
	}
	if code, ok := command.ExitCode(err); ok && code == 1 {
		return true, nil
	}
	return false, err
}

func (c *Client) Commit(message string) error {
	_, err := c.run("criar o commit", "commit", "-m", message, "--", ".")
	return err
}

func (c *Client) CurrentBranch() (string, error) {
	out, err := c.run("identificar a branch atual", "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		return "", fmt.Errorf("HEAD destacado: não foi possível identificar uma branch para o push")
	}
	return branch, nil
}

func (c *Client) HasOriginUpstream(branch string) (bool, error) {
	remote, err := c.configValue("branch." + branch + ".remote")
	if err != nil {
		return false, err
	}
	merge, err := c.configValue("branch." + branch + ".merge")
	if err != nil {
		return false, err
	}
	return remote == "origin" && merge == "refs/heads/"+branch, nil
}

func (c *Client) Push(branch string, setUpstream bool) error {
	args := []string{"push"}
	if setUpstream {
		args = append(args, "--set-upstream")
	}
	args = append(args, "origin", branch)
	_, err := c.run("enviar para o remote origin", args...)
	return err
}

func (c *Client) GitPath(name string) (string, error) {
	out, err := c.run("localizar os metadados do Zyr", "rev-parse", "--git-path", name)
	if err != nil {
		return "", err
	}
	path := filepath.Clean(strings.TrimSpace(out))
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Abs(path)
}

func (c *Client) configValue(key string) (string, error) {
	out, err := c.run("ler a configuração da branch", "config", "--get", key)
	if code, ok := command.ExitCode(err); ok && code == 1 {
		return "", nil
	}
	return strings.TrimSpace(out), err
}

func (c *Client) run(operation string, args ...string) (string, error) {
	executable, err := c.executor.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("Git não encontrado")
	}
	out, err := c.executor.CombinedOutput(executable, args...)
	if err != nil {
		return out, fmt.Errorf("%s: %w", operation, err)
	}
	return out, nil
}
