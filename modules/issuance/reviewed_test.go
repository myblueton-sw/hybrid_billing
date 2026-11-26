package issuance

import (
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"reflect"
	"testing"
	"time"
)

func reviewedDraft(t *testing.T, kind string) (fixture, DraftView, []Reference, *receivables.Coverage) {
	t.Helper()
	f := standard(t)
	d := draftOf("reviewed", kind)
	d.Language = "en"
	v := must(f.r.Submit("ws", d, human("clerk")))
	k := must(f.r.KindOf("ws", kind))
	refs, c, _, err := f.r.bind("ws", k.Kind, v.Draft, f.clock.read())
	if err != nil {
		t.Fatal(err)
	}
	return f, v, refs, c
}

func TestReviewedApprovalExactAndImmutableReplay(t *testing.T) {
	f, v, refs, c := reviewedDraft(t, "erp")
	f.clock.advance(time.Minute)
	approved := must(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker")))
	if approved.Approver != "checker" || !approved.At.Equal(f.clock.read()) || approved.Number != "" {
		t.Fatal("approval evidence")
	}
	saved := viewDraft(&approved)
	refs[0].Ref.ID = "caller-mutated"
	c.Due = "2000-01-01"
	approved.References[0].Ref.ID = "returned-mutated"
	approved.Coverage.Due = "2000-01-01"
	if actual := must(f.r.DraftOf("ws", v.Draft.ID)); !reflect.DeepEqual(actual, saved) {
		t.Fatal("aliased approval")
	}
	f.clock.advance(time.Minute)
	replay := must(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, saved.References, saved.Coverage, human("checker")))
	if !reflect.DeepEqual(replay, saved) {
		t.Fatal("replay rebound timestamp or facts")
	}
	expect(t, "other actor", second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, saved.References, saved.Coverage, human("other"))), "conflict")
}

func TestReviewedApprovalEveryReferenceDimension(t *testing.T) {
	edits := []func(*Reference){func(v *Reference) { v.Field += "x" }, func(v *Reference) { v.Subject += "x" }, func(v *Reference) { v.Change += "x" }, func(v *Reference) { v.Ref.ID += "x" }, func(v *Reference) { v.Ref.Fingerprint += "x" }, func(v *Reference) { v.Effective = v.Effective.Add(time.Nanosecond) }}
	for i, edit := range edits {
		f, v, refs, c := reviewedDraft(t, "erp")
		edit(&refs[0])
		expect(t, "reference mismatch", second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker"))), ReasonReferenceChanged)
		if !reflect.DeepEqual(must(f.r.DraftOf("ws", v.Draft.ID)), v) {
			t.Fatalf("mutation for reference dimension %d", i)
		}
	}
	for _, mode := range []string{"nil", "reordered"} {
		f, v, refs, c := reviewedDraft(t, "erp")
		if mode == "nil" {
			refs = nil
		} else {
			refs[0], refs[1] = refs[1], refs[0]
		}
		expect(t, mode, second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker"))), ReasonReferenceChanged)
		if !reflect.DeepEqual(must(f.r.DraftOf("ws", v.Draft.ID)), v) {
			t.Fatal("mutation on reference form")
		}
	}
}

func TestReviewedApprovalEveryCoverageDimension(t *testing.T) {
	edits := []func(*receivables.Coverage){func(v *receivables.Coverage) { v.Contract += "x" }, func(v *receivables.Coverage) { v.Binding += "x" }, func(v *receivables.Coverage) { v.Term += "x" }, func(v *receivables.Coverage) { v.Revision++ }, func(v *receivables.Coverage) { v.NetDays++ }, func(v *receivables.Coverage) { v.Mode += "x" }, func(v *receivables.Coverage) { v.Zone += "x" }, func(v *receivables.Coverage) { v.Date = "2000-01-01" }, func(v *receivables.Coverage) { v.Due = "2000-01-01" }}
	for i, edit := range edits {
		f, v, refs, c := reviewedDraft(t, "erp")
		edit(c)
		expect(t, "coverage mismatch", second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker"))), ReasonCoverageMismatch)
		if !reflect.DeepEqual(must(f.r.DraftOf("ws", v.Draft.ID)), v) {
			t.Fatalf("mutation for coverage dimension %d", i)
		}
	}
	for _, kind := range []string{"erp", "int"} {
		f, v, refs, c := reviewedDraft(t, kind)
		wrong := (*receivables.Coverage)(nil)
		if c == nil {
			wrong = &receivables.Coverage{}
		}
		expect(t, "nil mismatch", second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, wrong, human("checker"))), ReasonCoverageMismatch)
		if !reflect.DeepEqual(must(f.r.DraftOf("ws", v.Draft.ID)), v) {
			t.Fatal("nil mismatch changed draft")
		}
		must(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker")))
	}
}

func TestReviewedApprovalDriftAndExplicitRefresh(t *testing.T) {
	for _, mode := range []string{"midnight", "reference"} {
		t.Run(mode, func(t *testing.T) {
			f, v, refs, c := reviewedDraft(t, "erp")
			want := ReasonCoverageMismatch
			if mode == "midnight" {
				f.clock.advance(24 * time.Hour)
			} else {
				want = ReasonReferenceChanged
				f.fields.set(SubjectCustomer+v.Draft.Customer, protected.FieldDocumentRecipient, "new-change", "new-ref", f.clock.read().Add(time.Minute))
				f.clock.advance(time.Minute)
			}
			expect(t, "drift", second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker"))), want)
			if !reflect.DeepEqual(must(f.r.DraftOf("ws", v.Draft.ID)), v) {
				t.Fatal("drift mutated unapproved draft")
			}
			k := must(f.r.KindOf("ws", v.Draft.Kind))
			refs, c, _, err := f.r.bind("ws", k.Kind, v.Draft, f.clock.read())
			if err != nil {
				t.Fatal(err)
			}
			approved := must(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker")))
			f.clock.advance(24 * time.Hour)
			expect(t, "approved stale review", second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker"))), ReasonCoverageMismatch)
			freshRefs, freshC, _, err := f.r.bind("ws", k.Kind, v.Draft, f.clock.read())
			if err != nil {
				t.Fatal(err)
			}
			expect(t, "cannot replace stored review", second(f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, freshRefs, freshC, human("checker"))), ReasonCoverageMismatch)
			if !reflect.DeepEqual(must(f.r.DraftOf("ws", v.Draft.ID)), approved) {
				t.Fatal("approved snapshot rebound")
			}
		})
	}
}

func TestReviewedApprovalProviderFailureNoMutation(t *testing.T) {
	for _, provider := range []string{"fields", "terms"} {
		f, v, refs, c := reviewedDraft(t, "erp")
		if provider == "fields" {
			f.fields.fail = true
		} else {
			f.terms.err = receivables.ErrNotFound
		}
		if _, err := f.r.ApproveReviewed("ws", v.Draft.ID, v.Digest, refs, c, human("checker")); err == nil {
			t.Fatal("provider failure accepted")
		}
		if !reflect.DeepEqual(must(f.r.DraftOf("ws", v.Draft.ID)), v) {
			t.Fatal("provider failure mutated draft")
		}
	}
}
