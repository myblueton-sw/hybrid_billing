package identity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var stateTransitions = map[string]bool{"effective": true, "effectiveNow": true, "setRolePeriod": true, "ExpireRoles": true, "ApplyRecorded": true, "SuspendMemberships": true,
	"SuspendMembership": true}

func TestActiveMembershipIsComparedOnlyInThePredicateAndTheTransitions(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	found := map[string]bool{}
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
				b, ok := n.(*ast.BinaryExpr)
				if !ok || b.Op != token.EQL && b.Op != token.NEQ {
					return true
				}
				for _, side := range []ast.Expr{b.X, b.Y} {
					if id, ok := side.(*ast.Ident); ok && id.Name == "MembershipActive" {
						found[fn.Name.Name] = true
						if !stateTransitions[fn.Name.Name] {
							t.Errorf("%s compares MembershipActive in %s", name, fn.Name.Name)
						}
					}
				}
				return true
			})
		}
	}
	names := []string{}
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) != len(stateTransitions) {
		t.Errorf("listed transitions no longer compare the state: %v", names)
	}
	for _, checked := range []string{"CreateServiceUser", "onlySeat", "activeAdministrator", "transition", "inviterStands", "Member", "Access", "Recheck"} {
		if found[checked] {
			t.Errorf("%s compares the state directly", checked)
		}
	}
}
