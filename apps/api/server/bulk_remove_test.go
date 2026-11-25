package server

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
)

func (f world) remove(a actor, draft string, users ...string) reply {
	f.t.Helper()
	return f.run(a, "bulk_revocation.remove_targets", draft, map[string]string{"users": strings.Join(users, "\n")})
}

func removal(got reply) string {
	return field(got, "removed") + " " + field(got, "already") + " " + field(got, "ignored") + " " + field(got, "chosen")
}

func TestRemoveTargetsLeavesChosenPeopleOutOfAFilterList(t *testing.T) {
	f, p, root := swept(filteredWorld(t))
	fixed := f.match(root, "membership=active&role=member")
	draft := field(fixed, "draft")
	before := f.draftUsers(SampleAdmin, draft)
	for _, id := range []string{"u-ann", "u-cat", "u-hal"} {
		if !slices.Contains(before, id) {
			t.Fatalf("%s is not in the fixed list %v", id, before)
		}
	}
	got := f.remove(root, draft, "u-ann", "u-nobody", SampleAdmin, "u-ann", "u-cat", "u-bob")
	if got.status != 200 || removal(got) != "2 1 3 2" || field(got, "draft") != draft || got.body["business_state"] != "draft" {
		t.Fatalf("removal %s", got.raw)
	}
	if again := f.remove(root, draft, "u-cat"); removal(again) != "0 1 0 2" {
		t.Fatalf("a second removal of the same person %s", again.raw)
	}
	if users := f.draftUsers(SampleAdmin, draft); len(users) != len(before) {
		t.Fatalf("the list changed size %v", users)
	}
	listing := f.bulkRead(root, "")
	jobs, _ := listing.body["jobs"].([]any)
	if len(jobs) != 1 || listing.body["open_draft"] != true {
		t.Fatalf("the listing %s", listing.raw)
	}
	open := jobs[0].(map[string]any)
	if whole(open["chosen"]) != 2 || whole(open["matched"]) != len(before) || open["id"] != draft {
		t.Fatalf("the open draft in the listing %s", listing.raw)
	}
	p.suspendMembership("u-cat")
	p.refresh()
	prepared := f.prepare(root, draft, "suspend_membership")
	if prepared.status != 200 {
		t.Fatalf("prepare %s", prepared.raw)
	}
	id := field(prepared, "bulk")
	reasons := f.reasons(root, id)
	if reasons["u-ann"] != bulkrevoke.ExcludedChosen || reasons["u-cat"] != bulkrevoke.ExcludedNoLongerMatches || reasons["u-hal"] != "target" || len(reasons) != len(before) {
		t.Fatalf("reasons %v", reasons)
	}
	detail := f.bulkRead(root, "/"+id)
	exclusions := detail.body["exclusions"].(map[string]any)
	job := detail.body["job"].(map[string]any)
	counts := job["counts"].(map[string]any)
	sum := 0
	for _, n := range exclusions {
		sum += whole(n)
	}
	if whole(exclusions["chosen"]) != 1 || whole(exclusions["no_longer_matches"]) != 1 || sum != whole(counts["excluded"]) || whole(counts["targets"])+whole(counts["excluded"]) != len(before) ||
		whole(job["matched"]) != len(before) || whole(job["chosen"]) != 2 {
		t.Fatalf("detail %s", detail.raw)
	}
	if got := f.remove(root, id, "u-hal"); code(got) != "invalid_stage" {
		t.Errorf("a prepared list %s", got.raw)
	}
	if got := f.remove(root, draft, "u-hal"); got.status == 200 {
		t.Errorf("the draft identifier of a prepared list %s", got.raw)
	}
	done := f.execute(root, id, field(prepared, "digest"))
	if done.status != 200 && done.status != 202 {
		t.Fatalf("execute %s", done.raw)
	}
	items := f.bulkRead(root, "/"+id+"/items?outcome=excluded&limit=100")
	found := map[string]string{}
	for _, row := range items.body["items"].([]any) {
		x := row.(map[string]any)
		found[x["user_id"].(string)] = x["reason"].(string)
	}
	if found["u-ann"] != bulkrevoke.ExcludedChosen || found["u-cat"] != bulkrevoke.ExcludedNoLongerMatches {
		t.Errorf("excluded items %v", found)
	}
	all := f.bulkRead(root, "/"+id+"/items?limit=100")
	outcome := map[string]string{}
	for _, row := range all.body["items"].([]any) {
		x := row.(map[string]any)
		outcome[x["user_id"].(string)] = x["outcome"].(string)
	}
	if outcome["u-ann"] != string(bulkrevoke.OutcomeExcluded) || outcome["u-hal"] == string(bulkrevoke.OutcomeExcluded) || outcome["u-hal"] == "" {
		t.Errorf("outcomes %v", outcome)
	}
}

func TestRemoveTargetsRefusals(t *testing.T) {
	f, _, root := swept(filteredWorld(t))
	second := f.as(Admin2Login)
	fixed := f.match(root, "membership=active&role=member")
	draft := field(fixed, "draft")
	for name, c := range map[string]struct {
		who    actor
		target string
		users  string
		want   string
	}{
		"no identifiers":         {root, draft, "", "invalid_request"},
		"a malformed identifier": {root, draft, "u-ann\na b", "invalid_request"},
		"another person's draft": {second, draft, "u-ann", ""},
		"a draft that never was": {root, "bulk-draft-999", "u-ann", ""},
		"the staging target":     {root, bulkTarget, "u-ann", ""},
	} {
		got := f.run(c.who, "bulk_revocation.remove_targets", c.target, map[string]string{"users": c.users})
		if got.status == 200 || (c.want != "" && code(got) != c.want) {
			t.Errorf("%s: %d %s", name, got.status, got.raw)
		}
	}
	foreign, unknown := f.remove(second, draft, "u-ann"), f.remove(root, "bulk-draft-999", "u-ann")
	if code(foreign) == "" || code(foreign) != code(unknown) || foreign.status != unknown.status {
		t.Errorf("another person's draft answers %d %s, a draft that never was %d %s", foreign.status, code(foreign), unknown.status, code(unknown))
	}
	if chosen, err := f.s.bulk.DraftChosen(SampleWorkspace, SampleAdmin, draft); err != nil || len(chosen) != 0 {
		t.Fatalf("a refused request marked someone %v %v", chosen, err)
	}
	if got := f.run(root, "bulk_revocation.remove_targets", draft, map[string]string{"users": "u-ann", "draft": draft}); got.status == 200 {
		t.Errorf("an unknown field %s", got.raw)
	}
	f.discard(root, draft)
	staged := f.stage(root, "", "u-ann", "u-hal")
	if got := f.remove(root, field(staged, "draft"), "u-ann"); code(got) != "invalid_stage" {
		t.Errorf("a draft of chosen people %s", got.raw)
	}
}

func TestRemovingAdministratorsLetsAFilterListBePrepared(t *testing.T) {
	f, p, root := swept(filteredWorld(t))
	p.until(SampleAdmin, f.s.clock().Add(30*24*time.Hour))
	f.advance(time.Second)
	fixed := f.match(root, "role=administrator")
	draft := field(fixed, "draft")
	if got := f.prepare(root, draft, "suspend_membership"); code(got) != "prerequisite_failed" {
		t.Fatalf("stranding the anchors %s", got.raw)
	}
	removed := []string{}
	var prepared reply
	for _, id := range f.draftUsers(SampleAdmin, draft) {
		if id == SampleAdmin {
			continue
		}
		if got := f.remove(root, draft, id); field(got, "removed") != "1" {
			t.Fatalf("removing %s: %s", id, got.raw)
		}
		removed = append(removed, id)
		if prepared = f.prepare(root, draft, "suspend_membership"); prepared.status == 200 {
			break
		}
		if code(prepared) != "prerequisite_failed" {
			t.Fatalf("prepare after removing %v: %s", removed, prepared.raw)
		}
	}
	if prepared.status != 200 {
		t.Fatalf("no removal let the list be prepared: %v", removed)
	}
	reasons := f.reasons(root, field(prepared, "bulk"))
	if reasons[SampleAdmin] != bulkrevoke.ExcludedSelf {
		t.Errorf("reasons %v", reasons)
	}
	targets := 0
	for id, reason := range reasons {
		switch {
		case slices.Contains(removed, id) && reason != bulkrevoke.ExcludedChosen && reason != bulkrevoke.ExcludedNoMembership && reason != bulkrevoke.ExcludedAlreadyRevoked:
			t.Errorf("%s was removed and is %s", id, reason)
		case reason == "target":
			targets++
		}
	}
	if targets == 0 || !slices.ContainsFunc(removed, func(id string) bool { return reasons[id] == bulkrevoke.ExcludedChosen }) {
		t.Errorf("targets %d, removed %v, reasons %v", targets, removed, reasons)
	}
}

func TestChosenIsCountedForFilterListsOnly(t *testing.T) {
	f, _, root := swept(filteredWorld(t))
	fixed := f.match(root, "membership=active&role=member")
	prepared := f.prepare(root, field(fixed, "draft"), "suspend_membership")
	if prepared.status != 200 {
		t.Fatalf("prepare %s", prepared.raw)
	}
	detail := f.bulkRead(root, "/"+field(prepared, "bulk"))
	if n, has := detail.body["exclusions"].(map[string]any)["chosen"]; !has || whole(n) != 0 || whole(detail.body["job"].(map[string]any)["chosen"]) != 0 {
		t.Errorf("a filter list without removals %s", detail.raw)
	}
	f.discard(root, field(prepared, "bulk"))
	id, _ := f.prepared(t, root, "suspend_membership", "u-ann", "u-hal")
	detail = f.bulkRead(root, "/"+id)
	if _, has := detail.body["exclusions"].(map[string]any)["chosen"]; has {
		t.Errorf("a list of chosen people %s", detail.raw)
	}
}
