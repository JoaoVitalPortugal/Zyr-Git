package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	githubclient "github.com/JoaoVitalPortugal/zyr-git-commit/internal/github"
)

type GitHub interface {
	GitHubSession
	HasDeleteRepoScope() (bool, error)
	AuthorizeDeleteRepo() error
	Repositories() ([]githubclient.Repository, error)
	DeleteRepository(fullName string) error
}

type DeleteRepoApplication struct {
	GitHub    GitHub
	Installer GitHubCLIInstaller
	UI        UI
}

func (a DeleteRepoApplication) Run(ctx context.Context) error {
	_ = ctx
	a.UI.Banner()

	ready, err := ensureGitHubSession(a.GitHub, a.Installer, a.UI)
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}

	hasDeleteScope, err := a.GitHub.HasDeleteRepoScope()
	if err != nil {
		return err
	}
	if !hasDeleteScope {
		a.UI.Warning("O GitHub exige a permissão especial delete_repo para excluir repositórios.")
		a.UI.Println("Essa permissão permite excluir permanentemente repositórios administrados pela sua conta.")
		accepted, promptErr := a.UI.Confirm("Deseja autorizar essa permissão pelo GitHub CLI agora? [S/N]")
		if promptErr != nil {
			return promptErr
		}
		if !accepted {
			a.UI.Println("Operação cancelada. Nenhuma alteração foi realizada.")
			return nil
		}
		a.UI.Println("O GitHub CLI abrirá o navegador para confirmar a permissão.")
		if err := a.GitHub.AuthorizeDeleteRepo(); err != nil {
			return fmt.Errorf("não foi possível autorizar a permissão delete_repo: %w", err)
		}
		hasDeleteScope, err = a.GitHub.HasDeleteRepoScope()
		if err != nil {
			return err
		}
		if !hasDeleteScope {
			return fmt.Errorf("a autorização terminou, mas a permissão delete_repo não foi concedida")
		}
		a.UI.Success("Permissão delete_repo confirmada.")
	}

	repositories, err := a.GitHub.Repositories()
	if err != nil {
		return err
	}
	if len(repositories) == 0 {
		a.UI.Println("Nenhum repositório acessível foi encontrado.")
		return nil
	}

	a.UI.Println("")
	a.UI.Println("Repositórios disponíveis:")
	for index, repository := range repositories {
		a.UI.Println(fmt.Sprintf("[%d] %s", index+1, repository.FullName))
	}

	selected, cancelled, err := a.selectRepository(repositories)
	if err != nil {
		return err
	}
	if cancelled {
		a.UI.Println("Operação cancelada. Nenhuma alteração foi realizada.")
		return nil
	}

	a.UI.Println("")
	a.UI.Println("Repositório: " + selected.Name)
	a.UI.Println("Proprietário: " + selected.Owner.Login)
	a.UI.Println("Tipo do proprietário: " + ownerType(selected.Owner.Type))
	a.UI.Println("Visibilidade: " + selected.Visibility)
	a.UI.Println("URL: " + selected.URL)
	a.UI.Println("Arquivado: " + yesNo(selected.Archived))
	a.UI.Println("Permissão administrativa: " + yesNo(selected.Permissions.Admin))
	a.UI.Println("")

	if !selected.Permissions.Admin {
		a.UI.Warning("Sua conta não possui permissão administrativa para excluir este repositório.")
		a.UI.Println("Operação cancelada. Nenhuma alteração foi realizada.")
		return nil
	}

	a.UI.Warning("ATENÇÃO: esta operação excluirá permanentemente o repositório remoto do GitHub.")
	a.UI.Println("Todos os commits, branches, tags, issues e configurações remotas desse repositório serão excluídos.")
	a.UI.Println("Os arquivos e o repositório Git local não serão alterados.")
	a.UI.Println("")
	answer, err := a.UI.Prompt(fmt.Sprintf("Digite %q para confirmar a exclusão permanente:", selected.FullName))
	if err != nil {
		return err
	}
	if answer != selected.FullName {
		a.UI.Println("Confirmação incorreta. Operação cancelada sem alterações.")
		return nil
	}

	if err := a.GitHub.DeleteRepository(selected.FullName); err != nil {
		return err
	}
	a.UI.Success("Repositório remoto excluído: " + selected.FullName)
	a.UI.Println("Os arquivos locais não foram alterados.")
	return nil
}

func (a DeleteRepoApplication) selectRepository(repositories []githubclient.Repository) (githubclient.Repository, bool, error) {
	for {
		answer, err := a.UI.Prompt("Escolha o número do repositório (0 para cancelar):")
		if err != nil {
			return githubclient.Repository{}, false, err
		}
		index, err := strconv.Atoi(strings.TrimSpace(answer))
		if err != nil || index < 0 || index > len(repositories) {
			a.UI.Warning(fmt.Sprintf("Digite um número entre 0 e %d.", len(repositories)))
			continue
		}
		if index == 0 {
			return githubclient.Repository{}, true, nil
		}
		return repositories[index-1], false, nil
	}
}

func firstLine(value string) string {
	if line, _, found := strings.Cut(value, "\n"); found {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(value)
}

func ownerType(value string) string {
	switch strings.ToLower(value) {
	case "organization":
		return "organização"
	case "user":
		return "usuário"
	default:
		return value
	}
}

func yesNo(value bool) string {
	if value {
		return "sim"
	}
	return "não"
}
