package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
)

func (f world) contractsRead(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/contracts"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) contract(a actor, id, query string) reply {
	f.t.Helper()
	return f.contractsRead(a, "/"+id+query)
}

func contractRowOf(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	for _, c := range got.body["contracts"].([]any) {
		if row := c.(map[string]any); row["id"] == id {
			return row
		}
	}
	t.Fatalf("no row %s in %v", id, got.raw)
	return nil
}

func actionOf(detail reply, command string) map[string]any {
	return detail.body["actions"].(map[string]any)[command].(map[string]any)
}

func ok(t testing.TB, name string, got reply) map[string]any {
	t.Helper()
	if got.status != 200 {
		t.Fatalf("%s: %d %s", name, got.status, got.raw)
	}
	return got.body["fields"].(map[string]any)
}

func TestContractsListShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.contractsRead(root, "")
	if got.status != 200 || got.body["on"] != "2026-11-16" || got.body["today"] != "2026-11-16" || len(got.body["contracts"].([]any)) != 5 ||
		fmt.Sprint(got.body["impact"]) != "map[erp:not_computed final_amount_decision:not_computed money:not_computed proration:not_computed publication:not_computed]" {
		t.Fatalf("list %+v", got.raw)
	}
	want := map[string]string{
		"sample-contract-01": "active 2 none 2026-01-01  0",
		"sample-contract-02": "suspended 1 none 2026-01-01  0",
		"sample-contract-03": "ended 0 pending_final 2026-03-01 2026-10-01 1",
		"sample-contract-04": "ended 0 settled 2026-02-01 2026-08-01 0",
		"sample-contract-05": "not_started 0 none 2027-01-01  0",
	}
	ids := []string{}
	for _, c := range got.body["contracts"].([]any) {
		row := c.(map[string]any)
		ids = append(ids, row["id"].(string))
		if have := fmt.Sprintf("%v %v %v %v %v %v", row["state"], row["lifecycle_revision"], row["settlement"], row["start"], row["end"], row["open_gaps"]); have != want[row["id"].(string)] {
			t.Errorf("%s: %s", row["id"], have)
		}
	}
	if strings.Join(ids, ",") != "sample-contract-01,sample-contract-02,sample-contract-03,sample-contract-04,sample-contract-05" {
		t.Errorf("order %v", ids)
	}
	first := contractRowOf(t, got, "sample-contract-01")
	if first["name"] != "North managed cloud (sample)" || first["profile"] != "sample-profile-north" || first["profile_name"] != "North invoice, USD (sample)" ||
		first["buyer"] != "sample-le-north" || first["payer"] != "sample-le-north" || first["issuer"] != "sample-le-seller" || first["path"] != "external_invoice" ||
		first["currency"] != "USD" || first["timezone"] != "UTC" || first["cycle"] != "monthly" || fmt.Sprint(first["sequence"]) != "1" {
		t.Errorf("row %+v", first)
	}
	past := f.contractsRead(root, "?on=2026-06-15")
	if row := contractRowOf(t, past, "sample-contract-01"); row["state"] != "active" || fmt.Sprint(row["lifecycle_revision"]) != "1" {
		t.Errorf("june %+v", row)
	}
	if row := contractRowOf(t, past, "sample-contract-03"); row["state"] != "active" {
		t.Errorf("june 03 %+v", row)
	}
	for _, q := range []string{"?on=2026-02-30", "?on=x", "?on=2026-1-01"} {
		f.code(t, q, f.contractsRead(root, q), 422, "invalid_request")
	}
}

func TestContractDetailShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.contract(root, "sample-contract-01", "")
	if got.status != 200 || got.body["from"] != "2026-11-01" || got.body["until"] != "2026-12-01" || got.body["today"] != "2026-11-16" ||
		got.body["contract"].(map[string]any)["id"] != "sample-contract-01" || fmt.Sprint(got.body["reopenings"]) != "0" || fmt.Sprint(got.body["earlier_records"]) != "0" {
		t.Fatalf("detail %s", got.raw)
	}
	if s := fmt.Sprint(got.body["segments"]); s != "[map[days:30 from:2026-11-01 revision:2 state:active until:2026-12-01]]" {
		t.Errorf("segments %s", s)
	}
	if s := fmt.Sprint(got.body["events"]); s != "[map[cancel_required:false from:2026-01-01 id:ev-01-start kind:start revision:1 withdraw_blocked:not_withdrawable] map[cancel_required:false from:2026-07-01 id:ev-01-change kind:change revision:2 withdraw_blocked:not_withdrawable]]" {
		t.Errorf("events %s", s)
	}
	cloud := "map[id:cloud-usage kind:provider_charge provider_payment_responsible:sample-le-seller seller_billable:true]"
	managed := "map[id:managed-service kind:seller_service provider_payment_responsible: seller_billable:false]"
	support := "map[id:support-plus kind:seller_service provider_payment_responsible: seller_billable:false]"
	if s := fmt.Sprint(got.body["terms"]); s != "[map[components:["+cloud+" "+managed+"] customer:sample-le-north path:resale revision:1 seller:sample-le-seller] map[components:["+cloud+" "+managed+" "+support+"] customer:sample-le-north path:resale revision:2 seller:sample-le-seller]]" {
		t.Errorf("terms %s", s)
	}
	records := got.body["records"].([]any)
	if r := records[0].(map[string]any); len(records) != 1 || r["actor"] != "seed" || r["actor_login"] != "" || r["reason"] != "sample seed" ||
		fmt.Sprint(r["added"]) != "[ev-01-start ev-01-change]" || r["kind"] != "batch" || r["state"] != "none" || r["write"] != "cw-01-seed" || r["at"] != stamp(start) {
		t.Errorf("records %+v", records)
	}
	want := map[string]string{
		"contract.suspend": "true  2026-11-16", "contract.resume": "false not_suspended ", "contract.renew": "true  2026-11-16", "contract.end": "true  2026-11-16",
		"contract.withdraw": "false nothing_withdrawable ", "contract.evidence.secured": "false not_pending_final ", "contract.evidence.missing": "true  ",
		"contract.settle": "false not_pending_final ",
		"contract.reopen": "false not_settled ",
	}
	for command, w := range want {
		a := actionOf(got, command)
		if have := fmt.Sprintf("%v %v %v", a["allowed"], a["blocked"], a["earliest"]); have != w {
			t.Errorf("%s: %s", command, have)
		}
	}
	span := f.contract(root, "sample-contract-01", "?from=2026-06-01&until=2026-08-01")
	if s := fmt.Sprint(span.body["segments"]); s != "[map[days:30 from:2026-06-01 revision:1 state:active until:2026-07-01] map[days:31 from:2026-07-01 revision:2 state:active until:2026-08-01]]" {
		t.Errorf("span %s", s)
	}
	month := f.contract(root, "sample-contract-02", "?from=2026-12-15")
	if month.body["until"] != "2027-01-01" {
		t.Errorf("default until %s", month.raw)
	}
	ended := f.contract(root, "sample-contract-04", "")
	if a := actionOf(ended, "contract.end"); a["blocked"] != "settled" {
		t.Errorf("settled end %+v", a)
	}
	if a := actionOf(ended, "contract.reopen"); a["allowed"] != true {
		t.Errorf("reopen %+v", a)
	}
	ev := ended.body["evidence"].([]any)[0].(map[string]any)
	if ev["kind"] != "secured" || ev["reference"] != "sample-final-04" || ev["covered_until"] != "2026-08-01" || ev["write"] != "cw-04-final" {
		t.Errorf("evidence %+v", ev)
	}
	gap := f.contract(root, "sample-contract-03", "")
	g := gap.body["evidence"].([]any)[0].(map[string]any)
	if g["kind"] != "missing" || g["owner"] != SampleAdmin || g["owner_login"] != AdminLogin || g["follow_up"] != "sample-followup-03" || g["resolved"] != false ||
		g["reason"] != "source_unavailable" || g["covered_until"] != "" {
		t.Errorf("gap %+v", g)
	}
	if a := actionOf(gap, "contract.settle"); a["blocked"] != "open_gaps" {
		t.Errorf("gap settle %+v", a)
	}
	if a := actionOf(gap, "contract.suspend"); a["blocked"] != "ended" {
		t.Errorf("ended suspend %+v", a)
	}
	future := f.contract(root, "sample-contract-05", "")
	if a := actionOf(future, "contract.suspend"); a["allowed"] != true || a["earliest"] != "2027-01-02" {
		t.Errorf("future suspend %+v", a)
	}
	for _, q := range []string{"?from=2026-13-01", "?from=2026-12-01&until=2026-12-01", "?from=2026-01-01&until=2027-01-03", "?until=bad", "?from=2026-12-01&until=2026-11-01"} {
		f.code(t, q, f.contract(root, "sample-contract-01", q), 422, "invalid_request")
	}
	f.code(t, "unknown", f.contract(root, "sample-contract-99", ""), 404, "unsupported_command")
	f.code(t, "bad id", f.contract(root, "bad%20id", ""), 404, "unsupported_command")
}

func TestContractSuspendResumeWithdraw(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-01"
	got := ok(t, "suspend", f.run(root, "contract.suspend", id, map[string]string{"from": "2026-12-01"}))
	suspend := got["event"].(string)
	if !strings.HasPrefix(suspend, "ev-") || got["sequence"] != "2" || !strings.HasPrefix(got["write"].(string), "cw-") {
		t.Fatalf("fields %+v", got)
	}
	detail := f.contract(root, id, "?from=2026-11-01&until=2027-01-01")
	if s := fmt.Sprint(detail.body["segments"]); s != "[map[days:30 from:2026-11-01 revision:2 state:active until:2026-12-01] map[days:31 from:2026-12-01 revision:2 state:suspended until:2027-01-01]]" {
		t.Errorf("segments %s", s)
	}
	record := detail.body["records"].([]any)[0].(map[string]any)
	if record["actor"] != SampleAdmin || record["actor_login"] != AdminLogin || record["reason"] != "sample reason" || fmt.Sprint(record["added"]) != "["+suspend+"]" {
		t.Errorf("record %+v", record)
	}
	if a := actionOf(detail, "contract.resume"); a["allowed"] != true || a["earliest"] != "2026-12-02" {
		t.Errorf("resume hint %+v", a)
	}
	if a := actionOf(detail, "contract.suspend"); a["blocked"] != "not_active" {
		t.Errorf("suspend hint %+v", a)
	}
	f.code(t, "same day", f.run(root, "contract.resume", id, map[string]string{"from": "2026-12-01"}), 422, "invalid_stage")
	resume := ok(t, "resume", f.run(root, "contract.resume", id, map[string]string{"from": "2026-12-15"}))["event"].(string)
	f.code(t, "withdraw middle", f.run(root, "contract.withdraw", id, map[string]string{"event": suspend}), 422, "invalid_stage")
	for _, e := range f.contract(root, id, "").body["events"].([]any) {
		if row := e.(map[string]any); row["id"] == suspend && (row["withdraw_blocked"] != "breaks_lifecycle" || row["cancel_required"] != false) {
			t.Errorf("middle hint %+v", row)
		}
	}
	f.code(t, "withdraw start", f.run(root, "contract.withdraw", id, map[string]string{"event": "ev-01-start"}), 422, "invalid_request")
	f.code(t, "withdraw unknown", f.run(root, "contract.withdraw", id, map[string]string{"event": "ev-none"}), 422, "invalid_request")
	f.code(t, "cancel not needed", f.run(root, "contract.withdraw", id, map[string]string{"event": resume, "cancel": "recorded_in_error"}), 422, "invalid_request")
	ok(t, "withdraw resume", f.run(root, "contract.withdraw", id, map[string]string{"event": resume}))
	ok(t, "withdraw suspend", f.run(root, "contract.withdraw", id, map[string]string{"event": suspend}))
	detail = f.contract(root, id, "")
	if len(detail.body["events"].([]any)) != 2 || fmt.Sprint(detail.body["records"].([]any)[0].(map[string]any)["withdrawn"]) != "["+suspend+"]" {
		t.Errorf("after withdraw %s", detail.raw)
	}
	for _, from := range []string{"2026-11-15", "2026-11-31", "soon", "2036-11-25", ""} {
		got := f.run(root, "contract.renew", id, map[string]string{"from": from})
		if from == "" {
			f.code(t, "missing", got, 422, "invalid_request")
			continue
		}
		f.code(t, from, got, 422, "invalid_request")
		if fmt.Sprint(got.body["fields"]) != "[from]" {
			t.Errorf("%s field %v", from, got.body["fields"])
		}
	}
	ok(t, "renew today", f.run(root, "contract.renew", id, map[string]string{"from": "2026-11-16"}))
	f.code(t, "suspend contract-03", f.run(root, "contract.suspend", "sample-contract-03", map[string]string{"from": "2026-12-01"}), 422, "invalid_stage")
	f.code(t, "unknown target", f.run(root, "contract.suspend", "sample-contract-99", map[string]string{"from": "2026-12-01"}), 403, "permission_denied")
}

func TestContractEndSettleReopen(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-02"
	end := ok(t, "end", f.run(root, "contract.end", id, map[string]string{"from": "2026-11-20"}))
	if got := f.run(root, "contract.end", id, map[string]string{"from": "2026-11-25"}); got.status != 422 {
		t.Errorf("second end %s", got.raw)
	}
	if row := contractRowOf(t, f.contractsRead(root, ""), id); row["settlement"] != "pending_final" || row["end"] != "2026-11-20" || row["state"] != "suspended" {
		t.Errorf("row %+v", row)
	}
	f.code(t, "settle without evidence", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-16"}), 422, "prerequisite_failed")
	f.code(t, "withdraw without cancel", f.run(root, "contract.withdraw", id, map[string]string{"event": end["event"].(string)}), 422, "invalid_request")
	f.code(t, "future coverage", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-02", "covered_until": "2026-11-20"}), 422, "invalid_request")
	f.code(t, "end not reached", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-02", "covered_until": "2026-11-16"}), 422, "invalid_stage")
	if a := actionOf(f.contract(root, id, ""), "contract.evidence.secured"); a["blocked"] != "end_not_reached" {
		t.Errorf("evidence hint %+v", a)
	}
	ok(t, "gap before end", f.run(root, "contract.evidence", id, map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f-02"}))
	f.advance(4 * 24 * time.Hour)
	root = f.as(AdminLogin)
	ok(t, "secured", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-02", "covered_until": "2026-11-20", "resolves": f.contract(root, id, "").body["evidence"].([]any)[0].(map[string]any)["write"].(string)}))
	f.code(t, "future as_of", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-21"}), 422, "invalid_request")
	f.code(t, "early as_of", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-19"}), 422, "prerequisite_failed")
	f.advance(24 * time.Hour)
	root = f.as(AdminLogin)
	if got := f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-20"}); got.status != 200 || got.body["business_state"] != "settled" {
		t.Fatalf("settle %s", got.raw)
	}
	f.code(t, "missing on settled", f.run(root, "contract.evidence", id, map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "late"}), 422, "invalid_stage")
	if a := actionOf(f.contract(root, id, ""), "contract.evidence.missing"); a["blocked"] != "settled" {
		t.Errorf("missing hint %+v", a)
	}
	f.code(t, "bad as_of", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-31"}), 422, "invalid_request")
	f.code(t, "settled renew", f.run(root, "contract.renew", id, map[string]string{"from": "2026-12-01"}), 422, "invalid_stage")
	f.code(t, "settled evidence", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-02b", "covered_until": "2026-11-20"}), 422, "invalid_stage")
	f.code(t, "settle twice", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-20"}), 422, "invalid_stage")
	if got := f.run(root, "contract.reopen", id, map[string]string{"reason": "late_source"}); got.status != 200 || got.body["business_state"] != "pending_final" {
		t.Fatalf("reopen %s", got.raw)
	}
	f.code(t, "reopen twice", f.run(root, "contract.reopen", id, map[string]string{"reason": "late_credit"}), 422, "invalid_stage")
	f.code(t, "stale evidence", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-20"}), 422, "prerequisite_failed")
	if a := actionOf(f.contract(root, id, ""), "contract.settle"); a["blocked"] != "evidence_needed" {
		t.Errorf("settle hint after reopen %+v", a)
	}
	ok(t, "secured again", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-02c", "covered_until": "2026-11-21"}))
	ok(t, "settle again", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-21"}))
	if detail := f.contract(root, id, ""); fmt.Sprint(detail.body["reopenings"]) != "1" || detail.body["contract"].(map[string]any)["settlement"] != "settled" {
		t.Errorf("detail %s", detail.raw)
	}
}

func TestContractWithdrawEndNeedsCancel(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-01"
	end := ok(t, "end", f.run(root, "contract.end", id, map[string]string{"from": "2026-12-31"}))["event"].(string)
	f.code(t, "no cancel", f.run(root, "contract.withdraw", id, map[string]string{"event": end}), 422, "invalid_request")
	f.code(t, "bad cancel", f.run(root, "contract.withdraw", id, map[string]string{"event": end, "cancel": "other"}), 422, "invalid_request")
	got := f.run(root, "contract.withdraw", id, map[string]string{"event": end, "cancel": "contract_continues"})
	if got.status != 200 || got.body["business_state"] != "none" {
		t.Fatalf("withdraw %s", got.raw)
	}
	if r := f.contract(root, id, "").body["records"].([]any)[0].(map[string]any); r["cancel"] != "contract_continues" {
		t.Errorf("record %+v", r)
	}
}

func TestContractEvidenceRules(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-03"
	bad := []struct {
		fields map[string]string
		field  string
	}{
		{map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01", "reason": "not_attempted"}, "reason"},
		{map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01", "follow_up": "x"}, "follow_up"},
		{map[string]string{"kind": "secured", "reference": "has space", "covered_until": "2026-10-01"}, "reference"},
		{map[string]string{"kind": "secured", "covered_until": "2026-10-01"}, "reference"},
		{map[string]string{"kind": "secured", "reference": "r", "covered_until": "Oct"}, "covered_until"},
		{map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01", "resolves": "a,a"}, "resolves"},
		{map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01", "resolves": "a,"}, "resolves"},
		{map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01", "resolves": "a,b,c,d,e,f,g,h,i"}, "resolves"},
		{map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01", "resolves": "cw-none"}, "resolves"},
		{map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f", "reference": "r"}, "reference"},
		{map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f", "covered_until": "2026-10-01"}, "covered_until"},
		{map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f", "resolves": "a"}, "resolves"},
		{map[string]string{"kind": "missing", "follow_up": "f"}, "reason"},
		{map[string]string{"kind": "missing", "reason": "not_attempted"}, "follow_up"},
		{map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "https://x"}, "follow_up"},
	}
	for index, c := range bad {
		got := f.run(root, "contract.evidence", id, c.fields)
		if got.status != 422 || fmt.Sprint(got.body["fields"]) != "["+c.field+"]" {
			t.Errorf("%d: %s", index, got.raw)
		}
	}
	gap := ok(t, "missing", f.run(f.as(Admin2Login), "contract.evidence", id, map[string]string{"kind": "missing", "reason": "provider_refused", "follow_up": "ticket-7"}))
	detail := f.contract(root, id, "")
	second := detail.body["evidence"].([]any)[1].(map[string]any)
	if second["owner"] != SampleAdmin2 || second["owner_login"] != Admin2Login || second["write"] != gap["write"] {
		t.Errorf("owner %+v", second)
	}
	ok(t, "resolve both", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-03", "covered_until": "2026-10-01",
		"resolves": "cw-03-gap," + gap["write"].(string)}))
	f.code(t, "resolve twice", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-03b", "covered_until": "2026-10-01",
		"resolves": "cw-03-gap"}), 422, "invalid_request")
	if got := f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "final-03c", "covered_until": "2026-10-01"}); got.status != 200 {
		t.Fatal(got.raw)
	}
	ok(t, "settle", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-10-01"}))
	f.code(t, "secured on none", f.run(root, "contract.evidence", "sample-contract-01", map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01"}), 422, "invalid_stage")
	ok(t, "missing on none", f.run(root, "contract.evidence", "sample-contract-01", map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f"}))
}

func TestContractAccessAndBoundary(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	f.code(t, "member list", f.contractsRead(member, ""), 403, "permission_denied")
	f.code(t, "member detail", f.contract(member, "sample-contract-01", ""), 403, "permission_denied")
	f.code(t, "anonymous", f.contractsRead(actor{}, ""), 401, "unauthenticated")
	if got := f.runWith(member, f.revision(f.as(AdminLogin)), "contract.suspend", "sample-contract-01", map[string]string{"from": "2026-12-01"}); got.status != 403 {
		t.Errorf("member command %s", got.raw)
	}
	f.s.contracts.byKey[planKey("other-ws", "sample-contract-01")] = &contractMeta{workspace: "other-ws", id: "sample-contract-01", name: "x"}
	f.s.contracts.byKey[planKey("other-ws", "other-contract")] = &contractMeta{workspace: "other-ws", id: "other-contract", name: "x"}
	root := f.as(AdminLogin)
	if n := len(f.contractsRead(root, "").body["contracts"].([]any)); n != 5 {
		t.Errorf("other workspace leaked %d", n)
	}
	f.code(t, "other detail", f.contract(root, "other-contract", ""), 404, "unsupported_command")
	h := lifecycleCommand{s: f.s, kind: "end"}
	if _, res := h.Resolve("other-ws", "other-contract"); res != commands.NotFound {
		t.Error("other workspace resolved")
	}
	if _, res := h.Resolve(SampleWorkspace, "sample-contract-01"); res != commands.Found {
		t.Error("own contract not resolved")
	}
	long := strings.Repeat("r", maxRationale+1)
	f.code(t, "long reason", f.admin(root, "contract.renew", "sample-contract-01", long, map[string]string{"from": "2026-12-01"}), 422, "invalid_request")
}

func TestContractStaleAndReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	old := f.revision(root)
	ok(t, "first", f.runWith(root, old, "contract.suspend", "sample-contract-01", map[string]string{"from": "2026-12-01"}))
	f.code(t, "stale", f.runWith(root, old, "contract.end", "sample-contract-01", map[string]string{"from": "2026-12-20"}), 409, "revision_conflict")
	body := `{"request_id":"req-r","target_id":"sample-contract-01","idempotency_key":"key-replay","expected_revision":"` + f.revision(root) +
		`","rationale":"r","body":{"from":"2026-12-10"}}`
	first := f.call("contract.resume", body, root.cookie, root.csrf)
	again := f.call("contract.resume", body, root.cookie, root.csrf)
	if first.status != 200 || again.status != 200 || fmt.Sprint(first.body["fields"]) != fmt.Sprint(again.body["fields"]) {
		t.Fatalf("replay %s %s", first.raw, again.raw)
	}
	if seq, _ := f.s.lifecycles.Sequence(SampleWorkspace, "sample-contract-01"); seq != 3 {
		t.Errorf("sequence %d", seq)
	}
}

func TestContractFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	end := ok(t, "end", f.run(root, "contract.end", "sample-contract-01", map[string]string{"from": "2026-12-31"}))["event"].(string)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	at = "contract-read"
	f.code(t, "list", f.contractsRead(root, ""), 503, "dependency_unavailable")
	f.code(t, "detail", f.contract(root, "sample-contract-01", ""), 503, "dependency_unavailable")
	cases := []struct {
		step, command, target string
		fields                map[string]string
	}{
		{"contract-sequence", "contract.renew", "sample-contract-02", map[string]string{"from": "2026-12-01"}},
		{"contract-amend", "contract.renew", "sample-contract-02", map[string]string{"from": "2026-12-01"}},
		{"contract-amend", "contract.withdraw", "sample-contract-01", map[string]string{"event": end, "cancel": "contract_continues"}},
		{"contract-evidence", "contract.evidence", "sample-contract-03", map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f"}},
		{"contract-settle", "contract.settle", "sample-contract-03", map[string]string{"as_of": "2026-10-01"}},
		{"contract-reopen", "contract.reopen", "sample-contract-04", map[string]string{"reason": "correction"}},
		{"contract-read", "contract.withdraw", "sample-contract-01", map[string]string{"event": "ev-01-start"}},
		{"contract-read", "contract.evidence", "sample-contract-03", map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-10-01"}},
	}
	for _, c := range cases {
		at = c.step
		got := f.run(root, c.command, c.target, c.fields)
		if c.step == "contract-sequence" || c.step == "contract-read" {
			f.code(t, c.step+" "+c.command, got, 503, "dependency_unavailable")
			continue
		}
		f.code(t, c.step+" "+c.command, got, 500, "internal_error")
	}
	at = ""
	f.s.fault = nil
}

func TestMapContractErrors(t *testing.T) {
	cases := map[error]commands.Code{
		contracts.ErrRejected: commands.InvalidStage, contracts.ErrState: commands.InvalidStage, contracts.ErrLimit: commands.PrerequisiteFailed,
		contracts.ErrConflict: commands.RevisionConflict, contracts.ErrInvalid: commands.InternalError, contracts.ErrNotFound: commands.InternalError,
		errors.New("other"): commands.InternalError,
	}
	for err, want := range cases {
		var failure *commands.Failure
		if !errors.As(mapContract(err, commands.InvalidStage, "from"), &failure) || failure.Code != want {
			t.Errorf("%v: %v", err, failure)
		}
	}
}
