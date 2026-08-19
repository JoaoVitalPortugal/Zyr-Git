package terminal

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Terminal struct {
	reader *bufio.Reader
	out    io.Writer
}

func New(input io.Reader, output io.Writer) *Terminal {
	return &Terminal{reader: bufio.NewReader(input), out: output}
}

func (t *Terminal) Banner() {
	fmt.Fprintln(t.out, `     ______   __      __  _______
    |___  /   \ \    / / |  __  \
       / /     \ \  / /  | |__) |
      / /       \ \/ /   |  _  /
     / /__       \  /    | | \ \
    /_____|       \/     |_|  \_\`)
	fmt.Fprintln(t.out)
	fmt.Fprintln(t.out, "    Zyr Git Commit")
	fmt.Fprintln(t.out)
}

func (t *Terminal) Println(message string) { fmt.Fprintln(t.out, message) }
func (t *Terminal) Success(message string) { fmt.Fprintln(t.out, "✓ "+message) }
func (t *Terminal) Warning(message string) { fmt.Fprintln(t.out, "⚠ "+message) }
func (t *Terminal) Error(message string)   { fmt.Fprintln(t.out, "✕ "+message) }

func (t *Terminal) Prompt(label string) (string, error) {
	fmt.Fprintln(t.out, label)
	fmt.Fprint(t.out, "> ")
	line, err := t.reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", fmt.Errorf("não foi possível ler a resposta: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (t *Terminal) Confirm(label string) (bool, error) {
	for {
		answer, err := t.Prompt(label)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "s", "sim", "y", "yes":
			return true, nil
		case "n", "não", "nao", "no":
			return false, nil
		default:
			t.Warning("Responda S ou N.")
		}
	}
}

// ConfirmExplicit asks only once. It is used by destructive operations so
// every response other than an explicit affirmative answer cancels the action.
func (t *Terminal) ConfirmExplicit(label string) (bool, error) {
	answer, err := t.Prompt(label)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "s", "sim", "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
