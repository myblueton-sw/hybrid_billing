package server

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var guardedStore = map[string]bool{"drop": true, "remove": true, "prune": true, "endManifest": true}

type expiryCall struct{ file, function, method string }

var allowedExpiry = map[expiryCall]bool{{"signin.go", "signInOutcome", "prune"}: true}

func sessionEndViolations(t *testing.T, extra map[string]string) []string {
	t.Helper()
	fset := token.NewFileSet()
	files := []*ast.File{}
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if ok, err := build.Default.MatchFile(".", name); err != nil || !ok {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	for name, src := range extra {
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	info := &gotypes.Info{Uses: map[*ast.Ident]gotypes.Object{}}
	conf := gotypes.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	if _, err := conf.Check("github.com/myblueton-sw/hybrid_billing/apps/api/server", fset, files, info); err != nil {
		t.Fatal(err)
	}
	out := []string{}
	for _, f := range files {
		file := filepath.Base(fset.Position(f.Pos()).Filename)
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if sessionReceiver(fn) == "sessions" {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				obj, ok := info.Uses[sel.Sel].(*gotypes.Func)
				if !ok {
					return true
				}
				if guardedCall(obj) && file != "sessionending.go" && !allowedExpiry[expiryCall{file, fn.Name.Name, obj.Name()}] {
					out = append(out, file+":"+fn.Name.Name+":"+obj.Name())
				}
				return true
			})
		}
	}
	sort.Strings(out)
	return out
}

func sessionReceiver(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func guardedCall(obj *gotypes.Func) bool {
	sig, ok := obj.Type().(*gotypes.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	recv := sig.Recv().Type()
	if p, ok := recv.(*gotypes.Pointer); ok {
		recv = p.Elem()
	}
	named, _ := recv.(*gotypes.Named)
	switch {
	case obj.Name() == "EndSession" && obj.Pkg() != nil && strings.HasSuffix(obj.Pkg().Path(), "/modules/identity"):
		return true
	case named != nil && named.Obj().Name() == "sessions" && guardedStore[obj.Name()]:
		return true
	case named != nil && named.Obj().Name() == "Server" && obj.Name() == "endModule":
		return true
	}
	return false
}

func TestSessionEndsGoThroughOneHelperFile(t *testing.T) {
	if got := sessionEndViolations(t, nil); len(got) != 0 {
		t.Fatalf("direct session ends outside sessionending.go: %v", got)
	}
}

func TestSessionEndGuardFiresOnAPlantedViolation(t *testing.T) {
	planted := `package server

import "github.com/myblueton-sw/hybrid_billing/modules/identity"

func (s *Server) plantedEnd(v identity.Verifier) {
	_ = s.identity.Boundary().EndSession(v)
	s.sessions.drop(v)
	s.sessions.prune(s.clock())
}
`
	got := sessionEndViolations(t, map[string]string{"planted.go": planted})
	want := "planted.go:plantedEnd:EndSession planted.go:plantedEnd:drop planted.go:plantedEnd:prune"
	if strings.Join(got, " ") != want {
		t.Fatalf("guard found %v", got)
	}
}
