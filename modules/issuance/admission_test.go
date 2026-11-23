package issuance

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func admissionState(t *testing.T, f fixture) string {
	t.Helper()
	all := map[string]any{}
	for id, s := range f.r.workspaces {
		drafts := map[string]any{}
		for key, d := range s.drafts {
			drafts[key] = []any{d.revisions, d.current, d.reserved, d.cancelled}
		}
		docs := map[string]DocumentView{}
		for key, d := range s.documents {
			docs[key] = viewDocument(d)
		}
		series := map[string]any{}
		for key, v := range s.series {
			series[key] = []any{v.view, v.numbers}
		}
		all[id] = []any{drafts, docs, series, s.external, s.numbers}
	}
	raw, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func checkUnchanged(t *testing.T, f fixture, want string, check func() error) {
	t.Helper()
	before := admissionState(t, f)
	expect(t, "preflight", check(), want)
	if admissionState(t, f) != before {
		t.Fatal("preflight changed retained state")
	}
}

func TestIssueAdmissionNoReservationAndCurrentRecheck(t *testing.T) {
	f := standard(t)
	v := f.ready(t, "ws", draftOf("d", "inv"))
	check := func() error { return f.r.CheckIssue("ws", "d", v.Digest, human("issuer")) }
	checkUnchanged(t, f, "ok", check)
	checkUnchanged(t, f, "ok", check)
	f.fields.set(SubjectCustomer+"cu1", protected.FieldDocumentRecipient, "new-change", "new-recipient", f.clock.read())
	checkUnchanged(t, f, ReasonReferenceChanged, check)
	expect(t, "effect rechecks", second(f.r.Issue("ws", "d", human("issuer"))), ReasonReferenceChanged)
	must(f.r.Approve("ws", "d", v.Digest, human("lead")))
	checkUnchanged(t, f, "ok", check)
	first := must(f.r.Issue("ws", "d", human("issuer")))
	checkUnchanged(t, f, ReasonAlreadyIssued, check)
	f.fields.clear(SubjectCustomer+"cu1", protected.FieldDocumentRecipient)
	replay := must(f.r.Issue("ws", "d", human("issuer")))
	if !reflect.DeepEqual(first, replay) {
		t.Fatal("historical replay changed")
	}
}

func TestIssueAdmissionRefusalsAndCapacity(t *testing.T) {
	for _, scenario := range []string{"digest", "unapproved", "terms", "documents", "numbers", "width", "reserved", "reserved_documents", "reserved_references", "reserved_series", "foreign", "assistant"} {
		t.Run(scenario, func(t *testing.T) {
			f := standard(t)
			v := f.ready(t, "ws", draftOf("d", "inv"))
			workspace, digest, actor, want := "ws", v.Digest, human("issuer"), "ok"
			if len(scenario) >= 8 && scenario[:8] == "reserved" {
				must(f.r.Reserve("ws", "d", human("clerk")))
				f.r.limits.Numbers = s(f).numbers
			}
			switch scenario {
			case "digest":
				digest = digestA
				want = ReasonDigestChanged
			case "unapproved":
				d := draftOf("d", "inv")
				d.Lines[0].Text = "Changed"
				v = must(f.r.Submit("ws", d, human("clerk")))
				digest = v.Digest
				want = ReasonNotApproved
			case "terms":
				c := f.terms.contracts["k1"]
				c.Revision++
				f.terms.put(c)
				want = ReasonReferenceChanged
			case "documents", "reserved_documents":
				f.r.limits.Documents = 0
				want = "limit"
			case "numbers":
				f.r.limits.Numbers = 0
				want = "limit"
			case "width":
				s(f).byKind["inv"].view.Next = ceiling(6) + 1
				want = "limit"
			case "reserved_references":
				f.fields.clear(SubjectCustomer+"cu1", protected.FieldDocumentRecipient)
				want = ReasonRecipientUnresolved
			case "reserved_series":
				s(f).byKind["inv"].view.State = StateProposed
				want = ReasonSeriesPending
			case "foreign":
				workspace = "foreign"
				want = "not_found"
			case "assistant":
				actor = assistant("helper")
				want = ReasonAINotAllowed
			}
			checkUnchanged(t, f, want, func() error { return f.r.CheckIssue(workspace, "d", digest, actor) })
			if scenario == "reserved" {
				must(f.r.Issue("ws", "d", human("issuer")))
				if s(f).numbers != 1 {
					t.Fatal("used another number")
				}
			}
		})
	}
}

func TestExternalAdmissionNoClaimAndTimeRules(t *testing.T) {
	f := standard(t)
	d := draftOf("external", "erp")
	d.Language = "en"
	v := f.ready(t, "ws", d)
	x := External{Number: "EXT-1", IssuedAt: f.clock.read()}
	check := func() error { return f.r.CheckExternal("ws", d.ID, v.Digest, x, human("issuer")) }
	checkUnchanged(t, f, "ok", check)
	checkUnchanged(t, f, "ok", check)
	x.IssuedAt = x.IssuedAt.Add(time.Second)
	checkUnchanged(t, f, ReasonFutureInstant, check)
	x.IssuedAt = f.clock.read().Add(-time.Second)
	checkUnchanged(t, f, ReasonBeforeApproval, check)
	x.IssuedAt = f.clock.read()
	f.clock.advance(time.Hour)
	f.fields.set(SubjectCustomer+"cu1", protected.FieldDocumentRecipient, "later", "later-ref", f.clock.read())
	checkUnchanged(t, f, "ok", check)
	first := must(f.r.RecordExternal("ws", d.ID, x, human("issuer")))
	checkUnchanged(t, f, ReasonAlreadyIssued, check)
	if !reflect.DeepEqual(first, must(f.r.RecordExternal("ws", d.ID, x, human("issuer")))) {
		t.Fatal("external historical replay changed")
	}
	d.ID = "other"
	v = f.ready(t, "ws", d)
	x.IssuedAt = f.clock.read()
	checkUnchanged(t, f, ReasonExternalInUse, check)
}

func TestExternalAdmissionRejectsUnsupportedAndUnavailable(t *testing.T) {
	f := standard(t)
	v := f.ready(t, "ws", draftOf("self", "inv"))
	x := External{Number: "EXT-1", IssuedAt: f.clock.read()}
	checkUnchanged(t, f, ReasonNotERP, func() error { return f.r.CheckExternal("ws", "self", v.Digest, x, human("issuer")) })
	d := draftOf("external", "erp")
	d.Language = "en"
	v = f.ready(t, "ws", d)
	checkUnchanged(t, f, "invalid", func() error { return f.r.CheckExternal("ws", d.ID, v.Digest, External{}, human("issuer")) })
	f.r.limits.Documents = 0
	checkUnchanged(t, f, "limit", func() error { return f.r.CheckExternal("ws", d.ID, v.Digest, x, human("issuer")) })
}

func TestExternalAdmissionRechecksAndCallerBoundaries(t *testing.T) {
	for _, scenario := range []string{"changed_reference", "missing_approval", "foreign", "assistant"} {
		t.Run(scenario, func(t *testing.T) {
			f := standard(t)
			d := draftOf("external", "erp")
			d.Language = "en"
			v := f.ready(t, "ws", d)
			x := External{Number: "EXT-1", IssuedAt: f.clock.read()}
			checkUnchanged(t, f, "ok", func() error { return f.r.CheckExternal("ws", d.ID, v.Digest, x, human("issuer")) })
			workspace, actor, want := "ws", human("issuer"), ReasonReferenceChanged
			switch scenario {
			case "changed_reference":
				f.fields.set(SubjectCustomer+"cu1", protected.FieldDocumentRecipient, "changed", "changed-ref", x.IssuedAt)
			case "missing_approval":
				d.Lines[0].Text = "Changed"
				v = must(f.r.Submit("ws", d, human("clerk")))
				want = ReasonNotApproved
			case "foreign":
				workspace = "foreign"
				want = "not_found"
			case "assistant":
				actor = assistant("helper")
				want = ReasonAINotAllowed
			}
			before := admissionState(t, f)
			checkUnchanged(t, f, want, func() error { return f.r.CheckExternal(workspace, d.ID, v.Digest, x, actor) })
			expect(t, "effect rechecks", second(f.r.RecordExternal(workspace, d.ID, x, actor)), want)
			if before != admissionState(t, f) {
				t.Fatal("refused external effect changed state")
			}
		})
	}
}

func TestAdmissionRealDocumentAndNumberCapacity(t *testing.T) {
	limits := testLimits()
	limits.Documents = 1
	limits.Numbers = 1
	f := open(t, limits)
	f.setup(t, "ws")
	a := f.ready(t, "ws", draftOf("first", "inv"))
	b := f.ready(t, "ws", draftOf("second", "inv"))
	must(f.r.Reserve("ws", "first", human("clerk")))
	checkUnchanged(t, f, "limit", func() error { return f.r.CheckIssue("ws", "second", b.Digest, human("issuer")) })
	checkUnchanged(t, f, "ok", func() error { return f.r.CheckIssue("ws", "first", a.Digest, human("issuer")) })
	must(f.r.Issue("ws", "first", human("issuer")))
	d := draftOf("external", "erp")
	d.Language = "en"
	v := f.ready(t, "ws", d)
	x := External{Number: "EXT-1", IssuedAt: f.clock.read()}
	checkUnchanged(t, f, "limit", func() error { return f.r.CheckExternal("ws", d.ID, v.Digest, x, human("issuer")) })
}
