package server

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func TestRetentionArtifactsOfAClassAreReadByIdentifier(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	before := whole(retGroup(t, f.retentionRead(root), retention.ClassConversation)["total"])
	for n := 0; n < 120; n++ {
		f.retRegister(fmt.Sprintf("zz-conv-%03d", n), retention.ClassConversation, 1, SampleWorkspace)
	}
	for n := 0; n < 10; n++ {
		f.retRegister(fmt.Sprintf("zz-elsewhere-%03d", n), retention.ClassConversation, 1, "other-workspace")
	}
	f.retRegister("zz-conv-shared", retention.ClassConversation, 1, "other-workspace", SampleWorkspace)
	total := before + 121
	got := f.retentionRead(root)
	group := retGroup(t, got, retention.ClassConversation)
	main, _ := group["artifacts"].([]any)
	if len(main) != retentionShown || whole(group["total"]) != total || group["more"] != true || group["next"] != main[len(main)-1].(map[string]any)["id"] {
		t.Fatalf("the main read lists the first fifty and says where it stopped: %d of %v, next %v", len(main), group["total"], group["next"])
	}
	for _, class := range retention.Classes {
		if g := retGroup(t, got, class); g["more"] == false && g["next"] != "" {
			t.Errorf("%s has a cursor without more rows: %v", class, g["next"])
		}
	}
	path := "/api/v1/retention/artifacts"
	ids, next, pages := []string{}, fmt.Sprint(group["next"]), 0
	for _, raw := range main {
		ids = append(ids, fmt.Sprint(raw.(map[string]any)["id"]))
	}
	for next != "" {
		page := f.readAs(root, path+"?class="+retention.ClassConversation+"&limit=30&after="+next)
		if page.status != 200 || whole(page.body["total"]) != total || whole(page.body["matching"]) != total || page.body["order"] != "identifier" || page.body["class"] != retention.ClassConversation {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["artifacts"].([]any) {
			row := raw.(map[string]any)
			if row["class"] != retention.ClassConversation {
				t.Fatalf("a row of another class: %v", row["id"])
			}
			ids = append(ids, fmt.Sprint(row["id"]))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages > 10 {
			t.Fatal("the artifacts do not end")
		}
	}
	if len(ids) != total {
		t.Fatalf("the pages hold %d of %d", len(ids), total)
	}
	for index, id := range ids {
		if strings.HasPrefix(id, "zz-elsewhere-") || index > 0 && id <= ids[index-1] {
			t.Fatalf("row %d is %s after %s", index, id, ids[max(index-1, 0)])
		}
	}
	first := f.readAs(root, path+"?class="+retention.ClassConversation)
	rows, _ := first.body["artifacts"].([]any)
	if len(rows) != retentionShown {
		t.Fatalf("the first page holds %d", len(rows))
	}
	for _, index := range []int{0, 49} {
		row := rows[index].(map[string]any)
		delete(row, "class")
		if !reflect.DeepEqual(row, main[index]) {
			t.Errorf("row %d of the further read differs from the main read's", index)
		}
	}
	byID := f.readAs(root, path+"?id=zz-conv-119,sample-source-1,zz-elsewhere-000,never-registered,zz-conv-shared")
	found, _ := byID.body["artifacts"].([]any)
	if byID.status != 200 || len(found) != 3 || whole(byID.body["matching"]) != 3 || byID.body["next"] != "" || found[0].(map[string]any)["id"] != "sample-source-1" ||
		found[0].(map[string]any)["class"] != retention.ClassSource || found[2].(map[string]any)["shared"] != true || strings.Contains(byID.raw, "elsewhere") {
		t.Fatalf("by identifier: %s", byID.raw[:min(len(byID.raw), 400)])
	}
	search := f.readAs(root, path+"?class="+retention.ClassConversation+"&q=CONV-07&limit=4")
	if rows, _ := search.body["artifacts"].([]any); search.status != 200 || len(rows) != 4 || whole(search.body["matching"]) != 10 || search.body["next"] != "zz-conv-073" || whole(search.body["total"]) != total {
		t.Errorf("a search: %s", search.raw[:min(len(search.raw), 300)])
	}
	if hidden := f.readAs(root, path+"?class="+retention.ClassConversation+"&q=elsewhere"); whole(hidden.body["matching"]) != 0 {
		t.Errorf("a search reaches another workspace's artifacts: %v", hidden.body["matching"])
	}
	many := []string{}
	for n := 0; n <= retentionIDs; n++ {
		many = append(many, fmt.Sprintf("zz-conv-%03d", n))
	}
	for _, query := range []string{"", "?class=", "?class=unknown", "?after=zz", "?class=" + retention.ClassConversation + "&after=", "?class=" + retention.ClassConversation + "&after=a,b",
		"?class=" + retention.ClassConversation + "&limit=0", "?class=" + retention.ClassConversation + "&limit=51", "?class=" + retention.ClassConversation + "&q=",
		"?class=" + retention.ClassConversation + "&class=" + retention.ClassClaim, "?class=" + retention.ClassConversation + "&state=x", "?id=", "?id=a,,b", "?id=a,a", "?id=has%20space",
		"?id=" + strings.Repeat("p", retentionIDBytes+1), "?id=" + strings.Join(many, ","), "?id=a&class=" + retention.ClassConversation, "?id=a&id=b"} {
		if refused := f.readAs(root, path+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	if twenty := f.readAs(root, path+"?id="+strings.Join(many[:retentionIDs], ",")); twenty.status != 200 || len(twenty.body["artifacts"].([]any)) != retentionIDs {
		t.Errorf("twenty identifiers: %d", twenty.status)
	}
	ok := path + "?class=" + retention.ClassConversation
	f.code(t, "member", f.readAs(f.as(MemberLogin), ok), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, ok), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if failed := f.readAs(root, ok); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, ok); out.status != 200 || len(out.body["artifacts"].([]any)) != 0 || whole(out.body["total"]) != 0 {
		t.Errorf("another workspace: %d", out.status)
	}
}

func TestRetentionReferencesAreReadByIdentifier(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	before := whole(retSection(f.retentionRead(root), "holds")["references"].(map[string]any)["total"])
	f.retRegister("yy-from", retention.ClassConversation, 1, SampleWorkspace)
	for n := 0; n < 70; n++ {
		id := fmt.Sprintf("yy-ref-%03d", n)
		f.retRegister(id, retention.ClassConversation, 1, SampleWorkspace)
		if err := f.s.retention.Refer(retention.Reference{From: "yy-from", To: id, Kind: retention.Requires}, retIngest); err != nil {
			t.Fatal(err)
		}
	}
	total := before + 70
	refs := retSection(f.retentionRead(root), "holds")["references"].(map[string]any)
	main := retRows(refs)
	if len(main) != retentionShown || whole(refs["total"]) != total || refs["more"] != true || refs["next"] != main[len(main)-1]["id"] {
		t.Fatalf("the main read lists the first fifty and says where it stopped: %d of %v, next %v", len(main), refs["total"], refs["next"])
	}
	path := "/api/v1/retention/references"
	ids, next, pages := []string{}, fmt.Sprint(refs["next"]), 0
	for _, row := range main {
		ids = append(ids, fmt.Sprint(row["id"]))
	}
	for next != "" {
		page := f.readAs(root, path+"?limit=15&after="+next)
		if page.status != 200 || whole(page.body["total"]) != total || whole(page.body["matching"]) != total || page.body["order"] != "identifier" {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["references"].([]any) {
			ids = append(ids, fmt.Sprint(raw.(map[string]any)["id"]))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages > 10 {
			t.Fatal("the references do not end")
		}
	}
	if len(ids) != total {
		t.Fatalf("the pages hold %d of %d", len(ids), total)
	}
	for index := 1; index < len(ids); index++ {
		if ids[index] <= ids[index-1] {
			t.Fatalf("row %d is %s after %s", index, ids[index], ids[index-1])
		}
	}
	first := f.readAs(root, path)
	if rows, _ := first.body["references"].([]any); len(rows) != retentionShown || !reflect.DeepEqual(rows[0], map[string]any(main[0])) || !reflect.DeepEqual(rows[49], map[string]any(main[49])) {
		t.Error("the first page of the further read differs from the main read's rows")
	}
	if search := f.readAs(root, path+"?q=REF-06"); whole(search.body["matching"]) != 10 || len(search.body["references"].([]any)) != 10 {
		t.Errorf("a search: %s", search.raw[:min(len(search.raw), 200)])
	}
	for _, query := range []string{"?class=" + retention.ClassConversation, "?id=yy-ref-001", "?after=", "?after=a,b", "?limit=0", "?limit=51", "?q=", "?limit=5&limit=6"} {
		if refused := f.readAs(root, path+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, path); out.status != 200 || len(out.body["references"].([]any)) != 0 || whole(out.body["total"]) != 0 {
		t.Errorf("another workspace: %d", out.status)
	}
}
