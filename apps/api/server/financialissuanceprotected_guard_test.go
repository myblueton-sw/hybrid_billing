package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func sourceProtectedRulesConfirmation(filename string, fn *ast.FuncDecl, sel *ast.SelectorExpr) bool {
	if filename != "financialissuanceprotectedrules.go" || fn.Name.Name != "confirmProtectedRules" || sel.Sel.Name != "Confirm" || fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 {
		return false
	}
	pointer, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	typ, ok := pointer.X.(*ast.Ident)
	if !ok || typ.Name != "financialIssuance" {
		return false
	}
	var expression ast.Expr = sel.X
	for _, name := range []string{"fields"} {
		owner, ok := expression.(*ast.SelectorExpr)
		if !ok || owner.Sel.Name != name {
			return false
		}
		expression = owner.X
	}
	base, ok := expression.(*ast.Ident)
	return ok && base.Name == fn.Recv.List[0].Names[0].Name
}

func TestProtectedRulesConfirmationExceptionIsExact(t *testing.T) {
	for _, tc := range []struct {
		file, receiver, function, expression string
		want                                 bool
	}{
		{"financialissuanceprotectedrules.go", "h *financialIssuance", "confirmProtectedRules", "h.fields.Confirm", true},
		{"financialissuanceprotectedrules.go", "handler *financialIssuance", "confirmProtectedRules", "handler.fields.Confirm", true},
		{"financialissuanceprotectedrules.go", "h *financialIssuance", "proposeProtectedRules", "h.fields.Confirm", false},
		{"other.go", "h *financialIssuance", "confirmProtectedRules", "h.fields.Confirm", false},
		{"financialissuanceprotectedrules.go", "h *financialIssuance", "Apply", "h.fields.Confirm", false},
		{"financialissuanceprotectedrules.go", "h financialIssuance", "confirmProtectedRules", "h.fields.Confirm", false},
		{"financialissuanceprotectedrules.go", "h *Other", "confirmProtectedRules", "h.fields.Confirm", false},
		{"financialissuanceprotectedrules.go", "h *financialIssuance", "confirmProtectedRules", "other.fields.Confirm", false},
		{"financialissuanceprotectedrules.go", "h *financialIssuance", "confirmProtectedRules", "h.webhooks.Confirm", false},
		{"financialissuanceprotectedrules.go", "h *financialIssuance", "confirmProtectedRules", "h.fields.Publish", false},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), tc.file, "package server; func ("+tc.receiver+") "+tc.function+"(){ _ = "+tc.expression+" }", 0)
		if err != nil {
			t.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		sel := fn.Body.List[0].(*ast.AssignStmt).Rhs[0].(*ast.SelectorExpr)
		if got := sourceProtectedRulesConfirmation(tc.file, fn, sel); got != tc.want {
			t.Fatal(tc, got)
		}
	}
}
