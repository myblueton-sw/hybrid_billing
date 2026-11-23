package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func sourceInputConfirmation(filename string, fn *ast.FuncDecl, sel *ast.SelectorExpr) bool {
	if filename != "inputreviewcommands.go" || fn.Name.Name != "confirmInput" || sel.Sel.Name != "Confirm" || fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 {
		return false
	}
	typ, ok := fn.Recv.List[0].Type.(*ast.Ident)
	if !ok || typ.Name != "inputReviewCommand" {
		return false
	}
	var expression ast.Expr = sel.X
	for _, name := range []string{"registry", "inputReviews", "s"} {
		owner, ok := expression.(*ast.SelectorExpr)
		if !ok || owner.Sel.Name != name {
			return false
		}
		expression = owner.X
	}
	base, ok := expression.(*ast.Ident)
	return ok && base.Name == fn.Recv.List[0].Names[0].Name
}

func TestInputConfirmationExceptionIsExact(t *testing.T) {
	for _, tc := range []struct {
		file, receiver, function, expression string
		want                                 bool
	}{
		{"inputreviewcommands.go", "h inputReviewCommand", "confirmInput", "h.s.inputReviews.registry.Confirm", true},
		{"inputreviewcommands.go", "handler inputReviewCommand", "confirmInput", "handler.s.inputReviews.registry.Confirm", true},
		{"other.go", "h inputReviewCommand", "confirmInput", "h.s.inputReviews.registry.Confirm", false},
		{"inputreviewcommands.go", "h inputReviewCommand", "Apply", "h.s.inputReviews.registry.Confirm", false},
		{"inputreviewcommands.go", "h *inputReviewCommand", "confirmInput", "h.s.inputReviews.registry.Confirm", false},
		{"inputreviewcommands.go", "h Server", "confirmInput", "h.s.inputReviews.registry.Confirm", false},
		{"inputreviewcommands.go", "h inputReviewCommand", "confirmInput", "other.s.inputReviews.registry.Confirm", false},
		{"inputreviewcommands.go", "h inputReviewCommand", "confirmInput", "h.s.webhooks.registry.Confirm", false},
		{"inputreviewcommands.go", "h inputReviewCommand", "confirmInput", "h.s.inputReviews.registry.Publish", false},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), tc.file, "package server; func ("+tc.receiver+") "+tc.function+"(){ _ = "+tc.expression+" }", 0)
		if err != nil {
			t.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		sel := fn.Body.List[0].(*ast.AssignStmt).Rhs[0].(*ast.SelectorExpr)
		if got := sourceInputConfirmation(tc.file, fn, sel); got != tc.want {
			t.Fatal(tc, got)
		}
	}
}
