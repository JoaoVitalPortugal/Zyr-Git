package app

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

type StatusGit interface {
	LookPath(name string) (string, error)
	CombinedOutput(name string, args ...string) (string, error)
}

type StatusApplication struct {
	Git    StatusGit
	Output io.Writer
}

type statusReport struct {
	branch    string
	upstream  string
	ahead     int
	behind    int
	hasCounts bool
	staged    []string
	modified  []string
	untracked []string
	deleted   []string
	conflicts []string
}

func (a StatusApplication) Run() error {
	if a.Git == nil || a.Output == nil {
		return fmt.Errorf("status não configurado")
	}
	gitPath, err := a.Git.LookPath("git")
	if err != nil {
		return fmt.Errorf("Git não encontrado. Instale o Git para consultar o status")
	}
	root, err := a.Git.CombinedOutput(gitPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("esta pasta não está em um repositório Git")
	}
	raw, err := a.Git.CombinedOutput(gitPath, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "-z")
	if err != nil {
		return fmt.Errorf("não foi possível consultar o status do Git: %w", err)
	}
	report, err := parseStatus(raw)
	if err != nil {
		return fmt.Errorf("não foi possível interpretar o status do Git: %w", err)
	}
	report.write(a.Output, filepath.Base(strings.TrimSpace(root)))
	return nil
}

func parseStatus(raw string) (statusReport, error) {
	var report statusReport
	records := strings.Split(raw, "\x00")
	if len(records) == 0 {
		return report, fmt.Errorf("resposta vazia")
	}
	for index := 0; index < len(records); index++ {
		record := records[index]
		if record == "" {
			continue
		}
		switch {
		case strings.HasPrefix(record, "# "):
			if err := report.header(record); err != nil {
				return report, err
			}
		case strings.HasPrefix(record, "1 "):
			fields := strings.SplitN(record, " ", 9)
			if len(fields) != 9 {
				return report, fmt.Errorf("registro de arquivo inválido")
			}
			report.change(fields[1], fields[8])
		case strings.HasPrefix(record, "2 "):
			fields := strings.SplitN(record, " ", 10)
			if len(fields) != 10 || index+1 >= len(records) {
				return report, fmt.Errorf("registro de renomeação inválido")
			}
			report.change(fields[1], fields[9])
			index++ // The next NUL-delimited field is the old path.
		case strings.HasPrefix(record, "u "):
			fields := strings.SplitN(record, " ", 11)
			if len(fields) != 11 {
				return report, fmt.Errorf("registro de conflito inválido")
			}
			report.conflicts = append(report.conflicts, fields[10])
		case strings.HasPrefix(record, "? "):
			report.untracked = append(report.untracked, record[2:])
		default:
			return report, fmt.Errorf("registro desconhecido: %.16q", record)
		}
	}
	if report.branch == "" {
		return report, fmt.Errorf("branch não informada")
	}
	return report, nil
}

func (r *statusReport) header(line string) error {
	switch {
	case strings.HasPrefix(line, "# branch.head "):
		r.branch = strings.TrimPrefix(line, "# branch.head ")
	case strings.HasPrefix(line, "# branch.upstream "):
		r.upstream = strings.TrimPrefix(line, "# branch.upstream ")
	case strings.HasPrefix(line, "# branch.ab "):
		fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
		if len(fields) != 2 || !strings.HasPrefix(fields[0], "+") || !strings.HasPrefix(fields[1], "-") {
			return fmt.Errorf("contagem de commits inválida")
		}
		ahead, errAhead := strconv.Atoi(fields[0][1:])
		behind, errBehind := strconv.Atoi(fields[1][1:])
		if errAhead != nil || errBehind != nil {
			return fmt.Errorf("contagem de commits inválida")
		}
		r.ahead, r.behind, r.hasCounts = ahead, behind, true
	}
	return nil
}

func (r *statusReport) change(xy, path string) {
	if len(xy) != 2 {
		return
	}
	if xy[0] != '.' {
		r.staged = append(r.staged, path)
	}
	if xy[1] == 'D' || xy[0] == 'D' {
		r.deleted = append(r.deleted, path)
	} else if xy[1] != '.' {
		r.modified = append(r.modified, path)
	}
}

func (r statusReport) write(out io.Writer, project string) {
	fmt.Fprintf(out, "Projeto: %s\n", project)
	switch r.branch {
	case "(detached)":
		fmt.Fprintln(out, "Branch: HEAD destacado")
	default:
		if r.upstream == "" {
			fmt.Fprintf(out, "Branch: %s (sem upstream)\n", r.branch)
		} else {
			fmt.Fprintf(out, "Branch: %s → %s\n", r.branch, r.upstream)
		}
	}
	fmt.Fprintln(out)
	if len(r.staged)+len(r.modified)+len(r.untracked)+len(r.deleted)+len(r.conflicts) == 0 {
		fmt.Fprintln(out, "Arquivos: nenhuma alteração")
	} else {
		fmt.Fprintln(out, "Arquivos")
		fmt.Fprintf(out, "  %d %s para commit\n", len(r.staged), word(len(r.staged), "preparado", "preparados"))
		fmt.Fprintf(out, "  %d %s\n", len(r.modified), word(len(r.modified), "modificado", "modificados"))
		fmt.Fprintf(out, "  %d %s\n", len(r.untracked), word(len(r.untracked), "novo", "novos"))
		fmt.Fprintf(out, "  %d %s\n", len(r.deleted), word(len(r.deleted), "excluído", "excluídos"))
		fmt.Fprintf(out, "  %d em conflito\n", len(r.conflicts))
	}
	writePaths(out, "Preparados", r.staged)
	writePaths(out, "Modificados", r.modified)
	writePaths(out, "Novos", r.untracked)
	writePaths(out, "Excluídos", r.deleted)
	writePaths(out, "Conflitos", r.conflicts)
	fmt.Fprintln(out)
	if r.upstream == "" {
		fmt.Fprintln(out, "Commits: upstream não configurado")
	} else if r.hasCounts {
		fmt.Fprintln(out, "Commits (desde a última sincronização local)")
		fmt.Fprintf(out, "  %d para enviar\n  %d para receber\n", r.ahead, r.behind)
	} else {
		fmt.Fprintln(out, "Commits: comparação indisponível")
	}
}

func word(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func writePaths(out io.Writer, title string, paths []string) {
	if len(paths) == 0 {
		return
	}
	fmt.Fprintf(out, "\n%s\n", title)
	for _, path := range paths {
		if strings.ContainsAny(path, "\r\n\t") {
			path = strconv.Quote(path)
		}
		fmt.Fprintf(out, "  %s\n", path)
	}
}
