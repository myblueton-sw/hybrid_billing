package bulkrevoke

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceRules(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, name := range files {
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
			t.Fatalf("%s has a comment", name)
		}
		if strings.Contains(string(data), "flo"+"at") {
			t.Fatalf("%s mentions binary fractions", name)
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		for _, spec := range file.Imports {
			if strings.Contains(strings.Trim(spec.Path.Value, `"`), ".") {
				t.Fatalf("%s imports %s", name, spec.Path.Value)
			}
		}
	}
}
