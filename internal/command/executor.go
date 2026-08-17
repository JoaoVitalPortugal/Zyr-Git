package command

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Executor is the small process boundary shared by Git and OS-specific setup.
type Executor interface {
	CombinedOutput(name string, args ...string) (string, error)
	Interactive(name string, args ...string) error
	LookPath(name string) (string, error)
}

type OSExecutor struct{}

type Error struct {
	ExitCode int
	Output   string
	Err      error
}

func (e *Error) Error() string {
	if strings.TrimSpace(e.Output) != "" {
		return strings.TrimSpace(e.Output)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

func (OSExecutor) CombinedOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = commandEnvironment()
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err == nil {
		return text, nil
	}
	return text, wrapError(err, text)
}

func (OSExecutor) Interactive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = commandEnvironment()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return wrapError(err, "")
	}
	return nil
}

func (OSExecutor) LookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err == nil {
		return path, nil
	}

	// Installers do not update the PATH of an already running process. These
	// common locations let the same invocation continue after installing Git.
	if runtime.GOOS == "windows" && strings.EqualFold(name, "git") {
		candidates := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Git", "cmd", "git.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "cmd", "git.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Git", "cmd", "git.exe"),
		}
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", err
}

func ExitCode(err error) (int, bool) {
	var commandErr *Error
	if errors.As(err, &commandErr) {
		return commandErr.ExitCode, true
	}
	return 0, false
}

func wrapError(err error, output string) error {
	code := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	return &Error{ExitCode: code, Output: output, Err: err}
}

func commandEnvironment() []string {
	env := os.Environ()
	if runtime.GOOS != "windows" {
		env = append(env, "LC_ALL=C", "LANG=C")
	}
	return env
}
