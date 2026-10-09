package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseStatusAndRender(t *testing.T) {
	raw := "# branch.oid abcdef\x00# branch.head main\x00# branch.upstream origin/main\x00# branch.ab +2 -1\x00" +
		"1 .M N... 100644 100644 100644 abc def changed file.txt\x00" +
		"1 M. N... 100644 100644 100644 abc def staged.txt\x00" +
		"1 .D N... 100644 100644 000000 abc def removed.txt\x00" +
		"2 R. N... 100644 100644 100644 abc def R100 renamed.txt\x00old.txt\x00" +
		"u UU N... 100644 100644 100644 100644 abc def ghi conflict.txt\x00" +
		"? new file.txt\x00"
	report, err := parseStatus(raw)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	report.write(&output, "example")
	for _, want := range []string{
		"Projeto: example",
		"Branch: main → origin/main",
		"2 preparados para commit",
		"1 modificado",
		"1 novo",
		"1 excluído",
		"1 em conflito",
		"renamed.txt",
		"2 para enviar",
		"1 para receber",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output missing %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "old.txt") {
		t.Errorf("old rename path was treated as a separate change:\n%s", output.String())
	}
}

func TestParseStatusWithoutUpstream(t *testing.T) {
	report, err := parseStatus("# branch.oid (initial)\x00# branch.head main\x00? first.txt\x00")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	report.write(&output, "new-project")
	if !strings.Contains(output.String(), "Branch: main (sem upstream)") ||
		!strings.Contains(output.String(), "Commits: upstream não configurado") {
		t.Fatal(output.String())
	}
}

func TestParseStatusRejectsMalformedRecord(t *testing.T) {
	_, err := parseStatus("# branch.head main\x001 .M broken\x00")
	if err == nil {
		t.Fatal("expected malformed status to fail")
	}
}
