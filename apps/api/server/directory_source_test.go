package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var directoryFields = map[string]bool{"people": true, "byLogin": true, "order": true, "departed": true, "version": true, "invites": true, "links": true, "holders": true}

var passwordStateCalls = []string{"admin.go departUser Remove", "admin.go recordLink Issue", "self.go changePassword Put", "signin.go activate Put", "signin.go setPassword Commit"}

func directoryField(target ast.Expr) string {
	for {
		switch node := target.(type) {
		case *ast.IndexExpr:
			target = node.X
			continue
		case *ast.SelectorExpr:
			if !directoryFields[node.Sel.Name] {
				return ""
			}
			if inner, ok := node.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "dir" {
				return node.Sel.Name
			}
			if id, ok := node.X.(*ast.Ident); ok && id.Name == "d" {
				return node.Sel.Name
			}
		}
		return ""
	}
}

func TestDirectoryStateIsWrittenOnlyThroughTheDirectory(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	outside, calls := []string{}, []string{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				targets := []ast.Expr{}
				switch node := n.(type) {
				case *ast.AssignStmt:
					targets = node.Lhs
				case *ast.IncDecStmt:
					targets = []ast.Expr{node.X}
				case *ast.CallExpr:
					if id, ok := node.Fun.(*ast.Ident); ok && id.Name == "delete" {
						targets = node.Args[:1]
					}
					if sel, ok := node.Fun.(*ast.SelectorExpr); ok && slices.Contains([]string{"Put", "Remove", "Commit", "Issue"}, sel.Sel.Name) {
						if inner, ok := sel.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "passwords" {
							calls = append(calls, name+" "+fn.Name.Name+" "+sel.Sel.Name)
						}
					}
				}
				for _, target := range targets {
					if field := directoryField(target); field != "" && name != "directory.go" {
						outside = append(outside, name+" "+fn.Name.Name+" "+field)
					}
				}
				return true
			})
		}
	}
	slices.Sort(outside)
	slices.Sort(calls)
	if len(outside) != 0 {
		t.Errorf("directory fields written outside directory.go: %v", outside)
	}
	if !slices.Equal(calls, passwordStateCalls) {
		t.Errorf("password state changes: %v", calls)
	}
}
