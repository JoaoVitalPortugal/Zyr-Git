package app

import (
	"context"
	"fmt"
)

type GitHubCLIUpdater interface {
	InstallGitHubCLI() error
	UpdateGitHubCLI() error
}

type UpdateGHApplication struct {
	GitHub    interface{ Version() (string, error) }
	Installer GitHubCLIUpdater
	UI        UI
}

func (a UpdateGHApplication) Run(ctx context.Context) error {
	_ = ctx
	a.UI.Banner()

	version, err := a.GitHub.Version()
	if err != nil {
		a.UI.Warning("GitHub CLI não encontrado.")
		confirmed, err := a.UI.Confirm("Deseja instalar o GitHub CLI? [S/N]")
		if err != nil {
			return err
		}
		if !confirmed {
			a.UI.Println("Operação cancelada. Nenhuma alteração foi realizada.")
			return nil
		}
		if err := a.Installer.InstallGitHubCLI(); err != nil {
			return fmt.Errorf("não foi possível instalar o GitHub CLI: %w", err)
		}
	} else {
		a.UI.Println("Versão atual: " + firstLine(version))
		confirmed, err := a.UI.Confirm("Deseja atualizar o GitHub CLI? [S/N]")
		if err != nil {
			return err
		}
		if !confirmed {
			a.UI.Println("Operação cancelada. Nenhuma alteração foi realizada.")
			return nil
		}
		if err := a.Installer.UpdateGitHubCLI(); err != nil {
			return fmt.Errorf("não foi possível atualizar o GitHub CLI: %w", err)
		}
	}

	version, err = a.GitHub.Version()
	if err != nil {
		return fmt.Errorf("o instalador terminou, mas o GitHub CLI ainda não pôde ser executado: %w; abra um novo terminal e confira a instalação", err)
	}
	a.UI.Success("GitHub CLI disponível: " + firstLine(version))
	return nil
}
