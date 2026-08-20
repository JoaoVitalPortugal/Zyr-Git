package app

import "fmt"

type GitHubSession interface {
	Version() (string, error)
	Authenticated() error
	Login() error
}

type GitHubCLIInstaller interface {
	InstallGitHubCLI() error
}

func ensureGitHubSession(client GitHubSession, installer GitHubCLIInstaller, ui UI) (bool, error) {
	version, err := client.Version()
	if err != nil {
		ui.Warning("GitHub CLI não encontrado.")
		accepted, promptErr := ui.Confirm("Deseja instalar o GitHub CLI automaticamente? [S/N]")
		if promptErr != nil {
			return false, promptErr
		}
		if !accepted {
			ui.Println("Operação cancelada. Nenhuma alteração foi realizada.")
			return false, nil
		}
		if err := installer.InstallGitHubCLI(); err != nil {
			return false, fmt.Errorf("não foi possível instalar o GitHub CLI: %w", err)
		}
		version, err = client.Version()
		if err != nil {
			return false, fmt.Errorf("o instalador terminou, mas o GitHub CLI ainda não pôde ser executado: %w", err)
		}
	}
	ui.Success("GitHub CLI encontrado: " + firstLine(version))

	if err := client.Authenticated(); err != nil {
		ui.Warning("O GitHub CLI não está autenticado em github.com.")
		accepted, promptErr := ui.Confirm("Deseja entrar na sua conta pelo GitHub CLI agora? [S/N]")
		if promptErr != nil {
			return false, promptErr
		}
		if !accepted {
			ui.Println("Operação cancelada. Nenhuma alteração foi realizada.")
			return false, nil
		}
		ui.Println("O GitHub CLI abrirá o navegador para realizar o login com segurança.")
		if err := client.Login(); err != nil {
			return false, fmt.Errorf("não foi possível entrar no GitHub: %w", err)
		}
		if err := client.Authenticated(); err != nil {
			return false, fmt.Errorf("o login terminou, mas a autenticação não pôde ser confirmada: %w", err)
		}
		ui.Success("Login no GitHub confirmado.")
	}
	return true, nil
}
