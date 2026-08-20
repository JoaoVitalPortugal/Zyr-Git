package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/JoaoVitalPortugal/zyr-git-commit/internal/launcher"
)

const (
	productName       = "Zyr Git Commit"
	publisher         = "João Vital/Jovenzinho"
	uninstallKey      = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Zyr Git Commit`
	componentProtocol = "zyr-component/git/1"
)

var version = "dev"

func main() {
	options, err := parseOptions(os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if options.showVersion {
		fmt.Printf("%s Setup %s\n", productName, version)
		return
	}
	if options.finishUninstall != "" {
		finishUninstall(options.finishUninstall, options.parentPID)
		return
	}
	if options.uninstall {
		if err := uninstall(options); err != nil {
			fatal(err)
		}
		return
	}
	if err := install(options); err != nil {
		fatal(err)
	}
}

type setupOptions struct {
	silent          bool
	installDir      string
	sharedHome      string
	noPath          bool
	noRegistry      bool
	uninstall       bool
	finishUninstall string
	parentPID       int
	showVersion     bool
}

type setupMode string

const (
	modeInstall setupMode = "instalação"
	modeUpdate  setupMode = "atualização"
	modeRepair  setupMode = "reparo"
)

type setupState struct {
	Mode             setupMode
	InstalledVersion string
	TargetVersion    string
}

func parseOptions(args []string) (setupOptions, error) {
	installDir, err := defaultInstallDirectory()
	if err != nil {
		return setupOptions{}, err
	}
	sharedHome, err := defaultSharedHome()
	if err != nil {
		return setupOptions{}, err
	}
	var options setupOptions
	flags := flag.NewFlagSet("ZyrGitCommit-Setup", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.BoolVar(&options.silent, "silent", false, "não mostra perguntas")
	flags.StringVar(&options.installDir, "install-dir", installDir, "diretório do componente Git")
	flags.StringVar(&options.sharedHome, "shared-home", sharedHome, "diretório compartilhado do Zyr CLI")
	flags.BoolVar(&options.noPath, "no-path", false, "não altera o PATH")
	flags.BoolVar(&options.noRegistry, "no-registry", false, "não cria a entrada de desinstalação")
	flags.BoolVar(&options.uninstall, "uninstall", false, "desinstala o componente")
	flags.StringVar(&options.finishUninstall, "finish-uninstall", "", "uso interno")
	flags.IntVar(&options.parentPID, "parent-pid", 0, "uso interno")
	flags.BoolVar(&options.showVersion, "version", false, "mostra a versão")
	if err := flags.Parse(args); err != nil {
		return setupOptions{}, err
	}
	options.installDir, err = cleanAbsoluteDirectory(options.installDir)
	if err != nil {
		return setupOptions{}, err
	}
	options.sharedHome, err = cleanAbsoluteDirectory(options.sharedHome)
	if err != nil {
		return setupOptions{}, err
	}
	if samePath(options.installDir, options.sharedHome) {
		return setupOptions{}, errors.New("o componente e o launcher compartilhado precisam de diretórios diferentes")
	}
	return options, nil
}

func defaultInstallDirectory() (string, error) {
	base, err := localAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Programs", productName), nil
}

func defaultSharedHome() (string, error) {
	base, err := localAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Programs", "Zyr CLI"), nil
}

func localAppData() (string, error) {
	if base := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); base != "" {
		return filepath.Clean(base), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar a pasta do usuário: %w", err)
	}
	return filepath.Join(home, "AppData", "Local"), nil
}

func cleanAbsoluteDirectory(directory string) (string, error) {
	abs, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	volume := filepath.VolumeName(abs)
	root := volume + string(os.PathSeparator)
	if abs == "" || strings.EqualFold(abs, filepath.Clean(root)) {
		return "", fmt.Errorf("diretório inseguro: %q", directory)
	}
	return abs, nil
}

func install(options setupOptions) error {
	if len(launcherPayload) == 0 || len(gitComponentPayload) == 0 {
		return errors.New("o instalador não contém todos os payloads; gere-o com scripts\\build.ps1")
	}

	launcherDir := filepath.Join(options.sharedHome, "bin")
	launcherPath := filepath.Join(launcherDir, "zyr.exe")
	componentsDir := filepath.Join(options.sharedHome, "components")
	manifestPath := filepath.Join(componentsDir, "git.json")
	legacyPath := filepath.Join(options.sharedHome, "legacy.json")
	componentPath := filepath.Join(options.installDir, "zyr-git-commit.exe")
	oldMonolithPath := filepath.Join(options.installDir, "zyr.exe")
	oldMonolithOwned := isOldGitCommitMonolith(oldMonolithPath)

	existingZyr, err := findZyrOnPath()
	if err != nil {
		return err
	}
	ownedPaths := []string{launcherPath}
	if oldMonolithOwned {
		ownedPaths = append(ownedPaths, oldMonolithPath)
	}
	existingZyr = filterOwnedPaths(existingZyr, ownedPaths...)
	if conflicts, conflictErr := machinePathConflicts(existingZyr); conflictErr != nil {
		return conflictErr
	} else if len(conflicts) > 0 {
		return fmt.Errorf("não é possível garantir o comando 'zyr' porque outro executável está no PATH do sistema:\n  %s\nnenhum arquivo foi alterado; um launcher compartilhado em nível de sistema é necessário para resolver esse conflito", strings.Join(conflicts, "\n  "))
	}
	printExistingZyrWarning(existingZyr)

	if err := verifyOwnedExecutable(launcherPath, "--zyr-launcher-protocol", launcher.Protocol); err != nil {
		return err
	}
	if err := verifyOwnedExecutable(componentPath, "--zyr-component-protocol", componentProtocol); err != nil {
		return err
	}
	if err := verifyOwnedGitManifest(manifestPath, componentPath); err != nil {
		return err
	}
	if err := ensureNoGitManifestConflict(componentsDir, manifestPath); err != nil {
		return err
	}
	legacy, err := launcher.LoadLegacy(legacyPath)
	if err != nil {
		return err
	}
	excludedLegacy := []string{launcherPath}
	if oldMonolithOwned {
		excludedLegacy = append(excludedLegacy, oldMonolithPath)
	}
	legacy.Executables = mergeExistingPaths(legacy.Executables, existingZyr, excludedLegacy...)
	state, err := detectSetupState(manifestPath, oldMonolithOwned, version)
	if err != nil {
		return err
	}
	confirmed, err := showSetupNotice(options, state, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	if !options.silent {
		fmt.Printf("%s %s\n", productName, version)
		fmt.Printf("Componente Git: %s\n", options.installDir)
		fmt.Printf("Launcher compartilhado: %s\n", options.sharedHome)
	}
	for _, directory := range []string{options.installDir, launcherDir, componentsDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("não foi possível criar %s: %w", directory, err)
		}
	}
	if err := os.WriteFile(launcherPath, launcherPayload, 0o755); err != nil {
		return fmt.Errorf("não foi possível instalar o launcher Zyr: %w", err)
	}
	if err := os.WriteFile(componentPath, gitComponentPayload, 0o755); err != nil {
		return fmt.Errorf("não foi possível instalar o componente Git: %w", err)
	}

	manifest := launcher.Component{
		SchemaVersion: 1,
		Command:       "git",
		Executable:    componentPath,
		Version:       version,
		Description:   "Cria commits e gerencia repositórios e históricos Git",
		Owner:         productName,
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return fmt.Errorf("não foi possível registrar o componente Git: %w", err)
	}
	if len(legacy.Executables) > 0 {
		legacy.SchemaVersion = 1
		if err := writeJSON(legacyPath, legacy); err != nil {
			return fmt.Errorf("não foi possível registrar executáveis Zyr legados: %w", err)
		}
	}

	currentExecutable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("não foi possível localizar o instalador: %w", err)
	}
	uninstallerPath := filepath.Join(options.installDir, "uninstall.exe")
	if !samePath(currentExecutable, uninstallerPath) {
		installerBytes, readErr := os.ReadFile(currentExecutable)
		if readErr != nil {
			return fmt.Errorf("não foi possível preparar o desinstalador: %w", readErr)
		}
		if writeErr := os.WriteFile(uninstallerPath, installerBytes, 0o755); writeErr != nil {
			return fmt.Errorf("não foi possível gravar o desinstalador: %w", writeErr)
		}
	}

	// Migração segura da versão monolítica anterior deste mesmo produto.
	if oldMonolithOwned {
		_ = os.Remove(oldMonolithPath)
	}
	if !options.noPath {
		removeFromPath := ""
		if oldMonolithOwned {
			removeFromPath = options.installDir
		}
		if err := prioritizeUserPath(launcherDir, removeFromPath); err != nil {
			return fmt.Errorf("os arquivos foram instalados, mas o PATH não pôde ser atualizado: %w", err)
		}
	}
	if !options.noRegistry {
		size := int64(len(launcherPayload) + len(gitComponentPayload))
		if err := registerUninstaller(options, componentPath, uninstallerPath, size); err != nil {
			return fmt.Errorf("o componente foi instalado, mas não pôde ser registrado para desinstalação: %w", err)
		}
	}
	notifyEnvironmentChanged()
	if !options.silent {
		switch state.Mode {
		case modeUpdate:
			fmt.Println("✓ Atualização do Zyr Git Commit concluída com sucesso.")
		case modeRepair:
			fmt.Println("✓ Reparo do Zyr Git Commit concluído com sucesso.")
		default:
			fmt.Println("✓ Instalação do Zyr Git Commit concluída com sucesso.")
		}
		fmt.Println("Abra um novo terminal e execute: zyr --help")
	}
	return nil
}

func detectSetupState(manifestPath string, oldMonolithOwned bool, targetVersion string) (setupState, error) {
	state := setupState{Mode: modeInstall, TargetVersion: targetVersion}
	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		if oldMonolithOwned {
			state.Mode = modeUpdate
			state.InstalledVersion = "versão monolítica anterior"
		}
		return state, nil
	}
	if err != nil {
		return setupState{}, err
	}
	var manifest launcher.Component
	if err := json.Unmarshal(data, &manifest); err != nil {
		return setupState{}, fmt.Errorf("não foi possível identificar a versão instalada: %w", err)
	}
	state.InstalledVersion = strings.TrimSpace(manifest.Version)
	if state.InstalledVersion == "" {
		state.InstalledVersion = "desconhecida"
	}
	if state.InstalledVersion == targetVersion {
		state.Mode = modeRepair
	} else {
		state.Mode = modeUpdate
	}
	return state, nil
}

func showSetupNotice(options setupOptions, state setupState, input io.Reader, output io.Writer) (bool, error) {
	if options.silent {
		switch state.Mode {
		case modeInstall:
			fmt.Fprintf(output, "ℹ Instalação silenciosa do Zyr Git Commit %s.\n", state.TargetVersion)
		case modeUpdate:
			fmt.Fprintf(output, "ℹ Atualização silenciosa do Zyr Git Commit: %s → %s.\n", state.InstalledVersion, state.TargetVersion)
		case modeRepair:
			fmt.Fprintf(output, "ℹ Reparo silencioso do Zyr Git Commit %s.\n", state.TargetVersion)
		}
		return true, nil
	}

	fmt.Fprintln(output)
	switch state.Mode {
	case modeInstall:
		fmt.Fprintln(output, "=== INSTALAÇÃO DO ZYR GIT COMMIT ===")
		fmt.Fprintf(output, "O Zyr Git Commit %s será instalado nesta máquina.\n", state.TargetVersion)
	case modeUpdate:
		fmt.Fprintln(output, "=== ATUALIZAÇÃO DO ZYR GIT COMMIT ===")
		fmt.Fprintf(output, "Versão instalada: %s\n", state.InstalledVersion)
		fmt.Fprintf(output, "Nova versão:       %s\n", state.TargetVersion)
		fmt.Fprintln(output, "O launcher, o componente Git e o manifesto serão atualizados.")
	case modeRepair:
		fmt.Fprintln(output, "=== REPARO DO ZYR GIT COMMIT ===")
		fmt.Fprintf(output, "A versão %s já está instalada.\n", state.TargetVersion)
		fmt.Fprintln(output, "O instalador irá restaurar o launcher, o componente Git,")
		fmt.Fprintln(output, "o manifesto e a configuração do PATH deste produto.")
	}
	fmt.Fprintln(output)
	fmt.Fprintln(output, "⚠ Outros executáveis e projetos Zyr não serão substituídos, apagados")
	fmt.Fprintln(output, "  ou modificados por esta operação.")
	fmt.Fprintf(output, "Componente Git:       %s\n", options.installDir)
	fmt.Fprintf(output, "Launcher compartilhado: %s\n", options.sharedHome)

	reader := bufio.NewReader(input)
	for {
		question := "Continuar com a instalação? [S/N]"
		switch state.Mode {
		case modeUpdate:
			question = "Continuar com a atualização? [S/N]"
		case modeRepair:
			question = "Continuar com o reparo? [S/N]"
		}
		fmt.Fprintln(output, question)
		fmt.Fprint(output, "> ")
		answer, err := reader.ReadString('\n')
		if err != nil && answer == "" {
			return false, fmt.Errorf("não foi possível ler a confirmação: %w", err)
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "s", "sim", "y", "yes":
			return true, nil
		case "n", "não", "nao", "no":
			fmt.Fprintln(output, "Operação cancelada. Nenhum arquivo foi alterado.")
			return false, nil
		default:
			fmt.Fprintln(output, "⚠ Responda S ou N.")
		}
	}
}

func uninstall(options setupOptions) error {
	if !options.silent {
		fmt.Printf("Desinstalar %s? [S/N]\n> ", productName)
		answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && answer == "" {
			return err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "s" && answer != "sim" && answer != "y" && answer != "yes" {
			fmt.Println("Operação cancelada.")
			return nil
		}
	}

	launcherDir := filepath.Join(options.sharedHome, "bin")
	componentsDir := filepath.Join(options.sharedHome, "components")
	manifestPath := filepath.Join(componentsDir, "git.json")
	componentPath := filepath.Join(options.installDir, "zyr-git-commit.exe")
	if err := verifyOwnedExecutable(componentPath, "--zyr-component-protocol", componentProtocol); err != nil {
		return err
	}
	manifestCountBeforeRemoval, err := componentManifestCount(componentsDir)
	if err != nil {
		return err
	}
	launcherPath := filepath.Join(launcherDir, "zyr.exe")
	if manifestCountBeforeRemoval <= 1 {
		if err := verifyOwnedExecutable(launcherPath, "--zyr-launcher-protocol", launcher.Protocol); err != nil {
			return err
		}
	}
	if err := removeOwnedGitManifest(manifestPath, componentPath); err != nil {
		return err
	}
	_ = os.Remove(componentPath)

	otherComponents, err := componentManifestCount(componentsDir)
	if err != nil {
		return err
	}
	if otherComponents == 0 {
		_ = os.Remove(filepath.Join(options.sharedHome, "legacy.json"))
		_ = os.Remove(launcherPath)
		_ = os.Remove(componentsDir)
		_ = os.Remove(launcherDir)
		_ = os.Remove(options.sharedHome)
		if !options.noPath {
			if err := removeUserPathEntries(launcherDir); err != nil {
				return fmt.Errorf("não foi possível restaurar o PATH: %w", err)
			}
		}
	}
	if !options.noRegistry {
		_ = runReg("delete", uninstallKey, "/f")
	}
	notifyEnvironmentChanged()
	if err := launchUninstallFinalizer(options.installDir); err != nil {
		return fmt.Errorf("o componente foi removido, mas o desinstalador não pôde limpar seus arquivos: %w", err)
	}
	if !options.silent {
		fmt.Println("✓ Componente Zyr Git Commit desinstalado.")
	}
	return nil
}

func verifyOwnedExecutable(path, argument, expected string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("o destino esperado para um executável é um diretório: %s", path)
	}
	out, err := exec.Command(path, argument).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != expected {
		return fmt.Errorf("um executável não reconhecido já existe em %s; ele não será sobrescrito", path)
	}
	return nil
}

func isOldGitCommitMonolith(path string) bool {
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return false
	}
	out, err := exec.Command(path, "--version").CombinedOutput()
	return err == nil && strings.HasPrefix(strings.TrimSpace(string(out)), "Zyr Git Commit ")
}

func verifyOwnedGitManifest(path, componentPath string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var manifest launcher.Component
	if json.Unmarshal(data, &manifest) != nil || !strings.EqualFold(manifest.Command, "git") || manifest.Owner != productName || !samePath(manifest.Executable, componentPath) {
		return fmt.Errorf("um manifesto do comando 'git' não pertencente a este produto já existe em %s; ele não será sobrescrito", path)
	}
	return nil
}

func removeOwnedGitManifest(path, componentPath string) error {
	if err := verifyOwnedGitManifest(path, componentPath); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func findZyrOnPath() ([]string, error) {
	out, err := exec.Command("where.exe", "zyr").CombinedOutput()
	if err != nil && len(out) == 0 {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("não foi possível verificar where.exe zyr: %w", err)
	}
	return parseWhereOutput(string(out)), nil
}

func parseWhereOutput(output string) []string {
	result := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		path := strings.TrimSpace(line)
		if path == "" || !filepath.IsAbs(path) {
			continue
		}
		duplicate := false
		for _, existing := range result {
			if samePath(existing, path) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, filepath.Clean(path))
		}
	}
	return result
}

func filterOwnedPaths(paths []string, owned ...string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		isOwned := false
		for _, known := range owned {
			if samePath(path, known) {
				isOwned = true
				break
			}
		}
		if !isOwned {
			result = append(result, path)
		}
	}
	return result
}

func printExistingZyrWarning(paths []string) {
	if len(paths) == 0 {
		return
	}
	fmt.Println("⚠ Um ou mais executáveis \"zyr\" já foram encontrados no PATH.")
	fmt.Println()
	for _, path := range paths {
		fmt.Println("  " + path)
	}
	fmt.Println()
	fmt.Println("Eles não serão apagados nem sobrescritos. O launcher compartilhado")
	fmt.Println("registrará esses caminhos para encaminhamento legado seguro.")
}

func machinePathConflicts(candidates []string) ([]string, error) {
	machinePath, err := readRegistryPath(`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)
	if err != nil {
		return nil, err
	}
	entries := splitPath(machinePath)
	conflicts := make([]string, 0)
	for _, candidate := range candidates {
		directory := filepath.Dir(candidate)
		for _, entry := range entries {
			if samePath(directory, expandWindowsEnvironment(entry)) {
				conflicts = append(conflicts, candidate)
				break
			}
		}
	}
	return conflicts, nil
}

var windowsEnvironmentVariable = regexp.MustCompile(`%([^%]+)%`)

func expandWindowsEnvironment(value string) string {
	return windowsEnvironmentVariable.ReplaceAllStringFunc(value, func(match string) string {
		name := strings.Trim(match, "%")
		if expanded := os.Getenv(name); expanded != "" {
			return expanded
		}
		return match
	})
}

func mergeExistingPaths(existing, discovered []string, excluded ...string) []string {
	candidates := append(append([]string(nil), existing...), discovered...)
	result := make([]string, 0)
	for _, path := range candidates {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "." || !filepath.IsAbs(path) {
			continue
		}
		skip := false
		for _, excludedPath := range excluded {
			if samePath(path, excludedPath) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			continue
		}
		duplicate := false
		for _, existing := range result {
			if samePath(existing, path) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, path)
		}
	}
	sort.Strings(result)
	return result
}

func ensureNoGitManifestConflict(directory, ownedManifest string) error {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		if entry.IsDir() || samePath(path, ownedManifest) || !strings.EqualFold(filepath.Ext(path), ".json") {
			continue
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var manifest launcher.Component
		if json.Unmarshal(data, &manifest) == nil && strings.EqualFold(strings.TrimSpace(manifest.Command), "git") {
			return fmt.Errorf("outro componente já registrou o comando 'git': %s", path)
		}
	}
	return nil
}

func writeJSON(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func componentManifestCount(directory string) (int, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			count++
		}
	}
	return count, nil
}

func launchUninstallFinalizer(installDir string) error {
	current, err := os.Executable()
	if err != nil {
		return err
	}
	temporary := filepath.Join(os.TempDir(), "zyr-git-commit-uninstall-"+strconv.Itoa(os.Getpid())+".exe")
	bytes, err := os.ReadFile(current)
	if err != nil {
		return err
	}
	if err := os.WriteFile(temporary, bytes, 0o755); err != nil {
		return err
	}
	cmd := exec.Command(temporary, "--finish-uninstall", installDir, "--parent-pid", strconv.Itoa(os.Getpid()), "--silent")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func finishUninstall(installDir string, parentPID int) {
	installDir, err := cleanAbsoluteDirectory(installDir)
	if err != nil {
		return
	}
	if parentPID > 0 {
		waitForProcess(uint32(parentPID))
	} else {
		time.Sleep(750 * time.Millisecond)
	}
	_ = os.Remove(filepath.Join(installDir, "zyr-git-commit.exe"))
	_ = os.Remove(filepath.Join(installDir, "uninstall.exe"))
	_ = os.Remove(installDir)
	if current, executableErr := os.Executable(); executableErr == nil {
		_ = deleteWithPOSIXSemantics(current)
		if !launchDelayedSelfDelete(current) {
			scheduleDeleteOnReboot(current)
		}
	}
}

func registerUninstaller(options setupOptions, componentPath, uninstallerPath string, payloadSize int64) error {
	baseArgs := fmt.Sprintf("--uninstall --install-dir \"%s\" --shared-home \"%s\"", options.installDir, options.sharedHome)
	values := []struct {
		name, kind, value string
	}{
		{"DisplayName", "REG_SZ", productName},
		{"DisplayVersion", "REG_SZ", version},
		{"Publisher", "REG_SZ", publisher},
		{"InstallLocation", "REG_SZ", options.installDir},
		{"DisplayIcon", "REG_SZ", componentPath},
		{"UninstallString", "REG_SZ", fmt.Sprintf("\"%s\" %s", uninstallerPath, baseArgs)},
		{"QuietUninstallString", "REG_SZ", fmt.Sprintf("\"%s\" %s --silent", uninstallerPath, baseArgs)},
		{"NoModify", "REG_DWORD", "1"},
		{"NoRepair", "REG_DWORD", "1"},
		{"EstimatedSize", "REG_DWORD", strconv.FormatInt((payloadSize*2)/1024, 10)},
	}
	for _, value := range values {
		if err := runReg("add", uninstallKey, "/v", value.name, "/t", value.kind, "/d", value.value, "/f"); err != nil {
			return err
		}
	}
	return nil
}

var pathLine = regexp.MustCompile(`(?im)^\s*Path\s+(REG_[A-Z_]+)\s+(.*)$`)

func readRegistryPath(key string) (string, error) {
	command := exec.Command("reg.exe", "query", key, "/v", "Path")
	out, err := command.CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("reg.exe query: %s", strings.TrimSpace(string(out)))
	}
	match := pathLine.FindStringSubmatch(string(out))
	if len(match) != 3 {
		return "", nil
	}
	return strings.TrimSpace(match[2]), nil
}

func prioritizeUserPath(launcherDir string, oldComponentDir string) error {
	current, err := readRegistryPath(`HKCU\Environment`)
	if err != nil {
		return err
	}
	updated := prioritizePathEntry(current, launcherDir, oldComponentDir)
	if updated == current {
		return nil
	}
	return runReg("add", `HKCU\Environment`, "/v", "Path", "/t", "REG_EXPAND_SZ", "/d", updated, "/f")
}

func removeUserPathEntries(entries ...string) error {
	current, err := readRegistryPath(`HKCU\Environment`)
	if err != nil {
		return err
	}
	updated := current
	for _, entry := range entries {
		updated = removePathEntry(updated, entry)
	}
	if updated == current {
		return nil
	}
	return runReg("add", `HKCU\Environment`, "/v", "Path", "/t", "REG_EXPAND_SZ", "/d", updated, "/f")
}

func prioritizePathEntry(current, entry string, remove ...string) string {
	parts := splitPath(current)
	kept := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		if samePath(part, entry) {
			continue
		}
		shouldRemove := false
		for _, old := range remove {
			if strings.TrimSpace(old) != "" && samePath(part, old) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			kept = append(kept, part)
		}
	}
	return strings.Join(append([]string{filepath.Clean(entry)}, kept...), ";")
}

func removePathEntry(current, entry string) string {
	parts := splitPath(current)
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if !samePath(part, entry) {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ";")
}

func splitPath(value string) []string {
	raw := strings.Split(value, ";")
	result := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(strings.Trim(part, `"`))
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func samePath(first, second string) bool {
	first = strings.TrimSpace(strings.Trim(first, `"`))
	second = strings.TrimSpace(strings.Trim(second, `"`))
	first = expandWindowsEnvironment(first)
	second = expandWindowsEnvironment(second)
	return strings.EqualFold(filepath.Clean(first), filepath.Clean(second))
}

func runReg(args ...string) error {
	out, err := exec.Command("reg.exe", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("reg.exe: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func notifyEnvironmentChanged() {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	text, _ := syscall.UTF16PtrFromString("Environment")
	const (
		hwndBroadcast   = 0xffff
		wmSettingChange = 0x001A
		smtoAbortIfHung = 0x0002
	)
	proc.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(text)), smtoAbortIfHung, 5000, 0)
}

func waitForProcess(pid uint32) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	open := kernel32.NewProc("OpenProcess")
	wait := kernel32.NewProc("WaitForSingleObject")
	closeHandle := kernel32.NewProc("CloseHandle")
	const synchronize = 0x00100000
	handle, _, _ := open.Call(synchronize, 0, uintptr(pid))
	if handle == 0 {
		time.Sleep(750 * time.Millisecond)
		return
	}
	wait.Call(handle, 30000)
	closeHandle.Call(handle)
}

func deleteWithPOSIXSemantics(path string) bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createFile := kernel32.NewProc("CreateFileW")
	setFileInformation := kernel32.NewProc("SetFileInformationByHandle")
	closeHandle := kernel32.NewProc("CloseHandle")
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	const (
		deleteAccess        = 0x00010000
		fileShareRead       = 0x00000001
		fileShareWrite      = 0x00000002
		fileShareDelete     = 0x00000004
		openExisting        = 3
		fileAttributeNormal = 0x00000080
		fileDispositionEx   = 21
		flagDelete          = 0x00000001
		flagPOSIXSemantics  = 0x00000002
	)
	handle, _, _ := createFile.Call(
		uintptr(unsafe.Pointer(pathPointer)),
		deleteAccess,
		fileShareRead|fileShareWrite|fileShareDelete,
		0,
		openExisting,
		fileAttributeNormal,
		0,
	)
	if handle == ^uintptr(0) || handle == 0 {
		return false
	}
	defer closeHandle.Call(handle)
	info := struct{ Flags uint32 }{Flags: flagDelete | flagPOSIXSemantics}
	result, _, _ := setFileInformation.Call(handle, fileDispositionEx, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	return result != 0
}

func launchDelayedSelfDelete(path string) bool {
	temporaryDirectory, err := filepath.Abs(os.TempDir())
	if err != nil {
		return false
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(temporaryDirectory, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return false
	}
	if !strings.HasPrefix(strings.ToLower(filepath.Base(target)), "zyr-git-commit-uninstall-") {
		return false
	}
	const script = `$target=$env:ZYR_DELETE_TARGET;for($i=0;$i -lt 20 -and [IO.File]::Exists($target);$i++){Start-Sleep -Milliseconds 250;try{[IO.File]::Delete($target)}catch{}}`
	cmd := exec.Command(
		"powershell.exe",
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle", "Hidden",
		"-EncodedCommand", encodePowerShell(script),
	)
	cmd.Env = append(os.Environ(), "ZYR_DELETE_TARGET="+target)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd.Start() == nil
}

func encodePowerShell(script string) string {
	encoded := utf16.Encode([]rune(script))
	bytes := make([]byte, len(encoded)*2)
	for index, value := range encoded {
		binary.LittleEndian.PutUint16(bytes[index*2:], value)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func scheduleDeleteOnReboot(path string) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	moveFileEx := kernel32.NewProc("MoveFileExW")
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	const moveFileDelayUntilReboot = 0x00000004
	moveFileEx.Call(uintptr(unsafe.Pointer(pathPointer)), 0, moveFileDelayUntilReboot)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "✕ "+err.Error())
	os.Exit(1)
}
