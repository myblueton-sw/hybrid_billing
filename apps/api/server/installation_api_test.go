package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
)

func (f world) installation(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/installation", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func installRow(t testing.TB, got reply, number int) map[string]any {
	t.Helper()
	for _, p := range got.body["proposals"].([]any) {
		if row := p.(map[string]any); fmt.Sprint(row["number"]) == fmt.Sprint(number) {
			return row
		}
	}
	t.Fatalf("no proposal %d in %s", number, got.raw)
	return nil
}

func proposalAt(number int) string { return fmt.Sprintf("installation-proposal-%d", number) }

func blockedAs(row map[string]any) string {
	return fmt.Sprintf("%v %v %v %v", row["can_approve"], row["approve_blocked"], row["can_withdraw"], row["withdraw_blocked"])
}

var (
	sampleAdminActor  = installer.Actor{ID: SampleAdmin, Kind: installer.KindHuman}
	sampleAdmin2Actor = installer.Actor{ID: SampleAdmin2, Kind: installer.KindHuman}
	sampleProbeActor  = installer.Actor{ID: installProbe, Kind: installer.KindSystem}
)

func (f world) moduleApproved(c installer.Change) {
	f.t.Helper()
	p, err := f.s.installer.Propose(c, sampleAdminActor)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.s.installer.Approve(p.Number, sampleAdmin2Actor); err != nil {
		f.t.Fatal(err)
	}
}

func (f world) installThroughProbe(number uint64) {
	f.t.Helper()
	run, err := f.s.installer.StartRun(installSite, number, sampleAdminActor)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, step := range run.Steps[:len(run.Steps)-1] {
		digest := sampleDigest(strings.TrimPrefix(step, "artifact:"), run.Version)
		if !strings.HasPrefix(step, "artifact:") {
			digest = sampleDigest(step, "evidence")
		}
		if _, err := f.s.installer.Complete(installSite, run.ID, step, installer.Evidence{Digest: digest}, sampleProbeActor); err != nil {
			f.t.Fatalf("%s: %v", step, err)
		}
	}
	code, err := f.s.installer.IssueBootstrap(installSite, run.ID, time.Minute, sampleProbeActor)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.s.installer.RegisterAdmin(code, sampleAdmin2Actor); err != nil {
		f.t.Fatal(err)
	}
}

func TestInstallationReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.installation(root)
	if got.status != 200 || got.body["state"] != "configured" || got.body["now"] != stamp(start) || got.body["runs"] != "none" || got.body["revision"] != f.s.Revision() {
		t.Fatalf("read %s", got.raw)
	}
	if s := fmt.Sprint(got.body["site"]); s != "map[config_overrides:[] id:sample-site platform:linux/amd64 probe:sample-probe]" {
		t.Errorf("site %s", s)
	}
	if s := fmt.Sprint(got.body["observation"]); s != "map[expires_at:2026-11-17T09:00:00Z observed_at:2026-11-16T09:00:00Z state:fresh]" {
		t.Errorf("observation %s", s)
	}
	if s := fmt.Sprint(got.body["installed"]); s != "map[backups:[] history:[] services:[] state:not_installed version: volumes:[]]" {
		t.Errorf("installed %s", s)
	}
	releases := got.body["releases"].([]any)
	newest := releases[0].(map[string]any)
	if len(releases) != 2 || newest["version"] != "1.1.0" || releases[1].(map[string]any)["version"] != "1.0.0" || newest["key_version"] != installKey ||
		fmt.Sprint(newest["upgrade_from"]) != "[1.0.0]" || fmt.Sprint(releases[1].(map[string]any)["upgrade_from"]) != "[]" || newest["mark"] != nil {
		t.Fatalf("releases %v", releases)
	}
	if s := fmt.Sprint(newest["platforms"], newest["artifacts"], newest["volumes"], newest["services"]); s != "[darwin/arm64 linux/amd64 linux/arm64 windows/amd64] "+
		"[map[kind:container_image name:api] map[kind:archive name:worker]] [map[data:false name:config] map[data:true name:ledger]] "+
		"[map[artifact:api name:api ports:[8443/tcp] volumes:[config ledger]] map[artifact:worker name:worker ports:[9000/tcp] volumes:[ledger]]]" {
		t.Errorf("release %s", s)
	}
	preview := got.body["preview"].(map[string]any)
	if got.body["preview_basis"] != "pending_plan" && got.body["preview_basis"] != "newest_release" {
		t.Errorf("preview basis %v", got.body["preview_basis"])
	}
	plan := installRow(t, got, 6)
	if preview["state"] != "available" || preview["kind"] != "install" || preview["version"] != "1.0.0" || preview["hash"] != plan["summary"].(map[string]any)["plan_hash"] ||
		len(preview["hash"].(string)) != 64 || fmt.Sprint(preview["permissions"]) != "[create_volume listen_port run_service write_config]" ||
		fmt.Sprint(preview["data_effects"]) != "[map[effect:new volume:ledger]]" || fmt.Sprint(preview["conflicts"]) != "[]" || fmt.Sprint(preview["leftovers"]) != "[]" ||
		fmt.Sprint(preview["services"]) != "[map[action:create conflict: kind:service target:api] map[action:create conflict: kind:service target:worker]]" ||
		fmt.Sprint(preview["ports"]) != "[map[action:bind conflict: kind:port target:api:8443/tcp] map[action:bind conflict: kind:port target:worker:9000/tcp]]" ||
		fmt.Sprint(preview["volumes"]) != "[map[action:create conflict: kind:volume target:config] map[action:create conflict: kind:volume target:ledger]]" {
		t.Errorf("preview %v", preview)
	}
	for _, name := range []string{"upgrade", "rollback"} {
		if p := got.body[name].(map[string]any); p["state"] != "not_applicable" || p["kind"] != name || p["hash"] != "" {
			t.Errorf("%s %v", name, p)
		}
	}
	kinds := []string{}
	for _, p := range got.body["proposals"].([]any) {
		row := p.(map[string]any)
		kinds = append(kinds, fmt.Sprintf("%v:%v:%v", row["number"], row["kind"], row["state"]))
	}
	if strings.Join(kinds, ",") != "7:destination:pending,6:plan:pending,5:destination:applied,4:site:applied,3:release:applied,2:release:applied,1:key:applied" {
		t.Errorf("proposals %v", kinds)
	}
	if s := fmt.Sprintf("%v %v %v %v %v", plan["summary"].(map[string]any)["plan_admin"], plan["proposer"], plan["proposer_login"], plan["rationale"], plan["created_at"]); s !=
		"sample-admin-2 sample-admin-2 admin2@example.invalid Sample data 2026-11-16T09:00:00Z" {
		t.Errorf("plan %s", s)
	}
	if s := blockedAs(plan); s != "true  false not_proposer" {
		t.Errorf("plan blockers for the first administrator %s", s)
	}
	export := installRow(t, got, 7)
	if s := blockedAs(export); s != "false own_proposal true " {
		t.Errorf("export blockers %s", s)
	}
	if s := fmt.Sprint(export["summary"]); s != "map[destination:sample-billing-export host:export.example.invalid key_version: plan_admin: plan_hash: plan_version: port:443 ranges:[198.51.100.0/24] release_version: site:sample-site]" {
		t.Errorf("export summary %s", s)
	}
	key := installRow(t, got, 1)
	if s := fmt.Sprintf("%v %v %v %v %v %v", key["approver"], key["approver_login"], key["closed_by"], key["closing_rationale"], key["summary"].(map[string]any)["key_version"], blockedAs(key)); s !=
		"sample-admin-2 admin2@example.invalid sample-admin-2 Sample data sample-key-1 false not_pending false not_pending" {
		t.Errorf("key %s", s)
	}
	if s := fmt.Sprintf("%v %v", installRow(t, got, 3)["summary"].(map[string]any)["release_version"], installRow(t, got, 4)["summary"].(map[string]any)["site"]); s != "1.1.0 sample-site" {
		t.Errorf("summaries %s", s)
	}
	if s := fmt.Sprint(got.body["destinations"]); s != "[map[host:erp.example.invalid id:sample-erp-endpoint port:443 ranges:[192.0.2.0/24]]]" {
		t.Errorf("destinations %s", s)
	}
	if s := fmt.Sprint(got.body["reports"]); s != "[map[address:192.0.2.10 cause:untrusted_issuer destination:sample-erp-endpoint failure:unknown_authority kind:tls steps:[add_internal_ca_to_product_trust verify_chain] via:]]" {
		t.Errorf("reports %s", s)
	}
	second := f.installation(f.as(Admin2Login))
	if s := blockedAs(installRow(t, second, 6)); s != "false own_proposal true " {
		t.Errorf("plan blockers for its proposer %s", s)
	}
	if s := blockedAs(installRow(t, second, 7)); s != "true  false not_proposer" {
		t.Errorf("export blockers for the second administrator %s", s)
	}
}

func TestInstallationReadDoesNotLeakSigningMaterial(t *testing.T) {
	f := setup(t, SeedSample).ready()
	raw := f.installation(f.as(AdminLogin)).raw
	secrets := []string{}
	for _, version := range []string{"1.0.0", "1.1.0"} {
		secrets = append(secrets, sampleDigest("api", version), sampleDigest("worker", version))
	}
	for _, e := range f.s.install.entries {
		c := e.view.Change
		for _, b := range [][]byte{c.Key.Public, c.Key.Proof, c.Signature} {
			if len(b) > 0 {
				secrets = append(secrets, fmt.Sprintf("%x", b), encode(b), string(b))
			}
		}
	}
	if len(secrets) != 16 {
		t.Fatalf("secrets %d", len(secrets))
	}
	for _, secret := range secrets {
		if strings.Contains(raw, secret) {
			t.Errorf("read carries signing material %q", secret)
		}
	}
	for _, word := range []string{"public", "proof", "signature", "digest", "private"} {
		if strings.Contains(raw, word) {
			t.Errorf("read names %s", word)
		}
	}
}

func TestInstallationNotConfiguredAndAccess(t *testing.T) {
	none := setup(t, SeedNone).ready()
	tok := none.links()[""]
	if got := none.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, none.pre()); got.status != 204 {
		t.Fatalf("activate %+v", got)
	}
	cookie, csrf, err := none.s.openFor(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	got := none.installation(actor{cookie, csrf})
	if got.status != 200 || got.body["state"] != "not_configured" || got.body["site"] != nil || len(none.s.install.entries) != 0 {
		t.Errorf("empty %s", got.raw)
	}
	none.code(t, "no seed command", none.runWith(actor{cookie, csrf}, none.revision(actor{cookie, csrf}), "installation.propose_plan", installTarget,
		map[string]string{"version": "1.0.0", "hash": strings.Repeat("a", 64)}), 403, "permission_denied")
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	f.code(t, "member read", f.installation(member), 403, "permission_denied")
	f.code(t, "anonymous", f.installation(actor{}), 401, "unauthenticated")
	f.code(t, "member approve", f.runWith(member, f.revision(f.as(AdminLogin)), "installation.approve", proposalAt(6), map[string]string{}), 403, "permission_denied")
	root := f.as(AdminLogin)
	f.s.bootWorkspace = "elsewhere"
	if got := f.installation(root); got.status != 200 || got.body["state"] != "not_configured" {
		t.Errorf("other workspace %s", got.raw)
	}
	f.s.bootWorkspace = SampleWorkspace
	for _, kind := range []string{installApprove, installWithdraw, installPlan} {
		h := installHandler{s: f.s, kind: kind}
		target := proposalAt(6)
		if kind == installPlan {
			target = installTarget
		}
		if _, res := h.Resolve("other-ws", target); res != commands.NotFound {
			t.Errorf("%s resolved in another workspace", kind)
		}
		if _, res := h.Resolve(SampleWorkspace, target); res != commands.Found {
			t.Errorf("%s did not resolve", kind)
		}
	}
	for _, target := range []string{"installation", proposalAt(99), "installation-proposal-07", "installation-proposal-", "installation-proposal-x", "proposal-6"} {
		f.code(t, target, f.run(root, "installation.approve", target, nil), 403, "permission_denied")
	}
	f.code(t, "plan on a proposal target", f.run(root, "installation.propose_plan", proposalAt(6), map[string]string{"version": "1.0.0", "hash": strings.Repeat("a", 64)}), 403, "permission_denied")
	long := strings.Repeat("r", maxRationale+1)
	f.code(t, "long reason", f.admin(root, "installation.approve", proposalAt(6), long, map[string]string{}), 422, "invalid_request")
	f.code(t, "no reason", f.admin(root, "installation.approve", proposalAt(6), "", map[string]string{}), 422, "invalid_request")
	f.s.install.site = ""
	f.code(t, "no site", f.run(root, "installation.propose_plan", installTarget, map[string]string{"version": "1.0.0", "hash": strings.Repeat("a", 64)}), 403, "permission_denied")
	if got := f.installation(root); got.body["state"] != "not_configured" {
		t.Errorf("no site %s", got.raw)
	}
}

func TestInstallationApproveAndWithdraw(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	got := f.run(root, "installation.approve", proposalAt(6), nil)
	if got.status != 200 || got.body["business_state"] != "applied" || fmt.Sprint(got.body["fields"]) != "map[proposal:6 target:installation-proposal-6]" {
		t.Fatalf("approve %s", got.raw)
	}
	plan := installRow(t, f.installation(root), 6)
	if s := fmt.Sprint(plan["state"], " ", plan["approver"], " ", plan["closed_by_login"], " ", plan["closing_rationale"], " ", blockedAs(plan)); s !=
		"applied sample-admin admin@example.invalid sample reason false not_pending false not_pending" {
		t.Errorf("approved plan %s", s)
	}
	again := f.admin(root, "installation.approve", proposalAt(6), "repeat", map[string]string{})
	if again.status != 200 || again.body["business_state"] != "applied" || installRow(t, f.installation(root), 6)["closing_rationale"] != "sample reason" {
		t.Errorf("repeat approval %s", again.raw)
	}
	f.code(t, "approve applied", f.run(second, "installation.approve", proposalAt(6), nil), 422, "invalid_stage")
	f.code(t, "withdraw applied", f.run(second, "installation.withdraw", proposalAt(6), nil), 422, "invalid_stage")
	f.code(t, "self approval", f.run(root, "installation.approve", proposalAt(7), nil), 422, "prerequisite_failed")
	f.code(t, "not proposer", f.run(second, "installation.withdraw", proposalAt(7), nil), 422, "prerequisite_failed")
	gone := f.run(root, "installation.withdraw", proposalAt(7), nil)
	if gone.status != 200 || gone.body["business_state"] != "withdrawn" {
		t.Fatalf("withdraw %s", gone.raw)
	}
	read := f.installation(second)
	export := installRow(t, read, 7)
	if s := fmt.Sprint(export["state"], " ", export["closed_by"], " ", blockedAs(export)); s != "withdrawn sample-admin false not_pending false not_pending" {
		t.Errorf("withdrawn %s", s)
	}
	if n := len(read.body["destinations"].([]any)); n != 1 {
		t.Errorf("withdrawn destination applied %d", n)
	}
	f.code(t, "approve after withdraw", f.run(second, "installation.approve", proposalAt(7), nil), 422, "invalid_stage")
	f.code(t, "withdraw twice", f.run(root, "installation.withdraw", proposalAt(7), nil), 422, "invalid_stage")
	g := setup(t, SeedSample).ready()
	if got := g.run(g.as(Admin2Login), "installation.approve", proposalAt(7), nil); got.status != 200 {
		t.Fatalf("approve export %s", got.raw)
	}
	read = g.installation(g.as(AdminLogin))
	if s := fmt.Sprint(read.body["destinations"]); s != "[map[host:erp.example.invalid id:sample-erp-endpoint port:443 ranges:[192.0.2.0/24]] map[host:export.example.invalid id:sample-billing-export port:443 ranges:[198.51.100.0/24]]]" {
		t.Errorf("destinations %s", s)
	}
	if blockedAs(installRow(t, read, 6)) != "true  false not_proposer" {
		t.Errorf("a destination keeps the plan valid %v", installRow(t, read, 6))
	}
}

func TestInstallationProposePlan(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	next, err := f.s.installer.Preview(installSite, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{"version": "1.1.0", "hash": next.Hash}
	got := f.run(root, "installation.propose_plan", installTarget, fields)
	if got.status != 200 || got.body["business_state"] != "pending" || fmt.Sprint(got.body["fields"]) != "map[proposal:8 target:installation-proposal-8]" {
		t.Fatalf("propose %s", got.raw)
	}
	read := f.installation(root)
	mine := installRow(t, read, 8)
	if s := fmt.Sprint(mine["proposer"], " ", mine["summary"].(map[string]any)["plan_admin"], " ", mine["rationale"], " ", blockedAs(mine)); s !=
		"sample-admin sample-admin sample reason false own_proposal true " {
		t.Errorf("proposal %s", s)
	}
	if p := read.body["preview"].(map[string]any); p["version"] != "1.1.0" || p["hash"] != next.Hash || read.body["preview_basis"] != "pending_plan" {
		t.Errorf("preview follows the newest pending plan %v", p)
	}
	if again := f.admin(root, "installation.propose_plan", installTarget, "again", fields); again.status != 200 || fmt.Sprint(again.body["fields"]) != "map[proposal:8 target:installation-proposal-8]" {
		t.Errorf("duplicate %s", again.raw)
	}
	if n := len(f.installation(root).body["proposals"].([]any)); n != 8 || installRow(t, f.installation(root), 8)["rationale"] != "sample reason" {
		t.Errorf("duplicate stored %d", n)
	}
	for _, c := range []struct {
		fields map[string]string
		field  string
	}{
		{map[string]string{"version": "9.9.9", "hash": next.Hash}, "version"},
		{map[string]string{"version": "1.1", "hash": next.Hash}, "version"},
		{map[string]string{"version": "1.1.0", "hash": strings.ToUpper(next.Hash)}, "hash"},
		{map[string]string{"version": "1.1.0", "hash": next.Hash[:63]}, "hash"},
		{map[string]string{"version": "1.1.0", "hash": next.Hash[:63] + "g"}, "hash"},
	} {
		bad := f.run(root, "installation.propose_plan", installTarget, c.fields)
		if bad.status != 422 || bad.body["code"] != "invalid_request" || fmt.Sprint(bad.body["fields"]) != "["+c.field+"]" {
			t.Errorf("%v: %s", c.fields, bad.raw)
		}
	}
	f.code(t, "wrong hash", f.run(root, "installation.propose_plan", installTarget, map[string]string{"version": "1.0.0", "hash": strings.Repeat("0", 64)}), 422, "invalid_stage")
	busy := installer.Observation{Platform: installPlatform, Volumes: []string{"ledger"}, Complete: true}
	if _, err := f.s.installer.Observe(installSite, busy, sampleProbeActor); err != nil {
		t.Fatal(err)
	}
	current, err := f.s.installer.Preview(installSite, "1.0.0")
	if err != nil || len(current.Conflicts) == 0 {
		t.Fatalf("conflicting preview %+v %v", current, err)
	}
	plan := installRow(t, f.installation(root), 6)
	if plan["approve_blocked"] != installer.ReasonPlanChanged || plan["current_hash"] != current.Hash || plan["can_approve"] != false {
		t.Errorf("plan changed %v", plan)
	}
	f.code(t, "approve changed plan", f.run(root, "installation.approve", proposalAt(6), nil), 422, "invalid_stage")
	f.code(t, "propose with conflicts", f.run(root, "installation.propose_plan", installTarget, map[string]string{"version": "1.0.0", "hash": current.Hash}), 422, "invalid_stage")
	f.s.install.keep(&installEntry{view: installer.ProposalView{Number: 90, State: installer.ProposalPending, Proposer: SampleAdmin2,
		Change: installer.Change{Kind: installer.ChangePlan, Acceptance: installer.Acceptance{Site: installSite, Version: "1.0.0", Hash: current.Hash, Admin: SampleAdmin2}}}})
	if s := blockedAs(installRow(t, f.installation(root), 90)); s != "false conflicts false not_proposer" {
		t.Errorf("conflicts %s", s)
	}
}

func TestInstallationObservationStates(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.advance(25 * time.Hour)
	root := f.as(AdminLogin)
	read := f.installation(root)
	if s := fmt.Sprint(read.body["observation"].(map[string]any)["state"], " ", read.body["preview"].(map[string]any)["state"], " ", read.body["preview"].(map[string]any)["reason"]); s !=
		"stale unavailable observation_stale" {
		t.Errorf("stale %s", s)
	}
	if s := blockedAs(installRow(t, read, 6)); s != "false observation_stale false not_proposer" {
		t.Errorf("stale blockers %s", s)
	}
	f.code(t, "approve stale", f.run(root, "installation.approve", proposalAt(6), nil), 422, "invalid_stage")
	f.code(t, "propose stale", f.run(root, "installation.propose_plan", installTarget, map[string]string{"version": "1.0.0", "hash": strings.Repeat("a", 64)}), 422, "invalid_stage")
	g := setup(t, SeedSample).ready()
	g.moduleApproved(installer.Change{Kind: installer.ChangeConfig, Config: installer.SiteConfig{Site: installSite, Version: "1.0.0"}})
	read = g.installation(g.as(AdminLogin))
	if s := fmt.Sprint(read.body["observation"].(map[string]any)["state"], " ", read.body["preview"].(map[string]any)["reason"], " ", blockedAs(installRow(t, read, 6))); s !=
		"missing not_observed false not_observed false not_proposer" {
		t.Errorf("missing %s", s)
	}
	h := setup(t, SeedSample).ready()
	if _, err := h.s.installer.RevokeRelease("1.0.0", "sample revocation", sampleAdminActor); err != nil {
		t.Fatal(err)
	}
	if _, err := h.s.installer.ConfirmMark(installer.Target{Kind: installer.MarkRelease, Version: "1.0.0"}, sampleAdmin2Actor); err != nil {
		t.Fatal(err)
	}
	read = h.installation(h.as(AdminLogin))
	if s := fmt.Sprint(read.body["observation"].(map[string]any)["state"], " ", read.body["preview"].(map[string]any)["reason"]); s != "unknown revoked" {
		t.Errorf("revoked %s", s)
	}
	if s := fmt.Sprint(read.body["releases"].([]any)[1].(map[string]any)["mark"]); s != "map[by:sample-admin by_login:admin@example.invalid confirmed_by:sample-admin-2 confirmed_by_login:admin2@example.invalid reason:sample revocation]" {
		t.Errorf("mark %s", s)
	}
	h.s.install.releases, h.s.install.entries = nil, nil
	read = h.installation(h.as(AdminLogin))
	if s := fmt.Sprint(read.body["preview"].(map[string]any)["state"], " ", read.body["observation"].(map[string]any)["state"], " ", len(read.body["releases"].([]any))); s !=
		"not_applicable unknown 0" {
		t.Errorf("no releases %s", s)
	}
}

func TestInstallationInstalledViews(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	if got := f.run(root, "installation.approve", proposalAt(6), nil); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.installThroughProbe(6)
	read := f.installation(root)
	installed := read.body["installed"].(map[string]any)
	if s := fmt.Sprint(installed["state"], " ", installed["version"], " ", installed["services"], " ", len(installed["volumes"].([]any)), " ", len(installed["history"].([]any))); s !=
		"installed 1.0.0 [map[name:api ports:[8443/tcp]] map[name:worker ports:[9000/tcp]]] 2 1" {
		t.Errorf("installed %s", s)
	}
	if h := installed["history"].([]any)[0].(map[string]any); h["kind"] != "install" || h["to"] != "1.0.0" || fmt.Sprint(h["runs"]) != "[]" {
		t.Errorf("history %v", h)
	}
	preview, upgrade, rollback := read.body["preview"].(map[string]any), read.body["upgrade"].(map[string]any), read.body["rollback"].(map[string]any)
	if s := fmt.Sprint(preview["state"], " ", preview["version"], " ", preview["reason"]); s != "unavailable 1.1.0 already_installed" {
		t.Errorf("install preview %s", s)
	}
	want, err := f.s.installer.PreviewUpgrade(installSite, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if upgrade["state"] != "available" || upgrade["version"] != "1.1.0" || upgrade["hash"] != want.Hash || upgrade["kind"] != "upgrade" {
		t.Errorf("upgrade %v", upgrade)
	}
	if _, err := f.s.installer.PreviewRollback(installSite); rollback["state"] != "unavailable" || rollback["reason"] != installReason(err) || rollback["kind"] != "rollback" {
		t.Errorf("rollback %v %v", rollback, err)
	}
	if read.body["observation"].(map[string]any)["state"] != "fresh" || read.body["runs"] != "none" {
		t.Errorf("observation %v", read.body["observation"])
	}
	f.s.install.releases = []string{"1.0.0"}
	if p := f.installation(root).body["upgrade"].(map[string]any); p["state"] != "not_applicable" {
		t.Errorf("no newer release %v", p)
	}
}

func TestInstallationStaleRevisionAndReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	old := f.revision(root)
	if got := f.runWith(root, old, "installation.approve", proposalAt(6), map[string]string{}); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.code(t, "stale", f.runWith(second, old, "installation.approve", proposalAt(7), map[string]string{}), 409, "revision_conflict")
	body := `{"request_id":"req-i","target_id":"` + proposalAt(7) + `","idempotency_key":"key-install","expected_revision":"` + f.revision(second) +
		`","rationale":"r","body":{}}`
	first := f.call("installation.approve", body, second.cookie, second.csrf)
	again := f.call("installation.approve", body, second.cookie, second.csrf)
	if first.status != 200 || again.status != 200 || again.body["replay"] != true || fmt.Sprint(first.body["fields"]) != fmt.Sprint(again.body["fields"]) {
		t.Fatalf("replay %s %s", first.raw, again.raw)
	}
	if n := len(f.installation(root).body["destinations"].([]any)); n != 2 {
		t.Errorf("replay applied twice %d", n)
	}
}
