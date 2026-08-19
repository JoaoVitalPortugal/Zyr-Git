package app

import (
	"context"
	"errors"
	"fmt"
)

const resetHistoryBranch = "zyr-reset-history"

type ResetHistoryApplication struct {
	Git Git
	UI  UI
}

func (a ResetHistoryApplication) Run(ctx context.Context) error {
	_ = ctx
	a.UI.Banner()

	isRepository, err := a.Git.IsRepository()
	if err != nil {
		return err
	}
	if !isRepository {
		return errors.New("o diretório atual não é um repositório Git")
	}

	repository, err := a.Git.RepositoryName()
	if err != nil {
		return err
	}
	branch, err := a.Git.CurrentBranch()
	if err != nil {
		return err
	}
	remote, found, err := a.Git.Origin()
	if err != nil {
		return err
	}
	if !found {
		return errors.New("o repositório não possui um remote \"origin\"")
	}

	a.UI.Warning("ATENÇÃO: esta operação substituirá permanentemente o histórico Git.")
	a.UI.Println("")
	a.UI.Println("Repositório: " + repository)
	a.UI.Println("Branch: " + branch)
	a.UI.Println("Remote: " + redactRemote(remote))
	a.UI.Println("")
	a.UI.Println("Todos os commits anteriores desta branch serão substituídos por um novo commit inicial.")
	a.UI.Println("")

	confirmed, err := a.UI.ConfirmExplicit("Continuar? [S/N]")
	if err != nil {
		return err
	}
	if !confirmed {
		a.UI.Println("Operação cancelada. Nenhuma alteração foi realizada.")
		return nil
	}

	if err := a.Git.CreateOrphanBranch(resetHistoryBranch); err != nil {
		return fmt.Errorf("falha ao criar a branch órfã: %w", err)
	}
	if err := a.Git.AddAllFiles(); err != nil {
		return fmt.Errorf("falha ao adicionar o estado atual dos arquivos: %w", err)
	}
	if err := a.Git.CommitInitial("Initial commit"); err != nil {
		return fmt.Errorf("falha ao criar o novo commit inicial: %w", err)
	}
	if err := a.Git.ReplaceCurrentBranch(branch); err != nil {
		return fmt.Errorf("falha ao substituir a branch %q: %w", branch, err)
	}
	if err := a.Git.ForcePush(branch); err != nil {
		return fmt.Errorf("push forçado falhou: %w", err)
	}

	a.UI.Success("Histórico Git resetado com sucesso.")
	a.UI.Success("Push forçado realizado com sucesso.")
	return nil
}
