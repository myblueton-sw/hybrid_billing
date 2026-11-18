package server

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

func (f world) proposeFields(age, step string) map[string]string {
	base, _ := f.s.trustBase(SampleWorkspace)
	return map[string]string{"max_age_minutes": age, "step_up_minutes": step, "expected_epoch": strconv.FormatUint(base.Epoch, 10),
		"expected_policy": strconv.FormatUint(base.Policy, 10)}
}

func (f world) trustPropose(a actor, age, step string) reply {
	f.t.Helper()
	return f.run(a, "trust.propose", trustTarget, f.proposeFields(age, step))
}

func (f world) view(id string) trustchange.View {
	f.t.Helper()
	v, ok := f.s.trust.Get(id, SampleWorkspace)
	if !ok {
		f.t.Fatalf("no change %s", id)
	}
	return v
}

func (f world) epochText() string {
	base, _ := f.s.trustBase(SampleWorkspace)
	return strconv.FormatUint(base.Epoch, 10)
}

func (f world) policyText() string {
	base, _ := f.s.trustBase(SampleWorkspace)
	return strconv.FormatUint(base.Policy, 10)
}

func (f world) trustValidate(a actor, id string) reply {
	f.t.Helper()
	v := f.view(id)
	return f.run(a, "trust.validate", id, map[string]string{"expected_change_revision": strconv.FormatUint(v.Revision, 10), "change_digest": v.ChangeDigest,
		"expected_epoch": f.epochText()})
}

func (f world) decisionFields(id string) map[string]string {
	v := f.view(id)
	evidence := ""
	if v.Validation != nil {
		evidence = v.Validation.Digest
	}
	return map[string]string{"change_digest": v.ChangeDigest, "evidence_digest": evidence, "expected_epoch": f.epochText(), "expected_policy": f.policyText()}
}

func (f world) trustApprove(a actor, id string) reply {
	f.t.Helper()
	fields := f.decisionFields(id)
	delete(fields, "expected_policy")
	return f.run(a, "trust.approve", id, fields)
}

func (f world) trustActivate(a actor, id string) reply {
	f.t.Helper()
	return f.run(a, "trust.activate", id, f.decisionFields(id))
}

func (f world) trustEnd(a actor, command, id string) reply {
	f.t.Helper()
	return f.run(a, command, id, map[string]string{"expected_change_revision": strconv.FormatUint(f.view(id).Revision, 10)})
}

func changeOf(t testing.TB, got reply) string {
	t.Helper()
	if got.status != 200 {
		t.Fatalf("command %d %s", got.status, got.raw)
	}
	return got.body["fields"].(map[string]any)["change"].(string)
}

type admins struct{ a, b, c actor }

func (f world) trustAdmins() admins {
	f.t.Helper()
	for _, login := range []string{AdminLogin, Admin2Login, Admin3Login} {
		f.setPassword(login, pw)
	}
	return admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
}

func (f world) approvedChange(p admins, age, step string) string {
	f.t.Helper()
	id := changeOf(f.t, f.trustPropose(p.a, age, step))
	if got := f.trustValidate(p.b, id); got.status != 200 || got.body["business_state"] != trustchange.ResultValidated {
		f.t.Fatalf("validate %s", got.raw)
	}
	if got := f.trustApprove(p.b, id); got.status != 200 || got.body["business_state"] != trustchange.StateApproved {
		f.t.Fatalf("approve %s", got.raw)
	}
	return id
}

func TestTrustHappyPath(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	f.setPassword(MemberLogin, pw)
	member := f.signedIn(MemberLogin, pw)
	ref := f.run(p.c, "identity.create_user", usersTarget, map[string]string{"login": "x@example.invalid", "email": "x@example.invalid", "role": "member"})
	if ref.status != 200 || len(f.s.dir.pickups) != 1 {
		t.Fatalf("pickup %s", ref.raw)
	}
	proposed := f.admin(p.a, "trust.propose", trustTarget, "Shorter sign-in", f.proposeFields("60", "10"))
	id := changeOf(t, proposed)
	if proposed.body["business_state"] != trustchange.StateDraft || proposed.body["fields"].(map[string]any)["change_revision"] != "1" {
		t.Errorf("proposed %s", proposed.raw)
	}
	validated := f.trustValidate(p.b, id)
	fields := validated.body["fields"].(map[string]any)
	if validated.status != 200 || validated.body["business_state"] != "validated" || len(fields["evidence_digest"].(string)) != 64 || fields["blocked_items"] != nil {
		t.Fatalf("validated %s", validated.raw)
	}
	if got := f.trustApprove(p.b, id); got.body["business_state"] != trustchange.StateApproved {
		t.Fatalf("approved %s", got.raw)
	}
	before := len(f.s.ledger.Decisions())
	done := f.trustActivate(p.c, id)
	out := done.body["fields"].(map[string]any)
	if done.status != 200 || done.body["business_state"] != trustchange.StateActivated || out["epoch"] != "2" || out["epoch_generation"] != "0" || out["epoch_number"] != "2" || out["your_session_ended"] != "true" ||
		out["sessions_ended"] != "4" || out["sessions_remaining"] != "0" || out["ledger_sequence"] != strconv.Itoa(len(f.s.ledger.Decisions())+1) {
		t.Fatalf("activated %s", done.raw)
	}
	if len(f.s.ledger.Decisions()) != before+1 || f.s.ledger.Decisions()[before].Decision.Trust.NewEpoch != 2 {
		t.Error("the activation is not in the ledger")
	}
	current, _ := f.s.policy.Policy(SampleWorkspace)
	if current.Epoch != 2 || current.Number != 2 || current.Revision.MaxAge.Minutes() != 60 || current.Activator != SampleAdmin3 || current.Proposer != SampleAdmin ||
		current.Approver != SampleAdmin2 || current.Validator != SampleAdmin2 {
		t.Errorf("published %+v", current)
	}
	for name, a := range map[string]actor{"proposer": p.a, "approver": p.b, "activator": p.c, "member": member} {
		if _, ok := f.s.sessions.get(a.verifier(f)); ok {
			t.Errorf("the %s session survived", name)
		}
		f.code(t, name+" read", f.me(a), 401, string(commands.Unauthenticated))
	}
	if len(f.s.dir.pickups) != 0 {
		t.Error("a pickup issued by an ended session survived")
	}
	fresh := f.signedIn(AdminLogin, pw)
	if found, _ := f.s.sessions.get(fresh.verifier(f)); found.trustEpoch != 2 {
		t.Errorf("a new session has trust epoch %d", found.trustEpoch)
	}
	page := f.policyRead(fresh).body
	row := page["changes"].([]any)[0].(map[string]any)
	trust := page["trust"].(map[string]any)
	inv := row["invalidation"].(map[string]any)
	if row["state"] != trustchange.StateActivated || row["activation"].(map[string]any)["epoch"].(map[string]any)["value"] != "2" || inv["state"] != "complete" ||
		len(inv["items"].([]any)) != 7 || trust["origin"] != originProcess || trust["epoch"] != "2" || trust["change"] != id || trust["activator"] != SampleAdmin3 {
		t.Errorf("read after activation %+v %+v", row, trust)
	}
	history := row["history"].([]any)
	if len(history) != 4 || history[0].(map[string]any)["rationale"] != "Shorter sign-in" {
		t.Errorf("history %+v", history)
	}
}

func TestTrustSeparation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := changeOf(t, f.trustPropose(p.a, "60", "0"))
	f.code(t, "approve a draft", f.trustApprove(p.b, id), 422, string(commands.InvalidStage))
	f.code(t, "activate a draft", f.trustActivate(p.c, id), 422, string(commands.IndependentApprovalRequired))
	if got := f.trustValidate(p.a, id); got.status != 200 {
		t.Fatalf("the proposer may validate %s", got.raw)
	}
	f.code(t, "self approval", f.trustApprove(p.a, id), 403, string(commands.SelfApprovalDenied))
	f.code(t, "reject by the proposer", f.trustEnd(p.a, "trust.reject", id), 403, string(commands.SelfApprovalDenied))
	f.code(t, "cancel by another", f.trustEnd(p.b, "trust.cancel", id), 403, string(commands.PermissionDenied))
	f.code(t, "activate validated", f.trustActivate(p.c, id), 422, string(commands.IndependentApprovalRequired))
	if got := f.trustApprove(p.b, id); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.code(t, "the proposer activates", f.trustActivate(p.a, id), 403, string(commands.SelfApprovalDenied))
	f.code(t, "the approver activates", f.trustActivate(p.b, id), 403, string(commands.SelfApprovalDenied))
	row := f.policyRead(p.a).body["changes"].([]any)[0].(map[string]any)
	can, why := row["can"].(map[string]any), row["why_not"].(map[string]any)
	if can["approve"] != false || can["activate"] != false || can["cancel"] != true || can["revise"] != true || can["reject"] != false || why["activate"] != "self" {
		t.Errorf("proposer view %+v %+v", can, why)
	}
	row = f.policyRead(p.b).body["changes"].([]any)[0].(map[string]any)
	if row["can"].(map[string]any)["activate"] != false || row["why_not"].(map[string]any)["activate"] != "approver" || row["can"].(map[string]any)["reject"] != true {
		t.Errorf("approver view %+v", row)
	}
	row = f.policyRead(p.c).body["changes"].([]any)[0].(map[string]any)
	if row["can"].(map[string]any)["activate"] != true || row["why_not"].(map[string]any)["activate"] != "" || row["validation"].(map[string]any)["evidence_current"] != true {
		t.Errorf("activator view %+v", row)
	}
	if got := f.trustEnd(p.c, "trust.reject", id); got.status != 200 || got.body["business_state"] != trustchange.StateRejected {
		t.Errorf("reject %s", got.raw)
	}
	other := changeOf(t, f.trustPropose(p.a, "61", "0"))
	if got := f.trustEnd(p.a, "trust.cancel", other); got.status != 200 || got.body["business_state"] != trustchange.StateCancelled {
		t.Errorf("cancel %s", got.raw)
	}
}

func TestAdministrationConfersNoTrustRight(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := changeOf(t, f.trustPropose(p.a, "60", "0"))
	for _, right := range trustRights {
		g, _ := f.s.grantView("trust-" + string(right) + "-" + SampleAdmin2)
		f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	}
	b := f.as(Admin2Login)
	for command, got := range map[string]reply{
		"trust.propose":  f.trustPropose(b, "61", "0"),
		"trust.validate": f.trustValidate(b, id),
		"trust.approve":  f.trustApprove(b, id),
		"trust.activate": f.trustActivate(b, id),
		"trust.reject":   f.trustEnd(b, "trust.reject", id),
		"trust.cancel":   f.trustEnd(b, "trust.cancel", id),
	} {
		f.code(t, command, got, 403, string(commands.PermissionDenied))
	}
	if got := f.policyRead(b); got.status != 200 || got.body["rights"].(map[string]any)["approve"] != false {
		t.Errorf("an administrator without trust rights still reads the page %s", got.raw)
	}
	f.setPassword(MemberLogin, pw)
	m := f.as(MemberLogin)
	raw, _ := json.Marshal(map[string]any{"request_id": "m", "target_id": trustTarget, "idempotency_key": "m", "expected_revision": f.revision(b), "rationale": "x",
		"body": f.proposeFields("61", "0")})
	f.code(t, "a member", f.call("trust.propose", string(raw), m.cookie, m.csrf), 403, string(commands.PermissionDenied))
}

func TestTrustEpochConflicts(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	fields := f.proposeFields("60", "0")
	fields["expected_epoch"] = "7"
	f.code(t, "old epoch", f.run(p.a, "trust.propose", trustTarget, fields), 409, string(commands.AuthorityEpochConflict))
	fields = f.proposeFields("60", "0")
	fields["expected_policy"] = "9"
	f.code(t, "old policy", f.run(p.a, "trust.propose", trustTarget, fields), 409, string(commands.AuthorityEpochConflict))
	fields["expected_policy"] = "01"
	f.code(t, "a malformed number", f.run(p.a, "trust.propose", trustTarget, fields), 422, string(commands.InvalidRequest))
	first := f.approvedChange(p, "60", "0")
	second := f.approvedChange(p, "90", "0")
	stale := changeOf(t, f.trustPropose(p.b, "45", "0"))
	if got := f.trustActivate(p.c, first); got.status != 200 {
		t.Fatal(got.raw)
	}
	p = admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
	if v := f.view(second); v.State != trustchange.StateSuperseded || v.SupersededBy != first || v.SupersededEpoch != 2 {
		t.Errorf("superseded %+v", v)
	}
	f.code(t, "activate a superseded change", f.trustActivate(p.c, second), 422, string(commands.InvalidStage))
	if f.view(stale).State != trustchange.StateSuperseded {
		t.Error("a draft on the old epoch was not superseded")
	}
	rollback := f.approvedChange(p, "720", "0")
	done := f.trustActivate(p.c, rollback)
	if done.body["fields"].(map[string]any)["epoch"] != "3" {
		t.Errorf("a rollback gets a newer epoch %s", done.raw)
	}
}

func TestTrustConcurrentActivation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	one, two := f.approvedChange(p, "60", "0"), f.approvedChange(p, "90", "0")
	actors := []actor{p.c, f.signedIn(Admin3Login, pw)}
	bodies := make([]string, 2)
	for i, id := range []string{one, two} {
		body, _ := json.Marshal(map[string]any{"request_id": fmt.Sprintf("c%d", i), "target_id": id, "idempotency_key": fmt.Sprintf("c%d", i), "expected_revision": f.revision(p.c),
			"rationale": "go", "body": f.decisionFields(id)})
		bodies[i] = string(body)
	}
	if a, b := f.revision(p.c), f.revision(actors[1]); a != b {
		t.Fatalf("revision moved while preparing %s %s", a, b)
	}
	var wg sync.WaitGroup
	results := make([]reply, 2)
	for i := range bodies {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = f.call("trust.activate", bodies[i], actors[i].cookie, actors[i].csrf)
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, r := range results {
		if r.status == 200 {
			ok++
		} else if r.status != 401 && r.status != 409 && r.status != 422 {
			t.Errorf("the other activation %d %s", r.status, r.raw)
		}
	}
	activated := 0
	for _, id := range []string{one, two} {
		if f.view(id).State == trustchange.StateActivated {
			activated++
		}
	}
	if ok != 1 || activated != 1 {
		t.Errorf("activations %d %d: %d %s / %d %s", ok, activated, results[0].status, results[0].raw, results[1].status, results[1].raw)
	}
}

func TestTrustCandidateMutation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "0")
	old := f.decisionFields(id)
	fields := f.proposeFields("45", "0")
	fields["expected_change_revision"] = "1"
	revised := f.run(p.a, "trust.propose", id, fields)
	if revised.status != 200 || revised.body["fields"].(map[string]any)["change_revision"] != "2" || f.view(id).State != trustchange.StateDraft {
		t.Fatalf("revise %s", revised.raw)
	}
	f.code(t, "a stale revise", f.run(p.a, "trust.propose", id, fields), 409, string(commands.ApprovalStale))
	f.code(t, "revise by another", f.run(p.b, "trust.propose", id, map[string]string{"max_age_minutes": "40", "step_up_minutes": "0", "expected_epoch": "1",
		"expected_policy": "1", "expected_change_revision": "2"}), 403, string(commands.PermissionDenied))
	delete(fields, "expected_change_revision")
	f.code(t, "revise without a revision", f.run(p.a, "trust.propose", id, fields), 422, string(commands.InvalidRequest))
	f.code(t, "validate the old digest", f.run(p.b, "trust.validate", id, map[string]string{"expected_change_revision": "2", "change_digest": old["change_digest"],
		"expected_epoch": "1"}), 409, string(commands.ApprovalStale))
	if got := f.trustValidate(p.b, id); got.status != 200 {
		t.Fatal(got.raw)
	}
	stale := f.decisionFields(id)
	stale["change_digest"] = old["change_digest"]
	delete(stale, "expected_policy")
	f.code(t, "approve the old digest", f.run(p.b, "trust.approve", id, stale), 409, string(commands.ApprovalStale))
	stale = f.decisionFields(id)
	stale["evidence_digest"] = old["evidence_digest"]
	delete(stale, "expected_policy")
	f.code(t, "approve other evidence", f.run(p.b, "trust.approve", id, stale), 409, string(commands.EvidenceStale))
	f.code(t, "a draft identical to the revised change", f.trustPropose(p.a, "45", "0"), 422, string(commands.InvalidStage))
	if got := f.trustPropose(p.b, "45", "0"); got.status != 200 {
		t.Errorf("the same candidate from another proposer %s", got.raw)
	}
}

func TestTrustIdempotency(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "0")
	body := func(key string, fields map[string]string) string {
		f.s.domain.RLock()
		revision := f.s.Revision()
		f.s.domain.RUnlock()
		raw, _ := json.Marshal(map[string]any{"request_id": "r-" + key, "target_id": id, "idempotency_key": key, "expected_revision": revision, "rationale": "go",
			"body": fields})
		return string(raw)
	}
	fields := f.decisionFields(id)
	wrong := f.decisionFields(id)
	wrong["evidence_digest"] = "x"
	f.code(t, "a refused activation", f.call("trust.activate", body("k1", wrong), p.c.cookie, p.c.csrf), 409, string(commands.EvidenceStale))
	original := body("k1", fields)
	first := f.call("trust.activate", original, p.c.cookie, p.c.csrf)
	if first.status != 200 {
		t.Fatalf("a refusal reserved the key %s", first.raw)
	}
	again := f.signedIn(Admin3Login, pw)
	replay := f.call("trust.activate", original, again.cookie, again.csrf)
	if replay.status != 200 || replay.body["replay"] != true || replay.body["fields"].(map[string]any)["epoch"] != "2" {
		t.Errorf("replay %s", replay.raw)
	}
	f.code(t, "the same key with another body", f.call("trust.activate", body("k1", wrong), again.cookie, again.csrf), 409, string(commands.IdempotencyConflict))
	g, _ := f.s.grantView("trust-" + string(gate.TrustActivate) + "-" + SampleAdmin3)
	f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	f.code(t, "a replay after the right is lost", f.call("trust.activate", original, again.cookie, again.csrf), 403, string(commands.PermissionDenied))
}

func TestTrustTargetsDoNotLeak(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := changeOf(t, f.trustPropose(p.a, "60", "0"))
	bodies := map[string]bool{}
	for _, target := range []string{"trust-00000000-1", "missing", id + "x", "trust"} {
		got := f.run(p.b, "trust.validate", target, map[string]string{"expected_change_revision": "1", "change_digest": "x", "expected_epoch": "1"})
		f.code(t, target, got, 403, string(commands.PermissionDenied))
		delete(got.body, "request_id")
		raw, _ := json.Marshal(got.body)
		bodies[string(raw)] = true
	}
	if len(bodies) != 1 {
		t.Errorf("refusals differ %v", bodies)
	}
	if _, res := (trustCommand{s: f.s, kind: "validate"}).Resolve("elsewhere", id); res != commands.NotFound {
		t.Error("another workspace")
	}
	for _, rec := range f.s.ledger.Decisions() {
		if rec.Decision.Trust != nil {
			t.Error("a draft reached the ledger")
		}
	}
}
