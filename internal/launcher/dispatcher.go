package launcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const Protocol = "zyr-launcher/1"

type Component struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Executable    string `json:"executable"`
	Version       string `json:"version"`
	Description   string `json:"description"`
	Owner         string `json:"owner"`
}

type LegacyConfig struct {
	SchemaVersion int      `json:"schemaVersion"`
	Executables   []string `json:"executables"`
}

type Runner interface {
	Run(executable string, args []string) (exitCode int, err error)
}

type OSRunner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (r OSRunner) Run(executable string, args []string) (int, error) {
	cmd := exec.Command(executable, args...)
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode(), nil
	}
	return 1, err
}

type Dispatcher struct {
	Home         string
	LauncherPath string
	Version      string
	Out          io.Writer
	Err          io.Writer
	Runner       Runner
}

func HomeFromExecutable() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("ZYR_CLI_HOME")); configured != "" {
		return filepath.Abs(configured)
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar o launcher Zyr: %w", err)
	}
	directory := filepath.Dir(executable)
	if strings.EqualFold(filepath.Base(directory), "bin") {
		return filepath.Dir(directory), nil
	}
	return directory, nil
}

func (d Dispatcher) Run(args []string) int {
	if d.Out == nil {
		d.Out = io.Discard
	}
	if d.Err == nil {
		d.Err = io.Discard
	}
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "help")) {
		return d.printHelp()
	}
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		fmt.Fprintf(d.Out, "Zyr CLI %s\n", d.Version)
		return 0
	}

	components, err := LoadComponents(filepath.Join(d.Home, "components"))
	if err != nil {
		fmt.Fprintln(d.Err, "✕ "+err.Error())
		return 1
	}
	command := strings.ToLower(args[0])
	if component, found := components[command]; found {
		if samePath(component.Executable, d.LauncherPath) {
			fmt.Fprintf(d.Err, "✕ O componente %q aponta para o próprio launcher; encaminhamento cancelado.\n", command)
			return 1
		}
		return d.forward(component.Executable, args[1:])
	}
	return d.forwardLegacy(args)
}

func (d Dispatcher) printHelp() int {
	components, err := LoadComponents(filepath.Join(d.Home, "components"))
	if err != nil {
		fmt.Fprintln(d.Err, "✕ "+err.Error())
		return 1
	}
	fmt.Fprintln(d.Out, "Zyr CLI")
	fmt.Fprintln(d.Out)
	fmt.Fprintln(d.Out, "Uso: zyr <comando> [opções]")
	fmt.Fprintln(d.Out)
	fmt.Fprintln(d.Out, "Comandos disponíveis:")
	commands := make([]string, 0, len(components))
	for command := range components {
		commands = append(commands, command)
	}
	sort.Strings(commands)
	if len(commands) == 0 {
		fmt.Fprintln(d.Out, "  Nenhum componente registrado.")
	}
	for _, command := range commands {
		component := components[command]
		fmt.Fprintf(d.Out, "  %-10s %s\n", command, component.Description)
	}
	legacy, legacyErr := LoadLegacy(filepath.Join(d.Home, "legacy.json"))
	if legacyErr != nil {
		fmt.Fprintln(d.Err, "⚠ "+legacyErr.Error())
	} else if validLegacy := existingUniquePaths(legacy.Executables, d.LauncherPath); len(validLegacy) > 0 {
		fmt.Fprintln(d.Out)
		fmt.Fprintln(d.Out, "Executáveis Zyr legados preservados:")
		for _, executable := range validLegacy {
			fmt.Fprintln(d.Out, "  "+executable)
		}
		if len(validLegacy) > 1 {
			fmt.Fprintln(d.Out, "  Comandos legados ambíguos não serão executados automaticamente.")
		}
	}
	fmt.Fprintln(d.Out)
	fmt.Fprintln(d.Out, "Use 'zyr <comando> --help' para mais informações.")
	return 0
}

func (d Dispatcher) forwardLegacy(args []string) int {
	legacy, err := LoadLegacy(filepath.Join(d.Home, "legacy.json"))
	if err != nil {
		fmt.Fprintln(d.Err, "✕ "+err.Error())
		return 1
	}
	valid := existingUniquePaths(legacy.Executables, d.LauncherPath)
	switch len(valid) {
	case 0:
		fmt.Fprintf(d.Err, "✕ Comando Zyr desconhecido: %s\n", args[0])
		fmt.Fprintln(d.Err, "Execute 'zyr --help' para listar os componentes disponíveis.")
		return 2
	case 1:
		return d.forward(valid[0], args)
	default:
		fmt.Fprintf(d.Err, "✕ O comando %q é ambíguo porque existem vários executáveis Zyr legados:\n", args[0])
		for _, executable := range valid {
			fmt.Fprintln(d.Err, "  "+executable)
		}
		fmt.Fprintln(d.Err, "Nenhum deles foi executado. Remova a ambiguidade no registro do Zyr CLI.")
		return 1
	}
}

func (d Dispatcher) forward(executable string, args []string) int {
	if d.Runner == nil {
		fmt.Fprintln(d.Err, "✕ Executor de componentes não configurado.")
		return 1
	}
	code, err := d.Runner.Run(executable, args)
	if err != nil {
		fmt.Fprintf(d.Err, "✕ Não foi possível executar %s: %v\n", executable, err)
		return 1
	}
	return code
}

var commandName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func LoadComponents(directory string) (map[string]Component, error) {
	result := make(map[string]Component)
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler os componentes Zyr: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("não foi possível ler %s: %w", path, readErr)
		}
		var component Component
		if parseErr := json.Unmarshal(data, &component); parseErr != nil {
			return nil, fmt.Errorf("manifesto de componente inválido %s: %w", path, parseErr)
		}
		component.Command = strings.ToLower(strings.TrimSpace(component.Command))
		component.Executable = filepath.Clean(strings.TrimSpace(component.Executable))
		if component.SchemaVersion != 1 || !commandName.MatchString(component.Command) {
			return nil, fmt.Errorf("manifesto de componente inválido: %s", path)
		}
		if !filepath.IsAbs(component.Executable) {
			return nil, fmt.Errorf("o componente %q não possui caminho absoluto", component.Command)
		}
		if _, statErr := os.Stat(component.Executable); statErr != nil {
			return nil, fmt.Errorf("o componente %q está indisponível em %s", component.Command, component.Executable)
		}
		if previous, duplicate := result[component.Command]; duplicate {
			return nil, fmt.Errorf("o comando %q possui componentes duplicados: %s e %s", component.Command, previous.Executable, component.Executable)
		}
		result[component.Command] = component
	}
	return result, nil
}

func LoadLegacy(path string) (LegacyConfig, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return LegacyConfig{SchemaVersion: 1}, nil
	}
	if err != nil {
		return LegacyConfig{}, fmt.Errorf("não foi possível ler os executáveis Zyr legados: %w", err)
	}
	var config LegacyConfig
	if err := json.Unmarshal(data, &config); err != nil || config.SchemaVersion != 1 {
		return LegacyConfig{}, fmt.Errorf("registro de executáveis Zyr legados inválido: %s", path)
	}
	return config, nil
}

func existingUniquePaths(paths []string, launcherPath string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "." || path == "" || samePath(path, launcherPath) {
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

func samePath(first, second string) bool {
	first, _ = filepath.Abs(strings.TrimSpace(first))
	second, _ = filepath.Abs(strings.TrimSpace(second))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(first), filepath.Clean(second))
	}
	return filepath.Clean(first) == filepath.Clean(second)
}
