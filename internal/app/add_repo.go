package app

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	githubclient "github.com/JoaoVitalPortugal/zyr-git-commit/internal/github"
)

type RepositoryCreator interface {
	GitHubSession
	CurrentUser() (string, error)
	GitignoreTemplates() ([]string, error)
	CreateRepository(options githubclient.CreateRepositoryOptions) (string, error)
}

type AddRepoApplication struct {
	GitHub    RepositoryCreator
	Installer GitHubCLIInstaller
	UI        UI
}

func (a AddRepoApplication) Run(ctx context.Context) error {
	_ = ctx
	a.UI.Banner()

	ready, err := ensureGitHubSession(a.GitHub, a.Installer, a.UI)
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}

	owner, err := a.GitHub.CurrentUser()
	if err != nil {
		return err
	}
	name, err := a.promptRepositoryName()
	if err != nil {
		return err
	}
	description, err := a.promptDescription()
	if err != nil {
		return err
	}
	visibility, err := a.promptVisibility()
	if err != nil {
		return err
	}
	addReadme, err := a.UI.Confirm("Adicionar um README? [S/N]")
	if err != nil {
		return err
	}
	addGitignore, err := a.UI.Confirm("Adicionar um .gitignore? [S/N]")
	if err != nil {
		return err
	}
	gitignore := ""
	if addGitignore {
		templates, err := a.GitHub.GitignoreTemplates()
		if err != nil {
			return err
		}
		if len(templates) == 0 {
			return fmt.Errorf("nenhum template de .gitignore foi encontrado")
		}
		gitignore, err = a.selectGitignoreTemplate(templates)
		if err != nil {
			return err
		}
	}

	a.UI.Println("")
	a.UI.Println("Novo repositório:")
	a.UI.Println("Proprietário: " + owner)
	a.UI.Println("Nome: " + name)
	if description == "" {
		a.UI.Println("Descrição: sem descrição")
	} else {
		a.UI.Println("Descrição: " + description)
	}
	a.UI.Println("Visibilidade: " + visibilityLabel(visibility))
	a.UI.Println("README: " + yesNo(addReadme))
	if gitignore == "" {
		a.UI.Println(".gitignore: não")
	} else {
		a.UI.Println(".gitignore: " + gitignore)
	}
	a.UI.Println("Licença: não")
	a.UI.Println("")

	confirmed, err := a.UI.Confirm("Criar este repositório no GitHub? [S/N]")
	if err != nil {
		return err
	}
	if !confirmed {
		a.UI.Println("Operação cancelada. Nenhum repositório foi criado.")
		return nil
	}

	url, err := a.GitHub.CreateRepository(githubclient.CreateRepositoryOptions{
		Owner:       owner,
		Name:        name,
		Description: description,
		Visibility:  visibility,
		AddReadme:   addReadme,
		Gitignore:   gitignore,
	})
	if err != nil {
		return err
	}
	a.UI.Success("Repositório criado com sucesso.")
	a.UI.Println("URL: " + url)
	a.UI.Println("Nenhum arquivo ou repositório local foi alterado.")
	return nil
}

func (a AddRepoApplication) promptRepositoryName() (string, error) {
	for {
		answer, err := a.UI.Prompt("Nome do repositório:")
		if err != nil {
			return "", err
		}
		name := strings.TrimSpace(answer)
		if githubclient.ValidRepositoryName(name) {
			return name, nil
		}
		a.UI.Warning("Use de 1 a 100 caracteres: letras, números, ponto, hífen ou sublinhado.")
	}
}

func (a AddRepoApplication) promptDescription() (string, error) {
	for {
		answer, err := a.UI.Prompt("Descrição (opcional, até 350 caracteres):")
		if err != nil {
			return "", err
		}
		if utf8.RuneCountInString(answer) <= 350 {
			return answer, nil
		}
		a.UI.Warning("A descrição deve ter no máximo 350 caracteres.")
	}
}

func (a AddRepoApplication) promptVisibility() (string, error) {
	a.UI.Println("")
	a.UI.Println("Escolha a visibilidade:")
	a.UI.Println("[1] Público — qualquer pessoa pode ver")
	a.UI.Println("[2] Privado — somente pessoas autorizadas podem ver")
	for {
		answer, err := a.UI.Prompt("Visibilidade [1/2]:")
		if err != nil {
			return "", err
		}
		switch strings.TrimSpace(answer) {
		case "1":
			return "public", nil
		case "2":
			return "private", nil
		default:
			a.UI.Warning("Digite 1 para público ou 2 para privado.")
		}
	}
}

func (a AddRepoApplication) selectGitignoreTemplate(templates []string) (string, error) {
	byName := make(map[string]string, len(templates))
	for _, template := range templates {
		byName[strings.ToLower(template)] = template
	}
	for {
		answer, err := a.UI.Prompt("Template do .gitignore (ex.: Go, Node, Python; ? para listar; 0 para não adicionar):")
		if err != nil {
			return "", err
		}
		answer = strings.TrimSpace(answer)
		if answer == "0" {
			return "", nil
		}
		if answer == "?" {
			a.printGitignoreTemplates(templates)
			continue
		}
		if canonical, found := byName[strings.ToLower(answer)]; found {
			return canonical, nil
		}
		a.UI.Warning("Template não encontrado. Digite ? para ver as opções.")
	}
}

func (a AddRepoApplication) printGitignoreTemplates(templates []string) {
	a.UI.Println("")
	a.UI.Println("Templates disponíveis:")
	line := ""
	for _, template := range templates {
		candidate := template
		if line != "" {
			candidate = line + ", " + template
		}
		if len(candidate) > 78 && line != "" {
			a.UI.Println(line)
			line = template
		} else {
			line = candidate
		}
	}
	if line != "" {
		a.UI.Println(line)
	}
	a.UI.Println("")
}

func visibilityLabel(visibility string) string {
	if visibility == "public" {
		return "público"
	}
	return "privado"
}
