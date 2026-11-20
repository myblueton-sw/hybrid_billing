package server

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
	more, _ := filepath.Glob("../cmd/hb-api/*.go")
	for _, name := range append(files, more...) {
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
				if !strings.HasPrefix(c.Text, "//go:build ") {
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
			path := strings.Trim(spec.Path.Value, `"`)
			if path == "github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal" && (name == "approvalproducer.go" || name == "approvalnormalized.go" || name == "candidateseed.go") {
				continue
			}
			if strings.Contains(path, ".") && !strings.HasPrefix(path, "github.com/myblueton-sw/hybrid_billing/modules/") && !strings.HasPrefix(path, "github.com/myblueton-sw/hybrid_billing/apps/api/") {
				t.Fatalf("%s imports %s", name, path)
			}
		}
	}
}
