package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
)

func (f world) delegationRow(a actor, id string) map[string]any {
	f.t.Helper()
	for _, d := range f.accessRead(a, "").body["delegations"].([]any) {
		if row := d.(map[string]any); row["id"] == id {
			return row
		}
	}
	return nil
}

func endedOf(row map[string]any) string {
	out := []string{}
	for _, id := range row["ended_delegations"].([]any) {
		out = append(out, id.(string))
	}
	return strings.Join(out, ",")
}

func TestRevokingTheDelegatorsBackingGrantEndsTheDelegation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	addDelegation(f, "del-1", SampleAdmin2, SampleAdmin, "")
	asked := f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil)
	if row := f.requestRow(second, requestOf(asked)); strings.Join(func() []string {
		out := []string{}
		for _, id := range row["may_end"].([]any) {
			out = append(out, id.(string))
		}
		return out
	}(), ",") != "del-1" {
		t.Errorf("may end %+v", row["may_end"])
	}
	if got := f.approveRequest(second, asked); got.status != 202 {
		t.Fatal(got.raw)
	}
	rows := f.revocationRows(root)
	if endedOf(rows[0]) != "del-1" || rows[0]["state"] != "revocation_complete" {
		t.Errorf("ended %+v", rows[0])
	}
	var c9 map[string]any
	for _, it := range rows[0]["cleanup"].([]any) {
		if item := it.(map[string]any); item["item"] == "C9" {
			c9 = item
		}
	}
	if c9["state"] != "done" || len(c9["refs"].([]any)) != 1 {
		t.Errorf("C9 %+v", c9)
	}
	if d := f.delegationRow(root, "del-1"); d["state"] != "ended" || d["effective"] != false {
		t.Errorf("delegation row %+v", d)
	}
}

func TestRevokingTheDelegatesOnlyGrantEndsTheDelegation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	made := f.grantVia(root, second, SampleMember, nil)
	addDelegation(f, "del-m", SampleAdmin, SampleMember, "")
	addDelegation(f, "del-u", SampleAdmin, SampleAdmin2, "")
	if got := f.revokeVia(second, root, made); got.status != 202 {
		t.Fatal(got.raw)
	}
	if rows := f.revocationRows(root); endedOf(rows[0]) != "del-m" {
		t.Errorf("ended %+v", rows[0])
	}
	if d := f.delegationRow(root, "del-m"); d["state"] != "ended" {
		t.Errorf("delegation row %+v", d)
	}
}

func TestRevokedDelegationEndsItsDescendants(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.grantVia(root, second, SampleMember, nil)
	addDelegation(f, "del-1", SampleAdmin, SampleAdmin2, "")
	addDelegation(f, "del-2", SampleAdmin2, SampleMember, "del-1")
	got := f.run(root, "access.revoke_delegation", "del-1", nil)
	if got.status != 202 {
		t.Fatal(got.raw)
	}
	rows := f.revocationRows(root)
	if endedOf(rows[0]) != "del-2" || rows[0]["target_kind"] != "delegation" || rows[0]["requested_by"] != SampleAdmin || rows[0]["approved_by"] != "" {
		t.Errorf("descendants %+v", rows[0])
	}
	if d := f.delegationRow(root, "del-2"); d["state"] != "ended" {
		t.Errorf("descendant %+v", d)
	}
	if raw, _ := os.ReadFile(ledgerFile(f)); !strings.Contains(string(raw), `"decision":"revoke_delegation","target":"del-1"`) {
		t.Errorf("ledger %s", raw)
	}
	if d := f.delegationRow(root, "del-1"); d["state"] != "revoked" {
		t.Errorf("revoked %+v", d)
	}
}

func TestReadDuringRevocations(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if got := f.accessRead(root, ""); got.status != 200 {
					t.Errorf("read %d", got.status)
				}
			}
		}()
	}
	made := f.grantVia(root, second, SampleMember, nil)
	if got := f.revokeVia(root, second, made); got.status != 202 {
		t.Error(got.raw)
	}
	wg.Wait()
}

func TestGrantChangesHaveFixedCallSites(t *testing.T) {
	allowed := map[string]bool{
		"accessrequests.go applyAdd AddGrant":               true,
		"revocations.go grantEffects RevokeGrant":           true,
		"revocations.go delegationEffects RevokeDelegation": true,
		"ledger.go replayOne RevokeGrant":                   true,
		"ledger.go replayOne RevokeDelegation":              true,
		"seed.go seedReaderGrants AddGrant":                 true,
		"seed.go grantInstallation AddGrant":                true,
		"seed.go grantTrust AddGrant":                       true,
		"seed.go grantBulk AddGrant":                        true,
		"integrationrights.go grantIntegration AddGrant":    true,
		"integrationrights.go rollbackGrants RevokeGrant":   true,
	}
	entries, _ := os.ReadDir(".")
	found := []string{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
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
				call, isCall := n.(*ast.CallExpr)
				var sel *ast.SelectorExpr
				if isCall {
					sel, _ = call.Fun.(*ast.SelectorExpr)
				}
				if sel != nil && (sel.Sel.Name == "AddGrant" || sel.Sel.Name == "RevokeGrant" || sel.Sel.Name == "RevokeDelegation") {
					site := name + " " + fn.Name.Name + " " + sel.Sel.Name
					found = append(found, site)
					if !allowed[site] {
						t.Errorf("unexpected grant change at %s", site)
					}
				}
				if fn.Name.Name == "Administration" {
					if ret, ok := n.(*ast.ReturnStmt); ok && (len(ret.Results) != 1 || !strings.Contains(types(ret.Results[0]), "admin")) {
						t.Error("the Administration accessor changed")
					}
				}
				return true
			})
		}
	}
	sort.Strings(found)
	unique := map[string]bool{}
	for _, site := range found {
		unique[site] = true
	}
	if len(unique) != len(allowed) {
		t.Errorf("sites %v", found)
	}
}

func types(e ast.Expr) string {
	if sel, ok := e.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	return ""
}
