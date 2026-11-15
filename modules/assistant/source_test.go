package assistant

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSourceRules(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if lines := strings.Count(string(data), "\n"); lines > 1500 {
			t.Fatalf("%s has %d lines", name, lines)
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, data, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if len(file.Comments) > 0 {
			t.Fatalf("%s has comments", name)
		}
		if strings.Contains(string(data), "flo"+"at") {
			t.Fatalf("%s mentions binary fractions", name)
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		for _, spec := range file.Imports {
			allowed := map[string]bool{"\"github.com/myblueton-sw/hybrid_billing/modules/commands\"": true, "\"github.com/myblueton-sw/hybrid_billing/modules/gate\"": true,
				"\"github.com/myblueton-sw/hybrid_billing/modules/identity\"": true}
			if strings.Contains(spec.Path.Value, ".") && !allowed[spec.Path.Value] {
				t.Fatalf("%s imports %s", name, spec.Path.Value)
			}
		}
	}
}

func TestRefusal(t *testing.T) {
	err := error(Refusal{Reason: ReasonBudget})
	if err.Error() != ErrRejected.Error()+": budget_exhausted" || !errors.Is(err, ErrRejected) {
		t.Errorf("refusal %v", err)
	}
}

func TestLedgerKeepsNoValues(t *testing.T) {
	allowed := map[string]bool{"Sequence": true, "Kind": true, "Tool": true, "Target": true, "Card": true, "Operation": true, "Outcome": true, "Replay": true, "At": true}
	kind := reflect.TypeOf(Entry{})
	for index := 0; index < kind.NumField(); index++ {
		if !allowed[kind.Field(index).Name] {
			t.Errorf("the ledger entry has a field %s", kind.Field(index).Name)
		}
	}
}
