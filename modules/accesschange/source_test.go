package accesschange

import (
	"go/parser"
	"go/token"
	"os"
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
		for _, group := range file.Comments {
			for _, c := range group.List {
				if c.Text != "" {
					t.Fatalf("%s has a comment", name)
				}
			}
		}
		if strings.Contains(string(data), "flo"+"at") {
			t.Fatalf("%s mentions binary fractions", name)
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		for _, spec := range file.Imports {
			if strings.Contains(spec.Path.Value, ".") {
				t.Fatalf("%s imports %s", name, spec.Path.Value)
			}
		}
	}
}
