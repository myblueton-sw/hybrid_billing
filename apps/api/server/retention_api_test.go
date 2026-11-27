package server

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

var (
	retA1     = retention.Actor{ID: SampleAdmin, Kind: retention.KindHuman}
	retA2     = retention.Actor{ID: SampleAdmin2, Kind: retention.KindHuman}
	retMember = retention.Actor{ID: SampleMember, Kind: retention.KindHuman}
	retAnn    = retention.Actor{ID: "test-ann", Kind: retention.KindHuman}
	retBob    = retention.Actor{ID: "test-bob", Kind: retention.KindHuman}
	retIngest = retention.Actor{ID: "test-ingest", Kind: retention.KindSystem}
	retWorker = retention.Actor{ID: "test-worker", Kind: retention.KindSystem}
)

func (f world) retentionRead(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/retention", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func retSection(got reply, name string) map[string]any {
	m, _ := got.body[name].(map[string]any)
	return m
}

func retRows(section map[string]any) []map[string]any {
	out := []map[string]any{}
	list, _ := section["rows"].([]any)
	for _, item := range list {
		out = append(out, item.(map[string]any))
	}
	return out
}

func retGroup(t testing.TB, got reply, class string) map[string]any {
	t.Helper()
	list, _ := got.body["expiry"].([]any)
	for _, item := range list {
		if g := item.(map[string]any); g["class"] == class {
			return g
		}
	}
	t.Fatalf("no class %s in %s", class, got.raw)
	return nil
}

func retArtifact(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	list, _ := got.body["expiry"].([]any)
	for _, item := range list {
		for _, a := range item.(map[string]any)["artifacts"].([]any) {
			if row := a.(map[string]any); row["id"] == id {
				return row
			}
		}
	}
	t.Fatalf("no artifact %s in %s", id, got.raw)
	return nil
}

func retFind(t testing.TB, rows []map[string]any, key, value string) map[string]any {
	t.Helper()
	for _, row := range rows {
		if fmt.Sprint(row[key]) == value {
			return row
		}
	}
	t.Fatalf("no row %s=%s", key, value)
	return nil
}

func retHold(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	return retFind(t, retRows(retSection(got, "holds")), "id", id)
}

func retRun(t testing.TB, got reply, label string) map[string]any {
	t.Helper()
	for _, row := range retRows(retSection(got, "deletion")) {
		if row["run"].(map[string]any)["label"] == label {
			return row
		}
	}
	t.Fatalf("no run %s in %s", label, got.raw)
	return nil
}

func retProposal(t testing.TB, got reply, number uint64) map[string]any {
	t.Helper()
	return retFind(t, retRows(retSection(got, "proposals")), "number", fmt.Sprint(number))
}

func retBlocked(row map[string]any) string {
	return fmt.Sprint(row["can_approve"], " ", row["approve_blocked"], " ", row["can_withdraw"], " ", row["withdraw_blocked"])
}

func retTarget(n uint64) string { return fmt.Sprintf("%s%d", retentionProposalTarget, n) }

func retJoin(values ...any) string {
	out := []string{}
	for _, v := range values {
		out = append(out, fmt.Sprint(v))
	}
	return strings.Join(out, " ")
}

func (f world) retStanding(id string) retention.Standing {
	f.t.Helper()
	v, err := f.s.retention.Standing(id)
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func (f world) retRunView(id string) retention.RunView {
	f.t.Helper()
	v, err := f.s.retention.Run(id)
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func (f world) retHoldView(id string) retention.HoldView {
	f.t.Helper()
	v, err := f.s.retention.Hold(id)
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func (f world) retRegister(id, class string, ageDays int, tenants ...string) {
	f.t.Helper()
	x := retention.Artifact{ID: id, Class: class, Tenants: tenants, Anchor: f.s.clock().Add(-time.Duration(ageDays) * 24 * time.Hour), Stores: []string{retentionDatabase}}
	if err := f.s.retention.Register(x, retIngest); err != nil {
		f.t.Fatal(err)
	}
	f.s.ret.artifacts = append(f.s.ret.artifacts, &retentionArtifact{id: id, class: class, tenants: tenants})
}

func (f world) retPlan(class, workspace string, ids ...string) string {
	f.t.Helper()
	p := retention.DeletionPlan{Class: class, Artifacts: ids, Chunk: len(ids), Lease: time.Hour, Reason: "test"}
	e, err := f.s.retentionPropose(retention.Change{Kind: retention.ChangePlan, Plan: p}, retAnn, workspace, "injected plan rationale")
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.s.retentionApprove(e, retBob, retention.Impact{}, "injected approval rationale"); err != nil {
		f.t.Fatal(err)
	}
	for n := uint64(1); ; n++ {
		id := retentionRunPrefix + fmt.Sprint(n)
		if n <= f.s.ret.plans {
			continue
		}
		if v, err := f.s.retention.Run(id); err != nil || !samePlan(v.Plan, p) {
			f.t.Fatalf("run %s %v", id, err)
		}
		f.s.ret.plans++
		f.s.ret.recordRun(id, workspace, ids)
		return id
	}
}

func (f world) retDelete(run string, chunk int, ids ...string) {
	f.t.Helper()
	c, err := f.s.retention.Claim(run, chunk, retWorker)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.s.retention.Report(run, chunk, c.Token, retentionDatabase, ids, nil, retWorker); err != nil {
		f.t.Fatal(err)
	}
}

func (f world) retBalanced(t *testing.T, got reply) {
	t.Helper()
	for _, item := range got.body["expiry"].([]any) {
		g := item.(map[string]any)
		sum := 0
		for _, key := range retentionStandings {
			n, ok := g["counts"].(map[string]any)[key]
			if !ok {
				t.Errorf("%s misses %s", g["class"], key)
			}
			sum += fmtInt(n)
		}
		if sum != fmtInt(g["total"]) || len(g["counts"].(map[string]any)) != 8 || g["population"] != retentionPopulation {
			t.Errorf("%s counts %v total %v", g["class"], g["counts"], g["total"])
		}
	}
	for _, row := range retRows(retSection(got, "deletion")) {
		sum := 0
		for _, n := range row["counts"].(map[string]any) {
			sum += fmtInt(n)
		}
		v := f.retRunView(row["run"].(map[string]any)["label"].(string))
		if sum != len(v.Plan.Artifacts) || len(row["counts"].(map[string]any)) != 9 || row["decision"] != notConnected {
			t.Errorf("run counts %v", row["counts"])
		}
		for _, item := range row["stores"].([]any) {
			s := item.(map[string]any)
			if fmtInt(s["target"]) != fmtInt(s["deleted"])+fmtInt(s["failed"])+fmtInt(s["excluded"])+fmtInt(s["pending"]) || fmtInt(s["pending"]) < 0 {
				t.Errorf("store %v", s)
			}
			if c := v.Stores[s["store"].(string)]; c.Target != fmtInt(s["target"]) || c.Deleted != fmtInt(s["deleted"]) || c.Failed != fmtInt(s["failed"]) ||
				c.Pending != fmtInt(s["excluded"])+fmtInt(s["pending"]) {
				t.Errorf("store %v module %+v", s, c)
			}
		}
	}
	for _, name := range []string{"holds", "deletion", "proposals"} {
		section := retSection(got, name)
		if _, ok := section["total"]; !ok || section["more"] == nil {
			t.Errorf("%s has no total or more", name)
		}
	}
	tomb := retSection(got, "decisions")["tombstones"].(map[string]any)
	if fmtInt(tomb["window"]) != retentionWindow || tomb["more"] == nil {
		t.Errorf("tombstones %v", tomb)
	}
}

func (f world) retAgrees(t *testing.T, got reply) {
	t.Helper()
	for _, item := range got.body["expiry"].([]any) {
		g := item.(map[string]any)
		class := g["class"].(string)
		eligible, held, referenced := []string{}, 0, 0
		for _, a := range g["artifacts"].([]any) {
			row := a.(map[string]any)
			id := row["id"].(string)
			v := f.retStanding(id)
			art, err := f.s.retention.Artifact(id)
			if err != nil {
				t.Fatal(err)
			}
			due := any(nil)
			if !v.DueAt.IsZero() {
				due = stamp(v.DueAt)
			}
			if row["access"] != v.Access || row["held"] != v.Held || row["damaged"] != v.Damaged || row["standing"] != standingOf(v) || row["anchor"] != stamp(art.Anchor) ||
				fmt.Sprint(row["due_at"]) != fmt.Sprint(due) || fmt.Sprint(row["stores"]) != fmt.Sprint(art.Stores) {
				t.Errorf("%s row %v standing %+v", id, row, v)
			}
			if _, has := row["decision"]; has != (v.Access != retention.AccessOK) || has && row["decision"] != notConnected {
				t.Errorf("%s decision %v", id, row["decision"])
			}
			switch row["standing"] {
			case standingEligible:
				eligible = append(eligible, id)
			case standingHeld:
				held++
			case standingReferenced:
				referenced++
			}
		}
		preview, err := f.s.retention.Preview(class, "", 1000)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(eligible, held, referenced) != fmt.Sprint(preview.Eligible, preview.Held, preview.Referenced) {
			t.Errorf("%s read %v %d %d preview %+v", class, eligible, held, referenced, preview)
		}
		policy := f.s.retention
		row := retFind(t, func() []map[string]any {
			out := []map[string]any{}
			for _, p := range got.body["policies"].([]any) {
				out = append(out, p.(map[string]any))
			}
			return out
		}(), "class", class)
		v, err := policy.Policy(class)
		if err != nil != (row["policy"] == nil) || err == nil && fmt.Sprint(row["policy"].(map[string]any)["period_days"], row["policy"].(map[string]any)["grace_days"],
			row["policy"].(map[string]any)["revision"]) != fmt.Sprint(v.Policy.Period, v.Policy.Grace, v.Number) {
			t.Errorf("%s policy %v %+v", class, row["policy"], v)
		}
	}
	for _, row := range retRows(retSection(got, "holds")) {
		v := f.retHoldView(row["id"].(string))
		outcomes, after := []retention.HoldOutcome{}, ""
		for {
			page, err := f.s.retention.HoldOutcomes(row["id"].(string), after, 1000)
			if err != nil {
				t.Fatal(err)
			}
			outcomes = append(outcomes, page...)
			if len(page) < 1000 {
				break
			}
			after = page[len(page)-1].Artifact
		}
		shown := []string{}
		for _, o := range outcomes {
			if f.s.ret.visibleArtifact(o.Artifact, SampleWorkspace) && len(shown) < retentionShown {
				shown = append(shown, fmt.Sprint("map[artifact:", o.Artifact, " outcome:", o.Outcome, " stores:", o.Stores, "]"))
			}
		}
		if row["released"] != v.Released || row["placer"].(map[string]any)["id"] != v.Placer || row["reason"] != v.Hold.Reason ||
			fmt.Sprint(row["outcomes"]) != "["+strings.Join(shown, " ")+"]" {
			t.Errorf("hold %v %+v", row, v)
		}
		if row["counts"] == nil {
			if row["counts_scope"] != countsOtherScope {
				t.Errorf("hold without counts %v", row)
			}
			continue
		}
		for _, key := range retentionHoldCounts {
			if fmtInt(row["counts"].(map[string]any)[key]) != v.Counts[key] || row["counts_scope"] != countsWorkspace {
				t.Errorf("hold counts %v %v", row["counts"], v.Counts)
			}
		}
	}
	for _, row := range retRows(retSection(got, "deletion")) {
		label := row["run"].(map[string]any)["label"].(string)
		v := f.retRunView(label)
		if row["state"] != v.State || fmtInt(row["next_chunk"]) != v.Next || fmtInt(row["damaged"]) != v.Damaged || row["run"].(map[string]any)["target"] != runTarget(label) {
			t.Errorf("run %v %+v", row, v)
		}
		for key, n := range v.Counts {
			if fmtInt(row["counts"].(map[string]any)[key]) != n {
				t.Errorf("run counts %v %v", row["counts"], v.Counts)
			}
		}
	}
	decisions := retSection(got, "decisions")
	if _, ok := decisions["watermark"]; ok {
		t.Errorf("the installation watermark is sent: %v", decisions["watermark"])
	}
	all := []retention.Tombstone{}
	for from := uint64(0); from < f.s.retention.Watermark(); from += 1000 {
		page, err := f.s.retention.Tombstones(from, int(min(f.s.retention.Watermark()-from, 1000)))
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, page...)
	}
	for _, row := range retRows(decisions["tombstones"].(map[string]any)) {
		if _, ok := row["sequence"]; ok {
			t.Errorf("a sequence is sent: %v", row)
		}
		var x retention.Tombstone
		for _, y := range all {
			if y.Artifact == row["artifact"] {
				x = y
			}
		}
		if row["artifact"] != x.Artifact || row["class"] != x.Class || row["at"] != stamp(x.At) || row["decision"] != notConnected {
			t.Errorf("tombstone %v %+v", row, x)
		}
	}
	for _, row := range retRows(retSection(got, "proposals")) {
		v, err := f.s.retention.Proposal(uint64(fmtInt(row["number"])))
		switch {
		case err == nil && (row["state"] != v.State || row["kind"] != v.Change.Kind):
			t.Errorf("proposal %v %+v", row, v)
		case err != nil && row["state"] != retention.ProposalWithdrawn && row["state"] != retentionRetired:
			t.Errorf("proposal %v %v", row, err)
		}
	}
}

func TestRetentionReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withoutThirdAdmin()
	root := f.as(AdminLogin)
	got := f.retentionRead(root)
	if got.status != 200 || got.body["state"] != "configured" || got.body["now"] != stamp(start) || got.body["revision"] != f.s.Revision() {
		t.Fatalf("read %s", got.raw)
	}
	f.retBalanced(t, got)
	f.retAgrees(t, got)
	classes := []string{}
	for _, p := range got.body["policies"].([]any) {
		row := p.(map[string]any)
		classes = append(classes, fmt.Sprint(row["class"], ":", row["policy"] != nil, ":", row["can_propose"], row["propose_blocked"], ":", row["pending"]))
		if row["origin"] != "artifact_anchor" || row["scope"] != notConnected || row["archive_rule"] != notConnected || row["target"] != retentionClassTarget+row["class"].(string) {
			t.Errorf("policy row %v", row)
		}
	}
	if s := strings.Join(classes, " "); s != "claim:false:true:<nil> conversation:false:true:<nil> final:true:true:<nil> identity:false:true:<nil> projection:false:true:<nil> "+
		"source:true:falsealready_proposed:7 temporary:true:true:<nil>" {
		t.Errorf("policies %s", s)
	}
	source := got.body["policies"].([]any)[5].(map[string]any)["policy"]
	if fmt.Sprint(source) != "map[approved_at:2026-11-16T09:00:00Z approver:map[id:sample-admin-2 login:admin2@example.invalid] grace_days:30 period_days:365 "+
		"proposer:map[id:sample-admin login:admin@example.invalid] revision:1]" {
		t.Errorf("source policy %v", source)
	}
	counts := []string{}
	for _, class := range retention.Classes {
		g := retGroup(t, got, class)
		counts = append(counts, retJoin(class, g["counts"], g["total"], g["more"], g["usage"], g["growth"], g["bytes"]))
	}
	if s := strings.Join(counts, "|"); s != "claim map[deleted:0 eligible:0 guarding:1 held:0 in_run:0 no_policy:0 not_due:0 referenced:0] 1 false not_connected not_connected not_connected|"+
		"conversation map[deleted:0 eligible:0 guarding:0 held:0 in_run:0 no_policy:1 not_due:0 referenced:0] 1 false not_connected not_connected not_connected|"+
		"final map[deleted:0 eligible:0 guarding:0 held:0 in_run:0 no_policy:0 not_due:1 referenced:0] 1 false not_connected not_connected not_connected|"+
		"identity map[deleted:0 eligible:0 guarding:0 held:0 in_run:0 no_policy:0 not_due:0 referenced:0] 0 false not_connected not_connected not_connected|"+
		"projection map[deleted:0 eligible:0 guarding:0 held:0 in_run:0 no_policy:1 not_due:0 referenced:0] 1 false not_connected not_connected not_connected|"+
		"source map[deleted:1 eligible:0 guarding:0 held:1 in_run:1 no_policy:0 not_due:3 referenced:1] 7 false not_connected not_connected not_connected|"+
		"temporary map[deleted:1 eligible:0 guarding:0 held:0 in_run:2 no_policy:0 not_due:0 referenced:0] 3 false not_connected not_connected not_connected" {
		t.Errorf("expiry %s", s)
	}
	row := func(id string) string {
		a := retArtifact(t, got, id)
		return fmt.Sprint(a["access"], " ", a["standing"], " ", a["held"], " ", a["damaged"], " ", a["run"], " ", a["other_scope"], " ", a["shared"], " ", a["decision"], " ",
			a["holdable"], " ", a["hold_target"])
	}
	for id, want := range map[string]string{
		"sample-source-1":    "deleted deleted false true <nil> false false not_connected true retention-artifact:sample-source-1",
		"sample-source-2":    "blocked in_run false true map[label:run/1 target:retention-run-1] false false not_connected true retention-artifact:sample-source-2",
		"sample-source-3":    "ok referenced false false <nil> false false <nil> true retention-artifact:sample-source-3",
		"sample-source-4":    "ok held true false <nil> false false <nil> true retention-artifact:sample-source-4",
		"sample-temporary-1": "deleted deleted true true <nil> false false not_connected true retention-artifact:sample-temporary-1",
		"sample-temporary-2": "blocked in_run true false map[label:run/2 target:retention-run-2] false false not_connected true retention-artifact:sample-temporary-2",
		"sample-temporary-3": "blocked in_run true false map[label:run/2 target:retention-run-2] false false not_connected true retention-artifact:sample-temporary-3",
		"sample-claim-1":     "ok guarding false false <nil> false false <nil> true retention-artifact:sample-claim-1",
	} {
		if got := row(id); got != want {
			t.Errorf("%s: %s", id, got)
		}
	}
	if a := retArtifact(t, got, "sample-source-5"); fmt.Sprint(a["anchor"], " ", a["due_at"], " ", a["stores"]) != "2026-01-20T09:00:00Z 2027-02-19T09:00:00Z [sample-database]" {
		t.Errorf("source 5 %v", a)
	}
	if a := retArtifact(t, got, "sample-claim-1"); a["due_at"] != nil {
		t.Errorf("claim due %v", a["due_at"])
	}
	holds := retSection(got, "holds")
	if retJoin(holds["total"], holds["more"], holds["tenant_target"], holds["references"]) != "2 false retention-tenant:sample-workspace "+
		"map[more:false next: population:registered_sample_artifacts rows:[map[class:source id:sample-source-3]] total:1]" {
		t.Errorf("holds %v", holds)
	}
	temporary := retHold(t, got, retentionTemporary)
	if s := fmt.Sprint(temporary); s != "map[can_propose_release:false counts:map[not_preserved:1 outcome_pending:0 partially_deleted:0] counts_scope:workspace "+
		"id:sample-hold-temporary number:2 "+
		"outcomes:[map[artifact:sample-temporary-1 outcome:not_preserved stores:map[sample-object-store:deleted]]] outcomes_more:false outcomes_next: "+
		"owner:map[id:sample-member login:member@example.invalid] placed_at:2026-11-16T09:00:00Z placer:map[id:sample-member login:member@example.invalid] "+
		"rationale:Sample data reason:Sample export review hold release:6 release_blocked:already_proposed released:false "+
		"scope:map[artifacts:[] classes:[temporary] workspace:true] target:retention-hold:sample-hold-temporary]" {
		t.Errorf("temporary hold %s", s)
	}
	dispute := retHold(t, got, retentionDispute)
	if s := retJoin(dispute["scope"], dispute["counts"], dispute["release"], dispute["release_blocked"], dispute["owner"]); s != "map[artifacts:[sample-source-4] "+
		"classes:[] workspace:false] map[not_preserved:0 outcome_pending:0 partially_deleted:0] <nil> no_eligible_approver map[id:sample-admin-2 login:admin2@example.invalid]" {
		t.Errorf("dispute hold %s", s)
	}
	one, two := retRun(t, got, "run/1"), retRun(t, got, "run/2")
	if s := fmt.Sprint(one["stores"]); s != "[map[deleted:2 excluded:2 failed:0 pending:0 store:sample-database target:4] "+
		"map[deleted:1 excluded:0 failed:1 pending:0 store:sample-object-store target:2]]" {
		t.Errorf("run 1 stores %s", s)
	}
	if s := retJoin(one["state"], one["can_stop"], one["stop_blocked"], one["chunks"], one["chunk_size"], one["lease_seconds"], one["revision"], one["reason"],
		one["stopper"], one["artifacts"]); s != "done false finished 2 2 3600 1 Sample expiry deletion <nil> map[more:false next: rows:[map[id:sample-source-1 outcome:deleted] "+
		"map[id:sample-source-2 outcome:failed] map[id:sample-source-3 outcome:referenced] map[id:sample-source-4 outcome:held]] total:4]" {
		t.Errorf("run 1 %s", s)
	}
	if s := retJoin(two["state"], two["can_stop"], two["stop_blocked"], two["next_chunk"], two["stores"]); s != "running true  1 [map[deleted:1 excluded:0 failed:0 pending:2 "+
		"store:sample-object-store target:3]]" {
		t.Errorf("run 2 %s", s)
	}
	deletion := retSection(got, "deletion")
	if fmt.Sprint(deletion["worker"], deletion["retry"], deletion["backup_expiry"], deletion["propose_plan"], retSection(got, "restore")) !=
		"not_connectednot_connectednot_connectednot_connectedmap[state:not_connected]" {
		t.Errorf("not connected %v", deletion)
	}
	decisions := retSection(got, "decisions")
	if s := fmt.Sprint(decisions); s != "map[durable_commit:not_connected epoch:not_connected ledger:in_memory restore_check:not_connected "+
		"tombstones:map[more:false next: rows:[map[artifact:sample-temporary-1 at:2026-11-16T09:00:00Z class:temporary decision:not_connected other_scope:false "+
		"run:map[label:run/2 target:retention-run-2]] map[artifact:sample-source-1 at:2026-11-16T09:00:00Z class:source decision:not_connected "+
		"other_scope:false run:map[label:run/1 target:retention-run-1]]] total:2 window:1000]]" {
		t.Errorf("decisions %s", s)
	}
	numbers := []string{}
	for _, row := range retRows(retSection(got, "proposals")) {
		numbers = append(numbers, fmt.Sprint(row["number"], ":", row["kind"], ":", row["state"], ":", retBlocked(row)))
	}
	if s := strings.Join(numbers, ","); s != "7:policy:pending:true  false not_proposer,6:release:pending:false own_proposal true ,5:plan:applied:false not_pending false not_pending,"+
		"4:plan:applied:false not_pending false not_pending,3:policy:applied:false not_pending false not_pending,2:policy:applied:false not_pending false not_pending,"+
		"1:policy:applied:false not_pending false not_pending" {
		t.Errorf("proposals %s", s)
	}
	short := retProposal(t, got, 7)
	impact, err := f.s.retention.Impact(retention.Policy{Class: retention.ClassSource, Period: 180, Grace: 30})
	if err != nil {
		t.Fatal(err)
	}
	if s := retJoin(short["impact_required"], short["impact_now"], short["summary"]); s != "true map[as_of:2026-11-16T09:00:00Z count:2 digest:"+impact.Impact.Digest+
		" workspace_count:2] map[plan:<nil> policy:map[class:source grace_days:30 impact_at_proposal:map[as_of:2026-11-16T09:00:00Z count:2 digest:"+impact.Impact.Digest+
		"] period_days:180] release:<nil>]" {
		t.Errorf("shortening %s", s)
	}
	if s := retJoin(retProposal(t, got, 5)["summary"], retProposal(t, got, 6)["summary"], retProposal(t, got, 6)["impact_now"]); s != "map[plan:map[artifact_count:3 "+
		"chunk_size:1 class:temporary lease_seconds:3600 reason:Sample temporary file expiry] policy:<nil> release:<nil>] map[plan:<nil> policy:<nil> "+
		"release:map[hold:sample-hold-temporary reason:Sample review finished]] <nil>" {
		t.Errorf("summaries %s", s)
	}
	applied := retProposal(t, got, 1)
	if s := retJoin(applied["approver"], applied["closed_by"], applied["closing_rationale"], applied["rationale"], applied["impact_required"], applied["target"],
		applied["summary"].(map[string]any)["policy"].(map[string]any)["impact_at_proposal"]); s != "map[id:sample-admin-2 login:admin2@example.invalid] "+
		"map[id:sample-admin-2 login:admin2@example.invalid] Sample data Sample data false retention-proposal-1 <nil>" {
		t.Errorf("applied %s", s)
	}
	second := f.retentionRead(f.as(Admin2Login))
	if s := retBlocked(retProposal(t, second, 7)) + "|" + retBlocked(retProposal(t, second, 6)); s != "false own_proposal true |true  false not_proposer" {
		t.Errorf("second view %s", s)
	}
}

func TestRetentionReadWithoutSeed(t *testing.T) {
	none := setup(t, SeedNone).ready()
	tok := none.links()[""]
	if got := none.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, none.pre()); got.status != 204 {
		t.Fatalf("activate %+v", got)
	}
	cookie, csrf, err := none.s.openFor(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	a := actor{cookie, csrf}
	got := none.retentionRead(a)
	if got.status != 200 || got.body["state"] != "configured" {
		t.Fatalf("empty %s", got.raw)
	}
	for _, p := range got.body["policies"].([]any) {
		if row := p.(map[string]any); row["policy"] != nil || row["can_propose"] != true {
			t.Errorf("policy %v", row)
		}
	}
	for _, class := range retention.Classes {
		if g := retGroup(t, got, class); fmtInt(g["total"]) != 0 || len(g["artifacts"].([]any)) != 0 || g["usage"] != notConnected {
			t.Errorf("group %v", g)
		}
	}
	for _, name := range []string{"holds", "deletion", "proposals"} {
		if s := retSection(got, name); fmtInt(s["total"]) != 0 || len(retRows(s)) != 0 {
			t.Errorf("%s %v", name, s)
		}
	}
	if s := retJoin(retSection(got, "decisions")["tombstones"], retSection(got, "restore")); s !=
		"map[more:false next: rows:[] total:0 window:1000] map[state:not_connected]" {
		t.Errorf("decisions %s", s)
	}
	for _, c := range []struct{ kind, target string }{{retentionApproveKind, retTarget(1)}, {retentionStopKind, "retention-run-1"},
		{retentionReleaseKind, retentionHoldTarget + retentionDispute}, {retentionHoldKind, retentionArtifactTarget + "sample-source-1"}} {
		none.code(t, c.kind, none.runWith(a, none.revision(a), retentionCommand+c.kind, c.target, retFields(c.kind, map[string]string{"reason": "x", "owner": "admin"})), 403,
			"permission_denied")
	}
	proposed := none.runWith(a, none.revision(a), "retention.propose_policy", retentionClassTarget+retention.ClassClaim, map[string]string{"period": "30", "grace": "0"})
	if proposed.status != 200 || proposed.body["business_state"] != "pending" {
		t.Errorf("first policy %s", proposed.raw)
	}
	if p := none.retentionRead(a).body["policies"].([]any)[0].(map[string]any); fmt.Sprint(p["pending"], p["propose_blocked"]) != "1already_proposed" {
		t.Errorf("pending %v", p)
	}
}

func TestRetentionAccess(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, root := f.as(MemberLogin), f.as(AdminLogin)
	f.code(t, "member read", f.retentionRead(member), 403, "permission_denied")
	f.code(t, "anonymous", f.retentionRead(actor{}), 401, "unauthenticated")
	targets := map[string]string{retentionApproveKind: retTarget(7), retentionWithdrawKind: retTarget(7), retentionPolicyKind: retentionClassTarget + retention.ClassClaim,
		retentionHoldKind: retentionArtifactTarget + "sample-source-5", retentionReleaseKind: retentionHoldTarget + retentionDispute, retentionStopKind: "retention-run-2"}
	fields := map[string]string{"period": "30", "grace": "0", "owner": SampleMember, "reason": "x"}
	for kind := range targets {
		if fields := retFields(kind, fields); len(fields) == 0 && kind != retentionApproveKind && kind != retentionWithdrawKind && kind != retentionStopKind {
			t.Errorf("%s has no fields", kind)
		}
	}
	before := f.s.Revision()
	for kind, target := range targets {
		f.code(t, "member "+kind, f.runWith(member, f.revision(root), retentionCommand+kind, target, retFields(kind, fields)), 403, "permission_denied")
		h := retentionHandler{s: f.s, kind: kind}
		if _, res := h.Resolve("other-ws", target); res != commands.NotFound {
			t.Errorf("%s resolved in another workspace", kind)
		}
		if _, res := h.Resolve(SampleWorkspace, target); res != commands.Found {
			t.Errorf("%s did not resolve %s", kind, target)
		}
	}
	if _, res := (retentionHandler{s: f.s, kind: retentionHoldKind}).Resolve(SampleWorkspace, retentionTenantTarget+SampleWorkspace); res != commands.Found {
		t.Error("tenant target")
	}
	f.s.bootWorkspace = "elsewhere"
	if got := f.retentionRead(root); got.status != 200 || got.body["state"] != "not_configured" || got.body["policies"] != nil {
		t.Errorf("other workspace %s", got.raw)
	}
	for kind, target := range targets {
		f.code(t, "other workspace "+kind, f.runWith(root, f.s.Revision(), retentionCommand+kind, target, retFields(kind, fields)), 403, "permission_denied")
	}
	f.s.bootWorkspace = SampleWorkspace
	for _, c := range []struct{ kind, target string }{
		{retentionApproveKind, "retention-proposal-07"}, {retentionApproveKind, "retention-proposal-x"}, {retentionApproveKind, "retention-proposal-99"},
		{retentionApproveKind, "7"}, {retentionWithdrawKind, "retention-proposal-"}, {retentionApproveKind, "retention-proposal-18446744073709551616"},
		{retentionApproveKind, retentionClassTarget + "source"}, {retentionPolicyKind, retentionClassTarget + "unknown"}, {retentionPolicyKind, "source"},
		{retentionPolicyKind, retentionClassTarget}, {retentionHoldKind, retentionArtifactTarget + "sample-absent"}, {retentionHoldKind, retentionTenantTarget + "other-ws"},
		{retentionHoldKind, "sample-source-5"}, {retentionHoldKind, retentionHoldTarget + retentionDispute}, {retentionReleaseKind, retentionHoldTarget + "sample-absent"},
		{retentionReleaseKind, retentionDispute}, {retentionStopKind, "retention-run-3"}, {retentionStopKind, "retention-run-01"}, {retentionStopKind, "retention-run-1x"},
		{retentionStopKind, "retention-run-"}, {retentionStopKind, "retention-run-0"},
	} {
		f.code(t, c.kind+" "+c.target, f.run(root, retentionCommand+c.kind, c.target, retFields(c.kind, fields)), 403, "permission_denied")
	}
	if f.s.Revision() != before {
		t.Errorf("refused commands moved the revision %s %s", before, f.s.Revision())
	}
	long := strings.Repeat("r", maxRationale+1)
	f.unchanged(t, "long reason", f.admin(root, "retention.approve", retTarget(7), long, map[string]string{}), "invalid_request")
	f.unchanged(t, "no reason", f.admin(root, "retention.withdraw", retTarget(7), "", map[string]string{}), "invalid_request")
	f.unchanged(t, "no period", f.run(root, "retention.propose_policy", retentionClassTarget+retention.ClassClaim, map[string]string{"grace": "0"}), "invalid_request")
}

func retFields(kind string, all map[string]string) map[string]string {
	names := map[string][]string{retentionPolicyKind: {"period", "grace"}, retentionHoldKind: {"owner", "reason"}, retentionReleaseKind: {"reason"}}[kind]
	out := map[string]string{}
	for _, name := range names {
		out[name] = all[name]
	}
	return out
}

func TestRetentionReadBlockers(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withoutThirdAdmin()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	plan, err := f.s.retentionPropose(retention.Change{Kind: retention.ChangePlan, Plan: retention.DeletionPlan{Class: retention.ClassSource,
		Artifacts: []string{"sample-source-5"}, Chunk: 1, Lease: time.Hour, Reason: "test"}}, retA1, SampleWorkspace, "test")
	if err != nil {
		t.Fatal(err)
	}
	read, other := f.retentionRead(root), f.retentionRead(second)
	if s := retBlocked(retProposal(t, read, plan.view.Number)) + "|" + retBlocked(retProposal(t, other, plan.view.Number)); s !=
		"false not_console_kind true |false not_console_kind false not_proposer" {
		t.Errorf("plan blockers %s", s)
	}
	f.s.dir.holders[SampleMember] = start.Add(time.Hour)
	if s := retHold(t, f.retentionRead(root), retentionDispute)["release_blocked"]; s != retention.ReasonPlacer {
		t.Errorf("placer %v", s)
	}
	if s := retHold(t, f.retentionRead(second), retentionDispute); s["can_propose_release"] != true || s["release_blocked"] != "" {
		t.Errorf("eligible %v", s)
	}
	delete(f.s.dir.holders, SampleMember)
	f.ok(t, "withdraw release", f.run(root, "retention.withdraw", retTarget(6), nil), "withdrawn", "")
	if s := retHold(t, f.retentionRead(second), retentionTemporary); s["can_propose_release"] != true || s["release_blocked"] != "" || s["release"] != nil {
		t.Errorf("after withdraw %v", s)
	}
	if got := f.run(root, "identity.suspend_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	read = f.retentionRead(root)
	if s := retHold(t, read, retentionTemporary)["release_blocked"]; s != noEligibleApprover {
		t.Errorf("suspended %v", s)
	}
	if s := retBlocked(retProposal(t, read, 6)); s != "false not_pending false not_pending" {
		t.Errorf("withdrawn %s", s)
	}
	if s := retBlocked(retProposal(t, read, 7)); s != "true  false not_proposer" {
		t.Errorf("shortening %s", s)
	}
	if got := f.run(root, "identity.reinstate_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatalf("reinstate %s", got.raw)
	}
	second = f.as(Admin2Login)
	held := f.run(root, "retention.place_hold", retentionArtifactTarget+"sample-source-6", map[string]string{"owner": SampleAdmin, "reason": "Admin hold"})
	f.ok(t, "place", held, "held", "")
	id := held.body["fields"].(map[string]any)["hold"].(string)
	if s := retHold(t, f.retentionRead(second), id)["release_blocked"]; s != noEligibleApprover {
		t.Errorf("admin hold %v", s)
	}
	proposed := f.run(root, "retention.propose_release", retentionHoldTarget+retentionTemporary, map[string]string{"reason": "Done"})
	f.ok(t, "propose release", proposed, "pending", "")
	number := proposed.body["fields"].(map[string]any)["proposal"].(string)
	f.ok(t, "approve release", f.run(second, "retention.approve", retentionProposalTarget+number, nil), "applied", "")
	read = f.retentionRead(root)
	if s := fmt.Sprint(retHold(t, read, retentionTemporary)["released"], retHold(t, read, retentionTemporary)["release_blocked"]); s != "truereleased" {
		t.Errorf("released %s", s)
	}
	if s := fmt.Sprint(retArtifact(t, read, "sample-temporary-2")["held"]); s != "false" {
		t.Errorf("held after release %s", s)
	}
	f.s.ret.entry(7).view.State = retentionRetired
	if s := retBlocked(retProposal(t, f.retentionRead(root), 7)); s != "false retired false retired" {
		t.Errorf("retired %s", s)
	}
	f.s.ret.entry(7).view.State = retention.ProposalPending
	f.ok(t, "stop", f.run(root, "retention.stop_run", "retention-run-2", nil), "done", "")
	if s := retRun(t, f.retentionRead(root), "run/2"); s["stop_blocked"] != runFinished || s["can_stop"] != false {
		t.Errorf("stopped run %v", s)
	}
	if stopBlocked(retention.RunStopped) != retention.RunStopped || stopBlocked(retention.RunComplete) != runFinished {
		t.Error("stop blockers")
	}
	for _, size := range []int{109, 110} {
		id := strings.Repeat("a", size)
		f.retRegister(id, retention.ClassIdentity, 1, SampleWorkspace)
		a := retArtifact(t, f.retentionRead(root), id)
		if fmt.Sprint(a["holdable"], a["hold_blocked"], a["hold_target"] != "") != map[int]string{109: "truetrue", 110: "falsetarget_too_longfalse"}[size] {
			t.Errorf("%d: %v", size, a)
		}
	}
	f.code(t, "too long target", f.run(root, "retention.place_hold", retentionArtifactTarget+strings.Repeat("a", 110), map[string]string{"owner": SampleAdmin, "reason": "x"}), 422,
		"invalid_request")
	f.ok(t, "longest target", f.run(root, "retention.place_hold", retentionArtifactTarget+strings.Repeat("a", 109), map[string]string{"owner": SampleAdmin, "reason": "x"}),
		"held", "")
}

func TestRetentionRosterAndClock(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.fault = func(step string) bool { return step == "roster" }
	f.code(t, "roster", f.retentionRead(root), 503, "dependency_unavailable")
	f.s.fault = nil
	before := f.retentionRead(root)
	revision := f.s.Revision()
	f.advance(2 * time.Hour)
	after := f.retentionRead(f.as(AdminLogin))
	now := retProposal(t, after, 7)["impact_now"].(map[string]any)
	if now["as_of"] != "2026-11-16T11:00:00Z" || retProposal(t, before, 7)["impact_now"].(map[string]any)["as_of"] != "2026-11-16T09:00:00Z" ||
		f.s.Revision() != revision || after.body["revision"] != before.body["revision"] {
		t.Errorf("clock %v %s", now, f.s.Revision())
	}
	snapshot := func() string {
		out := ""
		for n := uint64(1); n <= 7; n++ {
			v, err := f.s.retention.Proposal(n)
			out += fmt.Sprint(v, err)
		}
		for _, id := range []string{retentionDispute, retentionTemporary} {
			out += fmt.Sprint(f.retHoldView(id))
		}
		for _, id := range []string{"run/1", "run/2"} {
			out += fmt.Sprint(f.retRunView(id))
		}
		return out
	}
	unchanged := snapshot()
	for _, login := range []string{AdminLogin, Admin2Login} {
		if got := f.retentionRead(f.as(login)); got.status != 200 {
			t.Fatal(got.raw)
		}
	}
	if snapshot() != unchanged {
		t.Error("the read changed the module")
	}
}

func TestRetentionOtherTenantIsHidden(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	before := f.retentionRead(root)
	f.retRegister("other-only", retention.ClassSource, 400, "other-tenant")
	if err := f.s.retention.Refer(retention.Reference{From: "other-only", To: "sample-source-5", Kind: retention.Requires}, retIngest); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "other-hold", Owner: "other-owner", Reason: "other", Artifacts: []string{"other-only"}}, retAnn, "other-tenant",
		"other"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retentionPropose(retention.Change{Kind: retention.ChangeRelease, Release: retention.Release{Hold: "other-hold", Reason: "x"}}, retBob, "other-tenant", "other"); err != nil {
		t.Fatal(err)
	}
	after := f.retentionRead(root)
	for _, section := range []string{"deletion", "decisions", "proposals", "policies"} {
		if fmt.Sprint(after.body[section]) != fmt.Sprint(before.body[section]) {
			t.Errorf("%s changed", section)
		}
	}
	if fmt.Sprint(retRows(retSection(after, "holds"))) != fmt.Sprint(retRows(retSection(before, "holds"))) {
		t.Error("holds changed")
	}
	if s := fmt.Sprint(retArtifact(t, after, "sample-source-5")["standing"], retGroup(t, after, retention.ClassSource)["counts"],
		retSection(after, "holds")["references"].(map[string]any)["total"]); s != "not_duemap[deleted:1 eligible:0 guarding:0 held:1 in_run:1 no_policy:0 not_due:3 referenced:1] 2" {
		t.Errorf("reference %s", s)
	}
	f.retRegister("shared-a", retention.ClassSource, 400, "other-tenant", SampleWorkspace)
	f.retRegister("shared-b", retention.ClassSource, 400, "other-tenant", SampleWorkspace)
	run := f.retPlan(retention.ClassSource, "other-tenant", "shared-a", "shared-b")
	read := f.retentionRead(root)
	shared := retArtifact(t, read, "shared-a")
	if s := fmt.Sprint(shared["shared"], shared["run"], shared["other_scope"], shared["standing"], shared["access"], shared["decision"]); s !=
		"true <nil> truein_runblockednot_connected" {
		t.Errorf("owned %s", s)
	}
	f.retDelete(run, 0, "shared-a")
	read = f.retentionRead(root)
	tomb := retRows(retSection(read, "decisions")["tombstones"].(map[string]any))[0]
	if s := fmt.Sprint(tomb["artifact"], tomb["run"], tomb["other_scope"], tomb["decision"]); s != "shared-a<nil> truenot_connected" {
		t.Errorf("tombstone %s", s)
	}
	if s := fmt.Sprint(retArtifact(t, read, "shared-a")["standing"], retArtifact(t, read, "shared-b")["other_scope"]); s != "deletedtrue" {
		t.Errorf("after deletion %s", s)
	}
	for _, row := range retRows(retSection(read, "proposals")) {
		if row["kind"] == retention.ChangePlan && fmtInt(row["number"]) > 7 {
			t.Errorf("another tenant's plan is shown %v", row)
		}
	}
	if f.s.ret.visibleRun(run, SampleWorkspace) || fmtInt(retSection(read, "deletion")["total"]) != 2 {
		t.Error("another tenant's run is stored or shown")
	}
	for _, word := range []string{"other-tenant", "other-only", "other-hold", "other-owner", "\"" + run + "\"", runTarget(run), "test-ann", "test-bob",
		"injected plan rationale", "injected approval rationale"} {
		if strings.Contains(read.raw, word) {
			t.Errorf("read carries %s", word)
		}
	}
	for _, c := range []struct{ kind, target string }{{retentionHoldKind, retentionArtifactTarget + "other-only"}, {retentionHoldKind, retentionTenantTarget + "other-tenant"},
		{retentionReleaseKind, retentionHoldTarget + "other-hold"}, {retentionStopKind, runTarget(run)}, {retentionWithdrawKind, retTarget(uint64(len(f.s.ret.entries) + 1))}} {
		f.code(t, c.kind+" "+c.target, f.run(root, retentionCommand+c.kind, c.target, retFields(c.kind, map[string]string{"owner": SampleAdmin, "reason": "x"})), 403,
			"permission_denied")
	}
	for _, e := range f.s.ret.entries {
		if !f.s.ret.visibleEntry(e, SampleWorkspace) {
			f.code(t, "hidden proposal", f.run(root, "retention.approve", retTarget(e.view.Number), nil), 403, "permission_denied")
		}
	}
	ids := []string{}
	for _, row := range retRows(retSection(read, "holds")) {
		ids = append(ids, row["id"].(string))
	}
	sort.Strings(ids)
	if strings.Join(ids, " ") != "sample-hold-dispute sample-hold-temporary" {
		t.Errorf("holds %v", ids)
	}
}
