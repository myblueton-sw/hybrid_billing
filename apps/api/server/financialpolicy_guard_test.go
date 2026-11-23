package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func sourceFinancialConfirmation(filename string, fn *ast.FuncDecl, sel *ast.SelectorExpr) bool {
	if filename != "financialpolicy.go" || fn.Name.Name != "confirmFinancialPolicyForStage" || sel.Sel.Name != "Confirm" || fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 {
		return false
	}
	pointer, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	typ, ok := pointer.X.(*ast.Ident)
	if !ok || typ.Name != "Server" {
		return false
	}
	var expression ast.Expr = sel.X
	for _, name := range []string{"approvals"} {
		owner, ok := expression.(*ast.SelectorExpr)
		if !ok || owner.Sel.Name != name {
			return false
		}
		expression = owner.X
	}
	base, ok := expression.(*ast.Ident)
	return ok && base.Name == fn.Recv.List[0].Names[0].Name
}

func TestFinancialConfirmationExceptionIsExact(t *testing.T) {
	for _, tc := range []struct {
		file, receiver, function, expression string
		want                                 bool
	}{
		{"financialpolicy.go", "h *Server", "confirmFinancialPolicyForStage", "h.approvals.Confirm", true},
		{"financialpolicy.go", "handler *Server", "confirmFinancialPolicyForStage", "handler.approvals.Confirm", true},
		{"financialpolicy.go", "h *Server", "confirmFinancialPolicy", "h.approvals.Confirm", false},
		{"other.go", "h *Server", "confirmFinancialPolicyForStage", "h.approvals.Confirm", false},
		{"financialpolicy.go", "h *Server", "Apply", "h.approvals.Confirm", false},
		{"financialpolicy.go", "h Server", "confirmFinancialPolicyForStage", "h.approvals.Confirm", false},
		{"financialpolicy.go", "h *Other", "confirmFinancialPolicyForStage", "h.approvals.Confirm", false},
		{"financialpolicy.go", "h *Server", "confirmFinancialPolicyForStage", "other.approvals.Confirm", false},
		{"financialpolicy.go", "h *Server", "confirmFinancialPolicyForStage", "h.webhooks.Confirm", false},
		{"financialpolicy.go", "h *Server", "confirmFinancialPolicyForStage", "h.approvals.Publish", false},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), tc.file, "package server; func ("+tc.receiver+") "+tc.function+"(){ _ = "+tc.expression+" }", 0)
		if err != nil {
			t.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		sel := fn.Body.List[0].(*ast.AssignStmt).Rhs[0].(*ast.SelectorExpr)
		if got := sourceFinancialConfirmation(tc.file, fn, sel); got != tc.want {
			t.Fatal(tc, got)
		}
	}
}
