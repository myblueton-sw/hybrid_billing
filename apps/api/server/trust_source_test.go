package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type callSite struct {
	file     string
	function string
	receiver string
	method   string
}

func exprText(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return exprText(x.X) + "." + x.Sel.Name
	}
	return ""
}

func sourceSites(t *testing.T, root string, visit func(file, function string, n ast.Node)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				visit(path, fn.Name.Name, n)
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func calls(t *testing.T, root string, method string) []callSite {
	out := []callSite{}
	sourceSites(t, root, func(file, function string, n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == method {
			out = append(out, callSite{file: filepath.Base(file), function: function, receiver: exprText(sel.X), method: method})
		}
	})
	sort.Slice(out, func(a, b int) bool { return out[a].file+out[a].function < out[b].file+out[b].function })
	return out
}

func TestExplainInstallationHasOnlyGovernanceCallers(t *testing.T) {
	sites := append(calls(t, "../../../modules", "ExplainInstallation"), calls(t, ".", "ExplainInstallation")...)
	files := map[string]bool{}
	for _, c := range sites {
		files[c.file] = true
		switch {
		case c.file == "gate.go" && c.function == "ExplainInstallation" && c.receiver == "a.permissions":
		case c.file == "trustevidence.go" && c.receiver == "s.admin":
		case inputAuthorityExplanation(c):
		case issuanceProfileExplanation(c):
		default:
			t.Errorf("ExplainInstallation called from %+v", c)
		}
	}
	if len(files) != 4 || !files["financialissuancetermaccess.go"] || !files["gate.go"] || !files["trustevidence.go"] || !files["grantauthorityfacts.go"] {
		t.Errorf("callers %v", files)
	}
	for _, m03 := range []string{"access.go", "accessrequests.go", "ledger.go", "integrations.go", "revocations.go"} {
		if files[m03] {
			t.Errorf("%s reaches ExplainInstallation", m03)
		}
	}
}

func TestPolicyIsPublishedOnlyThroughTheTrustWorkflow(t *testing.T) {
	for _, c := range calls(t, ".", "Publish") {
		if c.receiver == "s.policy" && c.function != "publishTrust" {
			t.Errorf("Publish called from %+v", c)
		}
	}
	for _, c := range calls(t, ".", "publishTrust") {
		if c.function != "activateTrust" && c.function != "replayTrust" {
			t.Errorf("publishTrust called from %+v", c)
		}
	}
	for _, method := range []string{"Propose", "Approve", "Withdraw"} {
		for _, c := range calls(t, ".", method) {
			if c.receiver == "s.policy" {
				t.Errorf("authpolicy %s called from %+v", method, c)
			}
		}
	}
	for _, c := range calls(t, ".", "Activate") {
		if c.receiver == "s.trust" && c.function != "activateTrust" {
			t.Errorf("trustchange Activate called from %+v", c)
		}
	}
}

func TestOnlySessionsPutWritesTheTrustEpoch(t *testing.T) {
	writers := []string{}
	sourceSites(t, ".", func(file, function string, n ast.Node) {
		switch x := n.(type) {
		case *ast.KeyValueExpr:
			if k, ok := x.Key.(*ast.Ident); ok && k.Name == "trustEpoch" {
				writers = append(writers, filepath.Base(file)+" "+function)
			}
		case *ast.AssignStmt:
			for _, lhs := range x.Lhs {
				if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "trustEpoch" {
					writers = append(writers, filepath.Base(file)+" "+function)
				}
			}
		}
	})
	if strings.Join(writers, ",") != "session.go put" {
		t.Errorf("trust epoch writers %v", writers)
	}
}

func TestM03UsesTheInstallationKindPredicate(t *testing.T) {
	comparisons := []string{}
	sourceSites(t, ".", func(file, function string, n ast.Node) {
		b, ok := n.(*ast.BinaryExpr)
		if !ok || (b.Op != token.EQL && b.Op != token.NEQ) {
			return
		}
		for _, side := range []ast.Expr{b.X, b.Y} {
			if strings.HasSuffix(exprText(side), "AdministerInstallation") {
				comparisons = append(comparisons, filepath.Base(file)+" "+function+" "+b.Op.String())
			}
		}
	})
	if strings.Join(comparisons, ",") != "accessrequests.go holdsAdmin ==" {
		t.Errorf("comparisons with administer_installation %v", comparisons)
	}
}

func inputAuthorityExplanation(c callSite) bool {
	return c.file == "grantauthorityfacts.go" && c.function == "governor" && c.receiver == "s.admin" && c.method == "ExplainInstallation"
}
func TestInputAuthorityExplanationExceptionIsExact(t *testing.T) {
	good := callSite{file: "grantauthorityfacts.go", function: "governor", receiver: "s.admin", method: "ExplainInstallation"}
	if !inputAuthorityExplanation(good) {
		t.Fatal("exact site refused")
	}
	for _, field := range []string{"file", "function", "receiver", "method"} {
		bad := good
		switch field {
		case "file":
			bad.file = "accessrequests.go"
		case "function":
			bad.function = "Apply"
		case "receiver":
			bad.receiver = "other.admin"
		case "method":
			bad.method = "Explain"
		}
		if inputAuthorityExplanation(bad) {
			t.Fatalf("broad %s exception", field)
		}
	}
}

func issuanceProfileExplanation(c callSite) bool {
	return c.file == "financialissuancetermaccess.go" && c.function == "issuanceProfileGovernorStanding" && c.receiver == "s.admin" && c.method == "ExplainInstallation"
}
func TestIssuanceProfileExplanationExceptionIsExact(t *testing.T) {
	good := callSite{file: "financialissuancetermaccess.go", function: "issuanceProfileGovernorStanding", receiver: "s.admin", method: "ExplainInstallation"}
	if !issuanceProfileExplanation(good) {
		t.Fatal("exact standing site refused")
	}
	for _, field := range []string{"file", "function", "receiver", "method"} {
		bad := good
		switch field {
		case "file":
			bad.file = "accessrequests.go"
		case "function":
			bad.function = "termCaller"
		case "receiver":
			bad.receiver = "other.admin"
		case "method":
			bad.method = "Explain"
		}
		if issuanceProfileExplanation(bad) {
			t.Fatalf("broad %s exception", field)
		}
	}
}
