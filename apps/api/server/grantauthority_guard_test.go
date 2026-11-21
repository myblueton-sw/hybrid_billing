package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func sourceInputCapabilityConstant(filename string, fn *ast.FuncDecl, sel *ast.SelectorExpr) bool {
	if filename != "grantauthoritydomain.go" || fn.Name.Name != "terms" || sel.Sel.Name != "Confirm" || fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	receiver, ok := fn.Recv.List[0].Type.(*ast.Ident)
	if !ok || receiver.Name != "grantAuthorityDomain" {
		return false
	}
	owner, ok := sel.X.(*ast.Ident)
	if !ok || owner.Name != "inputauthority" {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		clause, ok := n.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, expr := range clause.List {
			if expr == sel {
				found = true
			}
		}
		return true
	})
	return found
}
func TestInputCapabilityConstantExceptionIsExact(t *testing.T) {
	for _, tc := range []struct {
		name, file, receiver, function, body string
		want                                 bool
	}{
		{"constant", "grantauthoritydomain.go", "d grantAuthorityDomain", "terms", "switch x {case inputauthority.Confirm:}", true},
		{"file", "other.go", "d grantAuthorityDomain", "terms", "switch x {case inputauthority.Confirm:}", false},
		{"method", "grantauthoritydomain.go", "d grantAuthorityDomain", "Apply", "switch x {case inputauthority.Confirm:}", false},
		{"type", "grantauthoritydomain.go", "d otherDomain", "terms", "switch x {case inputauthority.Confirm:}", false},
		{"pointer", "grantauthoritydomain.go", "d *grantAuthorityDomain", "terms", "switch x {case inputauthority.Confirm:}", false},
		{"owner", "grantauthoritydomain.go", "d grantAuthorityDomain", "terms", "switch x {case registry.Confirm:}", false},
		{"member", "grantauthoritydomain.go", "d grantAuthorityDomain", "terms", "switch x {case inputauthority.Publish:}", false},
		{"assignment", "grantauthoritydomain.go", "d grantAuthorityDomain", "terms", "f := inputauthority.Confirm", false},
		{"call", "grantauthoritydomain.go", "d grantAuthorityDomain", "terms", "inputauthority.Confirm()", false},
		{"case body", "grantauthoritydomain.go", "d grantAuthorityDomain", "terms", "switch x {case 1: _ = inputauthority.Confirm}", false},
		{"nested case", "grantauthoritydomain.go", "d grantAuthorityDomain", "terms", "switch x {case wrap(inputauthority.Confirm):}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), tc.file, "package server; func ("+tc.receiver+") "+tc.function+"(){"+tc.body+"}", 0)
			if err != nil {
				t.Fatal(err)
			}
			fn := file.Decls[0].(*ast.FuncDecl)
			var selector *ast.SelectorExpr
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if s, ok := n.(*ast.SelectorExpr); ok {
					selector = s
				}
				return true
			})
			if selector == nil || sourceInputCapabilityConstant(tc.file, fn, selector) != tc.want {
				t.Fatal("constant exception mismatch")
			}
		})
	}
}

func sourceDocumentCapabilityConstant(filename string, fn *ast.FuncDecl, sel *ast.SelectorExpr) bool {
	if filename != "grantauthoritydomain.go" || fn.Name.Name != "documentAuthorityTerms" || fn.Recv != nil || sel.Sel.Name != "Confirm" {
		return false
	}
	owner, ok := sel.X.(*ast.Ident)
	if !ok || owner.Name != "documentauthority" {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if clause, ok := n.(*ast.CaseClause); ok {
			for _, expr := range clause.List {
				if expr == sel {
					found = true
				}
			}
		}
		return true
	})
	return found
}
func TestDocumentCapabilityConstantExceptionIsExact(t *testing.T) {
	for _, tc := range []struct {
		file, name, body string
		want             bool
	}{
		{"grantauthoritydomain.go", "documentAuthorityTerms", "switch x {case documentauthority.Confirm:}", true},
		{"other.go", "documentAuthorityTerms", "switch x {case documentauthority.Confirm:}", false},
		{"grantauthoritydomain.go", "other", "switch x {case documentauthority.Confirm:}", false},
		{"grantauthoritydomain.go", "documentAuthorityTerms", "switch x {case registry.Confirm:}", false},
		{"grantauthoritydomain.go", "documentAuthorityTerms", "switch x {case documentauthority.Publish:}", false},
		{"grantauthoritydomain.go", "documentAuthorityTerms", "documentauthority.Confirm()", false},
		{"grantauthoritydomain.go", "documentAuthorityTerms", "switch x {case documentauthority.Confirm():}", false},
		{"grantauthoritydomain.go", "documentAuthorityTerms", "switch x {case 1: _ = documentauthority.Confirm}", false},
		{"grantauthoritydomain.go", "documentAuthorityTerms", "f := documentauthority.Confirm", false},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), tc.file, "package server; func "+tc.name+"(){"+tc.body+"}", 0)
		if err != nil {
			t.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		var selector *ast.SelectorExpr
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok {
				selector = s
			}
			return true
		})
		if selector == nil || sourceDocumentCapabilityConstant(tc.file, fn, selector) != tc.want {
			t.Fatal("constant exception mismatch", tc)
		}
	}
}

func sourceResultCapabilityConstant(filename string, fn *ast.FuncDecl, sel *ast.SelectorExpr) bool {
	if filename != "grantauthoritydomain.go" || fn.Name.Name != "resultAuthorityTerms" || fn.Recv != nil || sel.Sel.Name != "Confirm" {
		return false
	}
	owner, ok := sel.X.(*ast.Ident)
	if !ok || owner.Name != "resultauthority" {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if clause, ok := n.(*ast.CaseClause); ok {
			for _, expr := range clause.List {
				if expr == sel {
					found = true
				}
			}
		}
		return true
	})
	return found
}
func TestResultCapabilityConstantExceptionIsExact(t *testing.T) {
	for _, tc := range []struct {
		file, name, body string
		want             bool
	}{
		{"grantauthoritydomain.go", "resultAuthorityTerms", "switch x {case resultauthority.Confirm:}", true},
		{"other.go", "resultAuthorityTerms", "switch x {case resultauthority.Confirm:}", false},
		{"grantauthoritydomain.go", "other", "switch x {case resultauthority.Confirm:}", false},
		{"grantauthoritydomain.go", "resultAuthorityTerms", "switch x {case registry.Confirm:}", false},
		{"grantauthoritydomain.go", "resultAuthorityTerms", "switch x {case resultauthority.Publish:}", false},
		{"grantauthoritydomain.go", "resultAuthorityTerms", "resultauthority.Confirm()", false},
		{"grantauthoritydomain.go", "resultAuthorityTerms", "switch x {case resultauthority.Confirm():}", false},
		{"grantauthoritydomain.go", "resultAuthorityTerms", "switch x {case 1: _ = resultauthority.Confirm}", false},
		{"grantauthoritydomain.go", "resultAuthorityTerms", "f := resultauthority.Confirm", false},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), tc.file, "package server; func "+tc.name+"(){"+tc.body+"}", 0)
		if err != nil {
			t.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		var selector *ast.SelectorExpr
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok {
				selector = s
			}
			return true
		})
		if selector == nil || sourceResultCapabilityConstant(tc.file, fn, selector) != tc.want {
			t.Fatal("constant exception mismatch", tc)
		}
	}
}
