package server

import (
	"bytes"
	"encoding/json"
	"net/url"
	"testing"
)

func TestSessionSelectionBuildingPreviewAndOlderTokenDoNotWidenFrozenSet(t *testing.T) {
	f, a := selectionPopulation(t, 350)
	first := f.selfPost(a, "sessions/selections", "{}")
	if first.status != 200 || first.body["state"] != "building" || first.body["count"] != nil {
		t.Fatal("bounded building premise")
	}
	ref := first.body["selection"].(string)
	preview := f.selectionRead(a, ref+"?limit=1")
	if preview.status != 200 || preview.body["state"] != "building" || preview.body["count"] != nil || len(preview.body["rows"].([]any)) != 0 || preview.body["next"] != "" {
		t.Fatal("building preview invented a sealed set")
	}
	if got := f.selectionRead(a, ref+"?limit=1&after="+url.QueryEscape(first.body["after"].(string))); got.status != 409 || got.body["rows"] != nil {
		t.Fatal("building preview accepted a continuation")
	}
	second := f.selfPost(a, "sessions/selections", selectionResumeBody(first))
	third := f.selfPost(a, "sessions/selections", selectionResumeBody(second))
	if second.status != 200 || third.status != 200 || second.body["state"] != "building" || third.body["state"] != "building" || countOf(third.body["observed"]) <= countOf(second.body["observed"]) {
		t.Fatal("three genuine bounded build positions were not produced")
	}
	beforeUnits := f.s.sessionRecordUnits
	old := f.selfPost(a, "sessions/selections", selectionResumeBody(first))
	if old.status != 409 || old.body["rows"] != nil || f.s.sessionRecordUnits != beforeUnits || len(f.s.selfEnds) != 0 {
		t.Fatal("older-than-previous build token was accepted or changed effects")
	}
	sealed := f.selfPost(a, "sessions/selections", selectionResumeBody(third))
	if sealed.status != 200 || sealed.body["state"] != "sealed" || countOf(sealed.body["count"]) != 350 {
		t.Fatal("valid build continuation did not seal the exact set")
	}
	readFrozen := func(operation string) []any {
		all := []any{}
		seen := map[string]bool{}
		after := ""
		for page := 0; page < 5; page++ {
			path := ref + "?limit=100"
			if after != "" {
				path += "&after=" + url.QueryEscape(after)
			}
			got := f.selectionRead(a, path)
			if got.status != 200 || countOf(got.body["count"]) != 350 {
				t.Fatal("frozen preview count or admission changed")
			}
			if operation != "" && (got.body["operation"] != operation || got.body["state"] != "accepted") {
				t.Fatal("accepted preview lost operation identity")
			}
			rows := got.body["rows"].([]any)
			if len(rows) > 100 {
				t.Fatal("preview exceeded requested page bound")
			}
			for _, raw := range rows {
				handle := raw.(map[string]any)["handle"].(string)
				if seen[handle] || handle == handleOfActor(a) {
					t.Fatal("frozen preview duplicated a target or included its caller")
				}
				seen[handle] = true
				all = append(all, raw)
			}
			next := got.body["next"].(string)
			if next == "" {
				if len(all) != 350 {
					t.Fatal("frozen preview omitted targets")
				}
				return all
			}
			if next == after {
				t.Fatal("frozen preview did not advance")
			}
			after = next
		}
		t.Fatal("bounded frozen preview traversal did not terminate")
		return nil
	}
	before := readFrozen("")
	accepted := f.selfPost(a, "sessions/end", selectionEndBody(sealed))
	if accepted.status != 202 || countOf(accepted.body["recorded_count"]) != 350 {
		t.Fatal("sealed set did not reach one accepted operation", accepted.status)
	}
	for key, want := range map[string]int{"ended_count": 350, "gone_count": 0, "not_included_count": 0} {
		if accepted.body[key] == nil || countOf(accepted.body[key]) != want {
			t.Fatal("accepted operation did not account for the exact frozen population", key)
		}
	}
	operation, known := accepted.body["operation"].(string)
	if !known || operation == "" {
		t.Fatal("accepted operation identity missing")
	}
	after := readFrozen(operation)
	beforeRows, err := json.Marshal(before)
	f.must(err)
	afterRows, err := json.Marshal(after)
	f.must(err)
	if !bytes.Equal(beforeRows, afterRows) {
		t.Fatal("accepted preview replaced frozen metadata with current state")
	}
	if got := f.me(a); got.status != 200 {
		t.Fatal("acceptance ended its own caller")
	}
	assertSessionRecordAccounting(t, f)
}

func TestLegacySessionEndQuotaStagesHaveNoEffectAndReleaseExactly(t *testing.T) {
	for _, budget := range []int{256, 257, 259} {
		f := setupWith(t, func(c *Config) {
			c.Identity.SessionsPerUser = 100
			c.Access = DefaultAccessSettings()
			c.Access.SessionSelectionTargets = 64
			c.Access.SessionRecordBudget = budget
		})
		f.setPassword(MemberLogin, pw)
		actors := []actor{f.signedIn(MemberLogin, pw)}
		for len(actors) < 64 {
			cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
			f.must(err)
			actors = append(actors, actor{cookie, csrf})
		}
		for _, caller := range actors[:4] {
			prepared := f.selfPost(caller, "sessions/selections", "{}")
			if prepared.status != 200 || prepared.body["state"] != "sealed" || countOf(prepared.body["count"]) != 63 {
				t.Fatal("four legitimate pending views were not created", budget)
			}
		}
		if f.s.sessionRecordUnits != 256 || len(f.s.sessionRecords) != 4 {
			t.Fatal("pending views did not charge exact references", budget)
		}
		before := f.s.Revision()
		body, err := json.Marshal(map[string]string{"session": handleOfActor(actors[1])})
		f.must(err)
		got := f.selfPost(actors[0], "sessions/end", string(body))
		if got.status != 429 || got.body["operation"] != nil || f.s.Revision() != before || len(f.s.selfEnds) != 0 || len(f.s.sessionOperations) != 0 || len(f.s.sessionRecords) != 4 || f.s.sessionRecordUnits != 256 {
			t.Fatal("legacy quota failure changed sessions or leaked reservations", budget, got.status)
		}
		for _, peer := range actors {
			if f.me(peer).status != 200 {
				t.Fatal("legacy quota refusal ended a peer", budget)
			}
		}
		assertSessionRecordAccounting(t, f)
	}
}
