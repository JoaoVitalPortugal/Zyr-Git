package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type Git interface {
	Version() (string, error)
	GlobalConfig(key string) (string, error)
	SetGlobalConfig(key, value string) error
	IsRepository() (bool, error)
	Init() error
	Origin() (url string, found bool, err error)
	AddOrigin(url string) error
	HasChanges() (bool, error)
	AddAll() error
	HasStagedChanges() (bool, error)
	Commit(message string) error
	CurrentBranch() (string, error)
	HasOriginUpstream(branch string) (bool, error)
	Push(branch string, setUpstream bool) error
	RepositoryName() (string, error)
	CreateOrphanBranch(name string) error
	AddAllFiles() error
	CommitInitial(message string) error
	ReplaceCurrentBranch(name string) error
	ForcePush(branch string) error
}

type GitInstaller interface {
	InstallGit() error
}

type ProjectState interface {
	AddAllConfirmed() (bool, error)
	ConfirmAddAll() error
}

type IgnoreManager interface {
	Ensure() (created bool, err error)
}

type UI interface {
	Banner()
	Println(message string)
	Success(message string)
	Warning(message string)
	Prompt(label string) (string, error)
	Confirm(label string) (bool, error)
	ConfirmExplicit(label string) (bool, error)
}

type Application struct {
	Git       Git
	Installer GitInstaller
	State     ProjectState
	Ignore    IgnoreManager
	UI        UI
}

func (a Application) Run(ctx context.Context) error {
	_ = ctx
	a.UI.Banner()

	version, err := a.Git.Version()
	if err != nil {
		a.UI.Warning("Git não encontrado.")
		accepted, promptErr := a.UI.Confirm("Deseja instalar o Git automaticamente? [S/N]")
		if promptErr != nil {
			return promptErr
		}
		if !accepted {
			a.UI.Println("Operação cancelada.")
			return nil
		}
		if err := a.Installer.InstallGit(); err != nil {
			return fmt.Errorf("não foi possível instalar o Git: %w", err)
		}
		version, err = a.Git.Version()
		if err != nil {
			return fmt.Errorf("o instalador terminou, mas o Git ainda não pôde ser executado: %w", err)
		}
	}
	a.UI.Success("Git encontrado: " + version)

	if err := a.ensureIdentity("user.name", "Nome do usuário do Git:"); err != nil {
		return err
	}
	if err := a.ensureIdentity("user.email", "Email do usuário do Git:"); err != nil {
		return err
	}

	isRepository, err := a.Git.IsRepository()
	if err != nil {
		return err
	}
	if !isRepository {
		a.UI.Warning("Repositório Git não encontrado. Inicializando...")
		if err := a.Git.Init(); err != nil {
			return fmt.Errorf("falha ao inicializar o repositório: %w", err)
		}
		a.UI.Success("Repositório Git inicializado.")
	} else {
		a.UI.Success("Repositório Git encontrado.")
	}

	origin, found, err := a.Git.Origin()
	if err != nil {
		return err
	}
	if found {
		a.UI.Success("Remote origin encontrado:")
		a.UI.Println(redactRemote(origin))
	} else {
		a.UI.Warning("Nenhum remote \"origin\" encontrado.")
		for {
			origin, err = a.promptRequired("Cole a URL do seu repositório GitHub:", "A URL não pode estar vazia.")
			if err != nil {
				return err
			}
			if validationErr := validateRemote(origin); validationErr != nil {
				a.UI.Warning(validationErr.Error())
				continue
			}
			break
		}
		if err := a.Git.AddOrigin(origin); err != nil {
			return fmt.Errorf("falha ao adicionar o remote origin: %w", err)
		}
		a.UI.Success("Remote origin configurado.")
	}

	created, err := a.Ignore.Ensure()
	if err != nil {
		return fmt.Errorf("falha ao verificar ou criar o .gitignore: %w", err)
	}
	if created {
		a.UI.Success(".gitignore criado.")
	}

	confirmed, err := a.State.AddAllConfirmed()
	if err != nil {
		return fmt.Errorf("falha ao ler o estado local do Zyr: %w", err)
	}
	if !confirmed {
		a.UI.Warning("O Zyr utiliza \"git add .\" para adicionar as alterações.\n\n" +
			"Isso adicionará os arquivos do diretório atual que não estiverem sendo ignorados pelo .gitignore.\n\n" +
			"É recomendado revisar o .gitignore antes de continuar.")
		accepted, err := a.UI.Confirm("Continuar? [S/N]")
		if err != nil {
			return err
		}
		if !accepted {
			a.UI.Println("Operação cancelada.")
			return nil
		}
		if err := a.State.ConfirmAddAll(); err != nil {
			return fmt.Errorf("falha ao salvar a confirmação local: %w", err)
		}
	}

	hasChanges, err := a.Git.HasChanges()
	if err != nil {
		return err
	}
	if !hasChanges {
		a.UI.Success("Nenhuma alteração encontrada.")
		a.UI.Println("Nada para commitar.")
		return nil
	}

	if err := a.Git.AddAll(); err != nil {
		return fmt.Errorf("falha ao adicionar arquivos: %w", err)
	}
	hasStagedChanges, err := a.Git.HasStagedChanges()
	if err != nil {
		return err
	}
	if !hasStagedChanges {
		a.UI.Success("Nenhuma alteração encontrada.")
		a.UI.Println("Nada para commitar.")
		return nil
	}
	a.UI.Success("Alterações adicionadas.")

	message, err := a.promptRequired("Mensagem do commit:", "A mensagem do commit não pode estar vazia.")
	if err != nil {
		return err
	}
	if err := a.Git.Commit(message); err != nil {
		return fmt.Errorf("falha ao criar o commit: %w", err)
	}
	a.UI.Success("Commit criado.")

	branch, err := a.Git.CurrentBranch()
	if err != nil {
		return err
	}
	hasUpstream, err := a.Git.HasOriginUpstream(branch)
	if err != nil {
		return err
	}
	if err := a.Git.Push(branch, !hasUpstream); err != nil {
		return fmt.Errorf("push falhou: %w", err)
	}
	a.UI.Success("Push realizado com sucesso.")
	return nil
}

func (a Application) ensureIdentity(key, label string) error {
	value, err := a.Git.GlobalConfig(key)
	if err != nil {
		return err
	}
	if strings.TrimSpace(value) != "" {
		return nil
	}
	value, err = a.promptRequired(label, "O valor não pode estar vazio.")
	if err != nil {
		return err
	}
	if err := a.Git.SetGlobalConfig(key, value); err != nil {
		return fmt.Errorf("falha ao configurar %s globalmente: %w", key, err)
	}
	return nil
}

func (a Application) promptRequired(label, emptyWarning string) (string, error) {
	for {
		value, err := a.UI.Prompt(label)
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value, nil
		}
		a.UI.Warning(emptyWarning)
	}
}

func validateRemote(value string) error {
	if strings.ContainsAny(value, "\r\n\t") || strings.Contains(value, " ") {
		return errors.New("A URL do remote contém espaços ou caracteres inválidos.")
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			return errors.New("Informe uma URL de repositório válida.")
		}
		if parsed.User != nil {
			return errors.New("Não inclua usuário, senha ou token na URL do remote.")
		}
	}
	return nil
}

func redactRemote(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.User == nil || parsed.Host == "" {
		return value
	}
	parsed.User = url.User("REDACTED")
	return parsed.String()
}
