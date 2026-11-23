package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func issuanceAuthorityList(f world, a actor, namespace, kind string, fields map[string]string) reply {
	if fields == nil {
		fields = map[string]string{}
	}
	return documentRead(f, a, namespace+"."+kind, SampleWorkspace, fields)
}

func issuanceAuthorityRows(t *testing.T, got reply) []map[string]string {
	t.Helper()
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	rows := []map[string]string{}
	for i := 0; i < 10; i++ {
		raw := field(got, fmt.Sprintf("row_%02d", i))
		if raw == "" {
			break
		}
		var row map[string]string
		if err := json.Unmarshal([]byte(raw), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	return rows
}

func TestIssuanceAuthorityReadsFindProposalsRecipientsAndCeilings(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		t.Run(ns.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second, third, member := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login), f.as(MemberLogin)
			capability, other := ns.capabilities[0], ns.capabilities[1]
			empty := issuanceAuthorityList(f, root, ns.name, "history", nil)
			if rows := issuanceAuthorityRows(t, empty); len(rows) != 0 || field(empty, "current_eligibility") != "unchecked" || !validPlanHash(field(empty, "snapshot")) {
				t.Fatal(empty.raw)
			}
			f.code(t, "non-governor history", issuanceAuthorityList(f, member, ns.name, "history", nil), 403, string(commands.PermissionDenied))
			f.code(t, "non-governor context", issuanceAuthorityList(f, member, ns.name, "context", map[string]string{"collection": "recipients", "capability": capability}), 403, string(commands.PermissionDenied))
			f.code(t, "another target", documentRead(f, root, ns.name+".history", "another-target", map[string]string{}), 409, string(commands.EvidenceStale))
			for _, foreign := range issuanceAuthorityNamespaces {
				known := false
				for _, own := range ns.capabilities {
					known = known || own == foreign.capabilities[0]
				}
				if !known {
					f.code(t, "capability of "+foreign.name, issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "ceilings", "capability": foreign.capabilities[0]}), 422, string(commands.InvalidRequest))
				}
			}
			f.code(t, "unknown collection", issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "grants", "capability": capability}), 422, string(commands.InvalidRequest))

			ceiling := f.run(root, ns.name+".propose_console", "read-ceiling", issuanceAuthorityTimes(f, capability))
			f.ok(t, "ceiling proposal", ceiling, inputauthority.Pending, "")
			found := issuanceAuthorityRows(t, issuanceAuthorityList(f, second, ns.name, "history", nil))
			if len(found) == 1 {
				keys := []string{}
				for key := range found[0] {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				if !reflect.DeepEqual(keys, []string{"capability", "decision_actor", "digest", "expires_at", "kind", "proposal", "proposed_at", "proposer", "recipient", "state"}) || found[0]["decision_actor"] != "" {
					t.Fatal("history row fields", keys)
				}
			}
			if len(found) != 1 || found[0]["proposal"] != "read-ceiling" || found[0]["digest"] != field(ceiling, "digest") || found[0]["state"] != inputauthority.Pending || found[0]["proposer"] != SampleAdmin || found[0]["capability"] != capability {
				t.Fatal(found)
			}
			if rows := issuanceAuthorityRows(t, issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "ceilings", "capability": capability})); len(rows) != 0 {
				t.Fatal("a pending ceiling offered", rows)
			}
			applied := f.run(second, ns.name+".approve", "read-ceiling", map[string]string{"digest": found[0]["digest"]})
			f.ok(t, "ceiling approval from the listed digest", applied, inputauthority.Applied, "")
			ceilings := issuanceAuthorityRows(t, issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "ceilings", "capability": capability}))
			if len(ceilings) != 1 || ceilings[0]["ceiling"] != field(applied, "grant") || ceilings[0]["capability"] != capability {
				t.Fatal(ceilings)
			}
			if rows := issuanceAuthorityRows(t, issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "ceilings", "capability": other})); len(rows) != 0 {
				t.Fatal("a ceiling of another capability offered", rows)
			}
			recipients := issuanceAuthorityRows(t, issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "recipients", "capability": capability}))
			recipient := ""
			for _, row := range recipients {
				if row["user"] == SampleAdmin {
					t.Fatal("the caller offered as a recipient")
				}
				if row["user"] == SampleMember {
					recipient = row["user"]
				}
			}
			if recipient == "" {
				t.Fatal("an ordinary member omitted", recipients)
			}
			human := issuanceAuthorityTimes(f, capability)
			human["recipient"], human["ceiling"] = recipient, ceilings[0]["ceiling"]
			f.ok(t, "human proposal from listed values", f.run(root, ns.name+".propose_human", "read-human", human), inputauthority.Pending, "")
			human["not_after"] = f.s.clock().Add(11 * time.Hour).Format(time.RFC3339Nano)
			rejected := f.run(root, ns.name+".propose_human", "read-rejected", human)
			f.ok(t, "second proposal", rejected, inputauthority.Pending, "")
			f.ok(t, "rejection", f.run(third, ns.name+".reject", "read-rejected", map[string]string{"digest": field(rejected, "digest")}), inputauthority.Rejected, "")
			states := map[string]string{}
			for _, row := range issuanceAuthorityRows(t, issuanceAuthorityList(f, third, ns.name, "history", nil)) {
				states[row["proposal"]] = row["state"]
				if row["proposal"] == "read-human" {
					f.ok(t, "human approval from the listed digest", f.run(second, ns.name+".approve", "read-human", map[string]string{"digest": row["digest"]}), inputauthority.Applied, "")
				}
			}
			for _, row := range issuanceAuthorityRows(t, issuanceAuthorityList(f, third, ns.name, "history", nil)) {
				if row["proposal"] == "read-rejected" && row["decision_actor"] != SampleAdmin3 {
					t.Fatal("decision actor", row)
				}
			}
			if len(states) != 3 || states["read-ceiling"] != inputauthority.Applied || states["read-human"] != inputauthority.Pending || states["read-rejected"] != inputauthority.Rejected {
				t.Fatal(states)
			}
			for _, foreign := range issuanceAuthorityNamespaces {
				if foreign.name != ns.name {
					if rows := issuanceAuthorityRows(t, issuanceAuthorityList(f, root, foreign.name, "history", nil)); len(rows) != 0 {
						t.Fatalf("%s history shows %s proposals", foreign.name, ns.name)
					}
				}
			}
			if rows := issuanceAuthorityRows(t, resultAuthorityReadCall(f, root, "history", nil)); len(rows) != 0 {
				t.Fatal("result history shows issuance proposals")
			}

			f.s.enterHold("Provisioning held")
			if got := issuanceAuthorityList(f, root, ns.name, "history", nil); got.status != 200 {
				t.Fatal("history refused during hold", got.raw)
			}
			if got := issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "recipients", "capability": capability}); got.status != 200 {
				t.Fatal("context refused during hold", got.raw)
			}
			f.s.hold = nil
			f.advance(13 * time.Hour)
			root = f.as(AdminLogin)
			if rows := issuanceAuthorityRows(t, issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "ceilings", "capability": capability})); len(rows) != 0 {
				t.Fatal("an expired ceiling offered", rows)
			}
		})
	}
}

func TestIssuanceAuthorityReadsPageAndFailClosed(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		t.Run(ns.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			a := f.s.grantAuthority(ns.domain)
			for i := 0; i < 12; i++ {
				times := issuanceAuthorityTimes(f, ns.capabilities[0])
				times["not_after"] = f.s.clock().Add(time.Duration(i+1) * time.Hour).Format(time.RFC3339Nano)
				f.ok(t, "proposal", f.run(root, ns.name+".propose_console", fmt.Sprintf("page-%02d", i), times), inputauthority.Pending, "")
				f.advance(time.Second)
			}
			first := issuanceAuthorityList(f, root, ns.name, "history", nil)
			if rows := issuanceAuthorityRows(t, first); len(rows) != 10 || rows[0]["proposal"] != "page-00" || field(first, "next") != "page-09" {
				t.Fatal(first.raw)
			}
			second := issuanceAuthorityList(f, root, ns.name, "history", map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")})
			if rows := issuanceAuthorityRows(t, second); len(rows) != 2 || rows[1]["proposal"] != "page-11" || field(second, "next") != "" || field(second, "snapshot") != field(first, "snapshot") {
				t.Fatal(second.raw)
			}
			f.code(t, "cursor that is not a row", issuanceAuthorityList(f, root, ns.name, "history", map[string]string{"after": "page-99", "snapshot": field(first, "snapshot")}), 409, string(commands.EvidenceStale))
			f.code(t, "cursor without a snapshot", issuanceAuthorityList(f, root, ns.name, "history", map[string]string{"after": "page-09"}), 409, string(commands.EvidenceStale))
			f.ok(t, "one more", f.run(root, ns.name+".propose_console", "page-12", issuanceAuthorityTimes(f, ns.capabilities[1])), inputauthority.Pending, "")
			f.code(t, "old snapshot after a change", issuanceAuthorityList(f, root, ns.name, "history", map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")}), 409, string(commands.EvidenceStale))

			saved := f.s.cfg.Commands
			for _, change := range []func(){func() { f.s.cfg.Commands.Fields = 18 }, func() { f.s.cfg.Commands.Value = 1023 }} {
				change()
				f.code(t, "transport budget", issuanceAuthorityList(f, root, ns.name, "history", nil), 501, string(commands.UnsupportedCapability))
				f.s.cfg.Commands = saved
			}
			var key string
			for id := range a.bindings {
				key = id
				break
			}
			spec := a.bindings[key]
			forged := spec
			forged.NotAfter = forged.NotAfter.Add(time.Minute)
			a.bindings[key] = forged
			f.code(t, "binding differs from the record", issuanceAuthorityList(f, root, ns.name, "history", nil), 409, string(commands.EvidenceStale))
			a.bindings[key] = spec
			a.ceilings["foreign-ceiling"] = spec
			f.code(t, "ceiling without an applied record", issuanceAuthorityList(f, root, ns.name, "history", nil), 409, string(commands.EvidenceStale))
			delete(a.ceilings, "foreign-ceiling")
			if got := issuanceAuthorityList(f, root, ns.name, "history", nil); got.status != 200 {
				t.Fatal(got.raw)
			}
		})
	}
}

func TestIssuanceAuthorityReadsUnseededAndCapabilityLists(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		listed := ns.domain.readCapabilities()
		if len(listed) != len(ns.capabilities) || ns.domain.readNamespace() != ns.name {
			t.Fatal("read profile differs from the namespace", ns.name)
		}
		for _, capability := range listed {
			if _, err := ns.domain.terms(SampleWorkspace, capability); err != nil {
				t.Fatal("listed capability has no terms", ns.name, capability)
			}
		}
	}
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal(got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	f.must(err)
	root := actor{cookie, csrf}
	for _, ns := range issuanceAuthorityNamespaces {
		f.code(t, ns.name+".history", documentRead(f, root, ns.name+".history", EmptyWorkspace, map[string]string{}), 501, string(commands.UnsupportedCapability))
		f.code(t, ns.name+".context", documentRead(f, root, ns.name+".context", EmptyWorkspace, map[string]string{"collection": "recipients", "capability": ns.capabilities[0]}), 501, string(commands.UnsupportedCapability))
	}
}

func TestIssuanceAuthorityReadsSeparateNamespacesAndStates(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, third := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login)
	snapshots := map[string]string{}
	for _, ns := range issuanceAuthorityNamespaces {
		empty := issuanceAuthorityList(f, root, ns.name, "history", nil)
		if empty.status != 200 || snapshots[field(empty, "snapshot")] != "" {
			t.Fatal("empty histories of two namespaces share a snapshot", empty.raw)
		}
		snapshots[field(empty, "snapshot")] = ns.name
		context := issuanceAuthorityList(f, root, ns.name, "context", map[string]string{"collection": "recipients", "capability": ns.capabilities[0]})
		var policy struct {
			Capabilities []string `json:"capabilities"`
			Lifetime     string   `json:"proposal_lifetime_seconds"`
			Validity     string   `json:"max_grant_validity_seconds"`
		}
		f.must(json.Unmarshal([]byte(field(context, "policy")), &policy))
		if !reflect.DeepEqual(policy.Capabilities, ns.capabilities) || policy.Lifetime != "3600" || policy.Validity != "86400" || field(empty, "policy") != "" {
			t.Fatal(ns.name, context.raw)
		}
		for i, capability := range policy.Capabilities {
			f.ok(t, "listed capability accepted", f.run(root, ns.name+".propose_console", fmt.Sprintf("listed-%02d", i), issuanceAuthorityTimes(f, capability)), inputauthority.Pending, "")
			f.advance(time.Second)
		}
	}
	first, other := issuanceAuthorityNamespaces[0], issuanceAuthorityNamespaces[1]
	page := issuanceAuthorityList(f, root, first.name, "history", nil)
	rows := issuanceAuthorityRows(t, page)
	f.code(t, "cursor and snapshot of another namespace", issuanceAuthorityList(f, root, other.name, "history", map[string]string{"after": rows[0]["proposal"], "snapshot": field(page, "snapshot")}), 409, string(commands.EvidenceStale))

	provisionIssuanceCapability(t, f, first.name, first.capabilities[len(first.capabilities)-1], SampleAdmin, "inspector-")
	provisionIssuanceCapability(t, f, first.name, first.capabilities[len(first.capabilities)-1], SampleMember, "member-inspector-")
	if got := issuanceAuthorityList(f, f.as(AdminLogin), first.name, "history", nil); got.status != 200 {
		t.Fatal("a governor who holds inspect refused", got.raw)
	}
	f.code(t, "a non-governor who holds inspect", issuanceAuthorityList(f, f.as(MemberLogin), first.name, "history", nil), 403, string(commands.PermissionDenied))

	shared := first.capabilities[len(first.capabilities)-1]
	if rows := issuanceAuthorityRows(t, issuanceAuthorityList(f, f.as(AdminLogin), first.name, "context", map[string]string{"collection": "ceilings", "capability": shared})); len(rows) != 2 {
		t.Fatal("active ceilings of the shared name not listed in their own namespace", rows)
	}
	for _, ns := range issuanceAuthorityNamespaces[1:] {
		if rows := issuanceAuthorityRows(t, issuanceAuthorityList(f, f.as(AdminLogin), ns.name, "context", map[string]string{"collection": "ceilings", "capability": shared})); len(rows) != 0 {
			t.Fatalf("%s lists a ceiling of %s for the shared name", ns.name, first.name)
		}
	}
	uncertain := f.run(second, other.name+".propose_console", "uncertain", issuanceAuthorityTimes(f, other.capabilities[0]))
	f.ok(t, "proposal", uncertain, inputauthority.Pending, "")
	f.s.fault = func(at string) bool { return at == other.domain.faultPrefix()+"after-grant" }
	if lost := f.run(third, other.name+".approve", "uncertain", map[string]string{"digest": field(uncertain, "digest")}); lost.status == 200 {
		t.Fatal(lost.raw)
	}
	f.s.fault = nil
	f.advance(2 * time.Hour)
	root = f.as(AdminLogin)
	states := map[string]string{}
	for after, snapshot := "", ""; ; {
		fields := map[string]string{}
		if after != "" {
			fields["after"], fields["snapshot"] = after, snapshot
		}
		got := issuanceAuthorityList(f, root, other.name, "history", fields)
		for _, row := range issuanceAuthorityRows(t, got) {
			states[row["proposal"]] = row["state"]
		}
		after, snapshot = field(got, "next"), field(got, "snapshot")
		if after == "" {
			break
		}
	}
	if states["uncertain"] != inputauthority.Unknown || states["listed-00"] != inputauthority.Expired {
		t.Fatal(states)
	}
	if got := issuanceAuthorityRead(f, root, other.name, "uncertain"); field(got, "state") != inputauthority.Unknown {
		t.Fatal("a history read settled an unknown decision", got.raw)
	}
}
