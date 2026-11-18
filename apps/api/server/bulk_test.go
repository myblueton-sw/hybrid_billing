package server

import (
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestBulkRevocationFlow(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	member := f.signedIn(MemberLogin, pw)
	outsider, _ := f.created(root, "outsider@example.invalid", "member")
	staged := f.stage(root, "", SampleMember, SampleAdmin2, SampleAdmin, "user-00000000000000ff", outsider, SampleMember)
	f.ok(t, "stage", staged, "draft", "map[count:5 draft:bulk-draft-1 duplicates:1]")
	page := f.bulkRead(root, "")
	if page.status != 200 || page.body["open_draft"] != true || page.body["can_prepare"] != true || len(page.body["jobs"].([]any)) != 1 ||
		page.body["jobs"].([]any)[0].(map[string]any)["staged"] != 5.0 {
		t.Fatalf("read with a draft %s", page.raw)
	}
	if got := f.bulkRead(f.as(Admin2Login), ""); len(got.body["jobs"].([]any)) != 0 || got.body["can_prepare"] != true {
		t.Errorf("another holder sees the draft %s", got.raw)
	}
	if got := f.bulkRead(f.as(Admin3Login), ""); got.body["can_prepare"] != false {
		t.Errorf("a holder of only administer_installation %s", got.raw)
	}
	prepared := f.prepare(root, "bulk-draft-1", "suspend_membership")
	f.ok(t, "prepare", prepared, "prepared", "")
	id, digest := field(prepared, "bulk"), field(prepared, "digest")
	if id != "bulk-revocation-1" || field(prepared, "targets") != "2" || field(prepared, "excluded") != "3" || len(digest) != 64 {
		t.Fatalf("prepared %s", prepared.raw)
	}
	detail := f.bulkRead(root, "/"+id)
	impact := detail.body["impact"].(map[string]any)
	exclusions := detail.body["exclusions"].(map[string]any)
	if impact["people"] != 2.0 || impact["administrators"] != 1.0 || impact["members"] != 1.0 || impact["installation_administrators"] != 1.0 ||
		whole(impact["sessions"]) < 1 || impact["api_keys"] != "not_applicable" || impact["anchors_after"] != 2.0 || impact["other_workspaces"] != "not_changed" ||
		len(impact["pending"].([]any)) != len(proposalStores) || exclusions["self"] != 1.0 || exclusions["not_found"] != 1.0 || exclusions["no_membership"] != 1.0 {
		t.Errorf("detail %s", detail.raw)
	}
	targets := f.bulkRead(root, "/"+id+"/targets?limit=1")
	if targets.status != 200 || targets.body["total"] != 2.0 || targets.body["next"] != "1" || len(targets.body["targets"].([]any)) != 1 {
		t.Errorf("targets %s", targets.raw)
	}
	if got := f.bulkRead(f.as(Admin2Login), "/"+id+"/targets"); got.status != 404 {
		t.Errorf("targets for someone else %s", got.raw)
	}
	f.fieldRefused(t, "a wrong digest", f.execute(root, id, "x"), "digest")
	got := f.execute(root, id, digest)
	if got.status != 202 || got.body["job_id"] != id || field(got, "targets") != "2" || field(got, "held") != "0" || field(got, "excluded") != "3" {
		t.Fatalf("execute %s", got.raw)
	}
	if me := f.me(member); me.status != 401 {
		t.Errorf("the member after the deny %+v", me)
	}
	if m := f.membership(SampleMember); m.State != identity.MembershipSuspended || m.SuspendedFor != identity.BulkRevocation {
		t.Errorf("member %+v", m)
	}
	header, items := f.records(t, decisionledger.BulkRevocationJob), f.records(t, decisionledger.SuspendMembership)
	if len(header) != 1 || len(items) != 2 || whole(header[0]["seq"]) >= whole(items[0]["seq"]) || header[0]["targets"] != 2.0 || header[0]["excluded"] != 3.0 ||
		header[0]["approved_by"] != SampleAdmin || items[0]["request"] != id || items[0]["v"] != 2.0 {
		t.Fatalf("records %+v %+v", header, items)
	}
	job := f.job(root, id)
	if job["state"] != "executed" || job["job_state"] != "running" || job["counts"].(map[string]any)["unprocessed"] != 2.0 {
		t.Errorf("before cleanup %+v", job)
	}
	if n := f.s.drainCleanup(); n != 1 {
		t.Errorf("tasks %d", n)
	}
	job = f.job(root, id)
	counts := job["counts"].(map[string]any)
	if job["job_state"] != "succeeded" || counts["succeeded"] != 2.0 || counts["targets"] != 2.0 || job["cleanup"].(map[string]any)["done"] != 2.0 ||
		whole(job["ledger"].(map[string]any)["last_sequence"]) <= whole(job["ledger"].(map[string]any)["first_sequence"]) {
		t.Errorf("after cleanup %+v", job)
	}
	done := f.items(root, id, "?outcome=succeeded")
	if len(done) != 2 || done[0]["state"] != bulkrevoke.Complete || len(done[0]["cleanup"].([]any)) != 10 {
		t.Errorf("items %+v", done)
	}
	if excluded := f.items(root, id, "?outcome=excluded&limit=2"); len(excluded) != 2 {
		t.Errorf("excluded items %+v", excluded)
	}
	f.failed(t, "execute twice", f.execute(root, id, digest), commands.InvalidStage)
	if row := f.row(root, SampleMember)["revocation"].(map[string]any); row["bulk"] != id || row["state"] != bulkrevoke.Complete {
		t.Errorf("row %+v", row)
	}
	f.ok(t, "reinstate after the bulk revocation", f.run(root, "identity.reinstate_membership", SampleMember, nil), "membership_active", "")
}

func TestBulkSuspendUserAndHeldTargets(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_user", SampleMember, SampleAdmin3)
	f.ok(t, "a role change between preparation and execution", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}), "role_member", "")
	got := f.execute(root, id, digest)
	if got.status != 202 || field(got, "held") != "1" {
		t.Fatalf("execute %s", got.raw)
	}
	held := f.items(root, id, "?outcome=held")
	if len(held) != 1 || held[0]["user_id"] != SampleAdmin3 || held[0]["reason"] != bulkrevoke.HeldChanged {
		t.Errorf("held %+v", held)
	}
	if _, state, _, _ := f.s.identity.Subjects().Subject(SampleMember); state != identity.Suspended {
		t.Errorf("member account %s", state)
	}
	if _, state, _, _ := f.s.identity.Subjects().Subject(SampleAdmin3); state != identity.Active {
		t.Errorf("a held person was suspended %s", state)
	}
	if len(f.records(t, decisionledger.SuspendUser)) != 1 {
		t.Error("user records")
	}
	f.s.drainCleanup()
	if job := f.job(root, id); job["job_state"] != "succeeded" || job["counts"].(map[string]any)["held"] != 1.0 {
		t.Errorf("job %+v", job)
	}
	otherID, otherDigest := f.prepared(t, f.as(Admin2Login), "suspend_membership", SampleAdmin3)
	f.failed(t, "another holder cannot run someone else's list", f.execute(root, otherID, otherDigest), commands.PermissionDenied)
}

func TestBulkAnchorRule(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.ok(t, "an end on the third", f.period(root, SampleAdmin3, f.at(48*time.Hour)), "role_period_set", "")
	f.ok(t, "an end on the first", f.period(f.as(Admin2Login), SampleAdmin, f.at(48*time.Hour)), "role_period_set", "")
	root = f.as(AdminLogin)
	staged := f.stage(root, "", SampleAdmin2)
	f.failed(t, "the last anchor", f.prepare(root, field(staged, "draft"), "suspend_membership"), commands.PrerequisiteFailed)
	f.ok(t, "discard", f.run(root, "bulk_revocation.discard", field(staged, "draft"), nil), "discarded", "")
	f.ok(t, "remove the end of the first", f.period(f.as(Admin2Login), SampleAdmin, ""), "role_period_set", "")
	id, digest := f.prepared(t, f.as(Admin2Login), "suspend_membership", SampleAdmin)
	f.ok(t, "an end on the second", f.period(f.as(AdminLogin), SampleAdmin2, f.at(48*time.Hour)), "role_period_set", "")
	f.failed(t, "a holder of administer_installation only", f.execute(f.as(Admin3Login), id, digest), commands.PermissionDenied)
	f.failed(t, "an execution that would leave no anchor", f.execute(f.as(Admin2Login), id, digest), commands.PrerequisiteFailed)
	if job := f.job(f.as(Admin2Login), id); job["state"] != "prepared" {
		t.Errorf("the refused list moved %+v", job)
	}
}

func TestBulkStagingLimitsAndOwnership(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.BulkMaxTargets, c.Access.BulkPending = 5000, 2
		c.Access.Given["bulk_revocation_max_targets"] = true
	})
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.withMember()
	ids := []string{SampleMember}
	for i := 1; len(ids) < 5000; i++ {
		ids = append(ids, "user-"+leftPad(strconv.FormatInt(int64(i), 16), 16))
	}
	per := 4096 / 22
	draft := ""
	for start, batch := 0, 0; start < len(ids); start, batch = start+per, batch+1 {
		chunk := ids[start:min(start+per, len(ids))]
		got := f.stage(root, draft, chunk...)
		if got.status != 200 {
			t.Fatalf("batch %d %s", batch, got.raw)
		}
		draft = field(got, "draft")
	}
	if got := f.stage(root, draft, "user-ffffffffffffffff"); got.body["code"] != string(commands.PrerequisiteFailed) {
		t.Errorf("one past the limit %s", got.raw)
	}
	exact := []string{}
	for len(exact) < 183 {
		exact = append(exact, "user-"+leftPad(strconv.Itoa(len(exact)), 16))
	}
	padded := append(append([]string{}, exact...), "user-"+leftPad("", 65))
	if n := len(joinLines(padded)); n != 4096 {
		t.Fatalf("padded %d", n)
	}
	if got := f.stage(second, "", padded...); got.status != 200 {
		t.Errorf("exactly 4096 bytes %s", got.raw)
	}
	over := append(append([]string{}, exact...), "user-"+leftPad("", 66))
	f.fieldRefused(t, "4097 bytes", f.stage(root, draft, over...), "users")
	f.failed(t, "a second open draft", f.stage(second, "", SampleMember), commands.PrerequisiteFailed)
	f.failed(t, "the pending limit", f.stage(f.as(Admin3Login), "", SampleMember), commands.PermissionDenied)
	foreign, unknown := f.stage(second, draft, SampleMember), f.stage(second, "bulk-draft-99", SampleMember)
	if foreign.status != unknown.status || foreign.body["code"] != unknown.body["code"] || foreign.body["code"] != string(commands.PermissionDenied) {
		t.Errorf("a foreign draft %s an unknown draft %s", foreign.raw, unknown.raw)
	}
	for _, command := range []string{"bulk_revocation.prepare", "bulk_revocation.discard"} {
		fields := map[string]string{}
		if command == "bulk_revocation.prepare" {
			fields["effect"] = "suspend_user"
		}
		a, b := f.runWith(second, "r1", command, draft, fields), f.runWith(second, "r1", command, "bulk-draft-99", fields)
		if a.status != b.status || a.body["code"] != b.body["code"] || a.body["code"] != string(commands.PermissionDenied) {
			t.Errorf("%s foreign %s unknown %s", command, a.raw, b.raw)
		}
	}
	got := f.prepare(root, draft, "suspend_membership")
	if got.status != 200 || field(got, "targets") != "1" || field(got, "excluded") != "4999" {
		t.Fatalf("prepare %s", got.raw)
	}
	if got := f.execute(root, field(got, "bulk"), field(got, "digest")); got.status != 202 {
		t.Errorf("execute %s", got.raw)
	}
	f.advance(31 * 60 * 1e9)
	root = f.as(AdminLogin)
	f.ok(t, "the expired draft frees the slot", f.stage(f.as(Admin2Login), "", SampleMember), "draft", "")
}

func leftPad(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}

func joinLines(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += "\n"
		}
		out += id
	}
	return out
}
