package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
)

func inspectInput(f world, a actor, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "input-read", "target_id": id, "body": map[string]string{}})
	return f.call("input.inspect", string(body), a.cookie, a.csrf)
}
func decideInput(f world, a actor, kind, digest, scope string) reply {
	return f.runWith(a, f.s.Revision(), "input."+kind, sampleInputReview, map[string]string{"digest": digest, "scope_revision": scope})
}
func inputRecord(t *testing.T, f world) inputreview.Record {
	t.Helper()
	record, err := f.s.inputReviews.registry.Record(SampleWorkspace, sampleInputReview)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestInputReviewHTTPLiveSeparatePermissions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	record := inputRecord(t, f)
	if record.State != inputreview.Awaiting || record.Decision != nil {
		t.Fatal("seed confirmed input")
	}
	if _, err := f.s.inputReviews.registry.Current(record.Evidence.Scope); !errors.Is(err, inputreview.ErrNotFound) {
		t.Fatal("seed published eligibility", err)
	}
	root, member := f.as(AdminLogin), f.as(MemberLogin)
	f.code(t, "admin lacks inspection", inspectInput(f, root, sampleInputReview), 403, string(commands.PermissionDenied))
	f.code(t, "member lacks confirmation", decideInput(f, member, "confirm_publish", record.Evidence.Digest, "0"), 403, string(commands.PermissionDenied))
	member, _, _ = provisionInputCapability(t, f, inputauthority.Inspect)
	before := f.s.counter
	inspected := inspectInput(f, member, sampleInputReview)
	if inspected.status != 200 || field(inspected, "document_total") != "15.0000" || field(inspected, "current_evidence") != "matched" || field(inspected, "current_eligibility") != "not_eligible" || field(inspected, "scope_revision") != "0" || field(inspected, "confirmation_permission") != "denied" || field(inspected, "confirmation_hold") != "clear" || inspected.body["result_revision"] != f.s.Revision() || f.s.counter != before {
		t.Fatal("inspection", inspected.raw)
	}
	if !reflect.DeepEqual(record, inputRecord(t, f)) {
		t.Fatal("inspection changed record")
	}
	f.code(t, "inspection is not confirmation", decideInput(f, member, "confirm_publish", record.Evidence.Digest, "0"), 403, string(commands.PermissionDenied))
	member, _, _ = provisionInputCapability(t, f, inputauthority.Confirm)
	inspected = inspectInput(f, member, sampleInputReview)
	confirmed := f.runWith(member, inspected.body["result_revision"].(string), "input.confirm_publish", sampleInputReview, map[string]string{"digest": field(inspected, "digest"), "scope_revision": field(inspected, "scope_revision")})
	f.ok(t, "confirmed", confirmed, inputreview.Eligible, "")
	if field(confirmed, "recorded_scope_revision") != "1" || field(confirmed, "receipt") != "historical_decision" {
		t.Fatal(confirmed.raw)
	}
	publication, err := f.s.inputReviews.registry.Current(record.Evidence.Scope)
	if err != nil || publication.Revision != 1 || publication.Record != sampleInputReview {
		t.Fatal(publication, err)
	}
	current := inspectInput(f, member, sampleInputReview)
	if current.status != 200 || field(current, "current_eligibility") != "eligible" || field(current, "confirmation_permission") != "allowed" {
		t.Fatal(current.raw)
	}
	repeat := decideInput(f, member, "confirm_publish", record.Evidence.Digest, "0")
	f.ok(t, "new-key replay", repeat, inputreview.Eligible, "")
	after, _ := f.s.inputReviews.registry.Current(record.Evidence.Scope)
	if after != publication {
		t.Fatal("repeat republished")
	}
	f.code(t, "opposite decision", decideInput(f, member, "reject", record.Evidence.Digest, "0"), 409, string(commands.EvidenceStale))
}

func TestInputReviewHTTPConfirmWithoutInspectionAndReject(t *testing.T) {
	for _, kind := range []string{"confirm_publish", "reject"} {
		t.Run(kind, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			member, _, _ := provisionInputCapability(t, f, inputauthority.Confirm)
			record := inputRecord(t, f)
			f.code(t, "confirmation is not inspection", inspectInput(f, member, sampleInputReview), 403, string(commands.PermissionDenied))
			got := decideInput(f, member, kind, record.Evidence.Digest, "0")
			state := inputreview.Eligible
			if kind == "reject" {
				state = inputreview.Rejected
			}
			f.ok(t, "decision", got, state, "")
			if kind == "reject" {
				if _, err := f.s.inputReviews.registry.Current(record.Evidence.Scope); !errors.Is(err, inputreview.ErrNotFound) {
					t.Fatal("rejection published", err)
				}
			}
		})
	}
}

func TestInputReviewHTTPHistoricalReplayAndFreshMappingCheck(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, _, grant := provisionInputCapability(t, f, inputauthority.Confirm)
	member, _, _ = provisionInputCapability(t, f, inputauthority.Inspect)
	record := inputRecord(t, f)
	body, _ := json.Marshal(map[string]any{"request_id": "input-decision", "target_id": sampleInputReview, "idempotency_key": "input-decision-key", "expected_revision": f.s.Revision(), "rationale": "sample reason", "body": map[string]string{"digest": record.Evidence.Digest, "scope_revision": "0"}})
	first := f.call("input.confirm_publish", string(body), member.cookie, member.csrf)
	f.ok(t, "first", first, inputreview.Eligible, "")
	p := f.s.chargeProducer
	run, err := p.normal.Run(SampleWorkspace, "sample-normalized-charge")
	if err != nil {
		t.Fatal(err)
	}
	mapping, err := p.normal.Revision(run.Spec.Mapping, run.Spec.Revision)
	if err != nil {
		t.Fatal(err)
	}
	for i := range mapping.Entries {
		if mapping.Entries[i].Field == "class" {
			mapping.Entries[i].Source = "service"
		}
	}
	if _, err := p.normal.DeclareMapping(run.Spec.Mapping, run.Spec.Revision, mapping.Entries); err != nil {
		t.Fatal(err)
	}
	before := f.s.counter
	replay := f.call("input.confirm_publish", string(body), member.cookie, member.csrf)
	f.ok(t, "historical replay", replay, inputreview.Eligible, "")
	if f.s.counter != before || field(replay, "receipt") != "historical_decision" {
		t.Fatal("replay changed publication")
	}
	f.code(t, "fresh attempt stale", decideInput(f, member, "confirm_publish", record.Evidence.Digest, "0"), 409, string(commands.EvidenceStale))
	inspected := inspectInput(f, member, sampleInputReview)
	if inspected.status != 200 || field(inspected, "recorded_state") != inputreview.Eligible || field(inspected, "current_eligibility") != "unverified" || field(inspected, "current_evidence") != "changed" {
		t.Fatal(inspected.raw)
	}
	if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
		t.Fatal(err)
	}
	f.code(t, "revoked cached replay", f.call("input.confirm_publish", string(body), member.cookie, member.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "revoked fresh attempt", decideInput(f, member, "confirm_publish", record.Evidence.Digest, "0"), 403, string(commands.PermissionDenied))
	if got := inspectInput(f, member, sampleInputReview); got.status != 200 || field(got, "confirmation_permission") != "denied" {
		t.Fatal("revoked confirmation hint", got.raw)
	}
}
