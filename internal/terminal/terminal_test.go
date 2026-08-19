package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmExplicitAcceptsOnlyAffirmativeAnswer(t *testing.T) {
	tests := []struct {
		name     string
		answer   string
		expected bool
	}{
		{name: "uppercase S", answer: "S\n", expected: true},
		{name: "uppercase SIM", answer: "SIM\n", expected: true},
		{name: "negative", answer: "n\n", expected: false},
		{name: "empty", answer: "\n", expected: false},
		{name: "invalid", answer: "talvez\nS\n", expected: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			confirmed, err := New(strings.NewReader(test.answer), &output).ConfirmExplicit("Continuar? [S/N]")
			if err != nil {
				t.Fatal(err)
			}
			if confirmed != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, confirmed)
			}
		})
	}
}
