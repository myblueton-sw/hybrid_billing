package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
)

func TestInputReviewHTTPHoldStepUpAndReadFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, _, _ := provisionInputCapability(t, f, inputauthority.Confirm)
	member, _, _ = provisionInputCapability(t, f, inputauthority.Inspect)
	record := inputRecord(t, f)
	for _, value := range []string{"", "01", "-1", "4294967296"} {
		f.code(t, "invalid scope revision", decideInput(f, member, "confirm_publish", record.Evidence.Digest, value), 422, string(commands.InvalidRequest))
	}
	f.code(t, "stale scope", decideInput(f, member, "confirm_publish", record.Evidence.Digest, "1"), 409, string(commands.EvidenceStale))
	f.code(t, "stale digest", decideInput(f, member, "confirm_publish", strings.Repeat("0", 64), "0"), 409, string(commands.EvidenceStale))
	f.advance(16 * time.Minute)
	f.code(t, "step-up required", decideInput(f, member, "confirm_publish", record.Evidence.Digest, "0"), 403, CodeReauthentication)
	if got := inspectInput(f, member, sampleInputReview); got.status != 200 {
		t.Fatal("read requires mutation stepup", got.raw)
	}
	member = f.as(MemberLogin)
	f.s.domain.Lock()
	f.s.enterHold("synthetic-test")
	f.s.domain.Unlock()
	f.code(t, "held confirm", decideInput(f, member, "confirm_publish", record.Evidence.Digest, "0"), 503, string(commands.DependencyUnavailable))
	if got := inspectInput(f, member, sampleInputReview); got.status != 200 {
		t.Fatal("held inspection", got.raw)
	}
	f.ok(t, "held rejection", decideInput(f, member, "reject", record.Evidence.Digest, "0"), inputreview.Rejected, "")
}

func TestInputReviewHTTPReadRechecksBothGrantSides(t *testing.T) {
	for _, side := range []string{"human", "console"} {
		t.Run(side, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			member, ceiling, grant := provisionInputCapability(t, f, inputauthority.Inspect)
			if got := inspectInput(f, member, sampleInputReview); got.status != 200 {
				t.Fatal(got.raw)
			}
			if side == "console" {
				grant = ceiling
			}
			if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
				t.Fatal(err)
			}
			before := f.s.counter
			f.code(t, "revoked inspection", inspectInput(f, member, sampleInputReview), 403, string(commands.PermissionDenied))
			if before != f.s.counter {
				t.Fatal("denied inspection changed revision")
			}
		})
	}
}

func TestInputReviewHTTPConcurrentDecisionsPublishAtMostOnce(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, _, _ := provisionInputCapability(t, f, inputauthority.Confirm)
	record := inputRecord(t, f)
	revision := f.s.Revision()
	results := make([]reply, 2)
	var wg sync.WaitGroup
	for i, kind := range []string{"confirm_publish", "reject"} {
		wg.Add(1)
		go func(i int, kind string) {
			defer wg.Done()
			results[i] = f.runWith(member, revision, "input."+kind, sampleInputReview, map[string]string{"digest": record.Evidence.Digest, "scope_revision": "0"})
		}(i, kind)
	}
	wg.Wait()
	success := 0
	for _, got := range results {
		if got.status == 200 {
			success++
		} else if got.status != 409 {
			t.Fatal(got.raw)
		}
	}
	if success != 1 {
		t.Fatal("competing decisions", results)
	}
	after := inputRecord(t, f)
	publication, err := f.s.inputReviews.registry.Current(record.Evidence.Scope)
	if after.State == inputreview.Eligible {
		if err != nil || publication.Revision != 1 {
			t.Fatal(publication, err)
		}
	} else if after.State != inputreview.Rejected || !errors.Is(err, inputreview.ErrNotFound) {
		t.Fatal(after, publication, err)
	}
}

func TestInputReviewHTTPUnseededNeverPreparesOrGrants(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal(got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	root := actor{cookie, csrf}
	if f.s.inputReviews != nil {
		t.Fatal("unseeded reviews initialized")
	}
	for _, g := range f.s.admin.Grants() {
		if g.Terms.Kind == inputauthority.ResourceKind {
			t.Fatal("unseeded review grant")
		}
	}
	f.code(t, "no default inspection", inspectInput(f, root, sampleInputReview), 403, string(commands.PermissionDenied))
	for _, capability := range []string{inputauthority.Confirm, inputauthority.Inspect} {
		spec := inputauthority.Spec{Capability: capability, Workspace: EmptyWorkspace, Kind: inputauthority.Console, Recipient: ConsoleClient, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
		if err := f.s.admin.AddGrant(inputAuthorityGrant("empty-console-"+capability, spec)); err != nil {
			t.Fatal(err)
		}
		spec.Kind = inputauthority.Human
		spec.Recipient = EmptyAdmin
		if err := f.s.admin.AddGrant(inputAuthorityGrant("empty-human-"+capability, spec)); err != nil {
			t.Fatal(err)
		}
	}
	before := f.s.counter
	f.code(t, "unsupported inspection", inspectInput(f, root, sampleInputReview), 501, string(commands.UnsupportedCapability))
	for _, kind := range []string{"confirm_publish", "reject"} {
		f.code(t, "unsupported decision", decideInput(f, root, kind, strings.Repeat("a", 64), "0"), 501, string(commands.UnsupportedCapability))
	}
	if before != f.s.counter || f.s.inputReviews != nil {
		t.Fatal("unseeded operation changed state")
	}
}

func TestInputReviewHTTPCatalogAndInspectionDoNotPublish(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, _, _ := provisionInputCapability(t, f, inputauthority.Inspect)
	initial := inputRecord(t, f)
	before := f.s.counter
	for i := 0; i < 3; i++ {
		if got := inspectInput(f, member, sampleInputReview); got.status != 200 {
			t.Fatal(got.raw)
		}
	}
	if before != f.s.counter || !reflect.DeepEqual(initial, inputRecord(t, f)) {
		t.Fatal("inspection mutated review")
	}
	data, err := f.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, entry := range f.s.commands.Catalogue() {
		if entry.ID == "input.inspect" {
			if entry.Mutating || entry.Permission.Action != gate.ViewCost || !reflect.DeepEqual(entry.Permission.Fields, []string{inputauthority.EvidenceField}) {
				t.Fatal(entry)
			}
		}
		if entry.ID == "input.confirm_publish" || entry.ID == "input.reject" {
			if !entry.Mutating || entry.Permission.Action != gate.ConfirmInput || !reflect.DeepEqual(entry.Refusals, []commands.Code{commands.EvidenceStale}) {
				t.Fatal(entry)
			}
		}
	}
	for _, kind := range []string{"inspect", "confirm_publish", "reject"} {
		if !strings.Contains(string(data), "/commands/input."+kind) {
			t.Fatal("missing contract", kind)
		}
	}
}
