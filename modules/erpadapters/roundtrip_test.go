package erpadapters

import (
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type simulatedERP struct {
	next  int
	docs  map[string][]erpledger.Row
	state map[string]string
	rev   map[string]int
}

func newSimulatedERP() *simulatedERP {
	return &simulatedERP{docs: map[string][]erpledger.Row{}, state: map[string]string{}, rev: map[string]int{}}
}

func (s *simulatedERP) draft(lines []erpledger.Line) string {
	s.next++
	ref := "SIM-" + strconv.Itoa(s.next)
	var rows []erpledger.Row
	for _, l := range lines {
		rows = append(rows, erpledger.Row{ID: l.ID, Product: "P-" + l.Product, Net: l.Net, Tax: l.Tax, Quantity: l.Quantity})
	}
	rows = append(rows, erpledger.Row{ID: "freight", Product: "FREIGHT", Net: "15", Tax: "0", Quantity: "1"})
	s.docs[ref], s.state[ref], s.rev[ref] = rows, erpledger.StateDraft, 1
	return ref
}

func (s *simulatedERP) bump(ref, state string) int {
	s.state[ref] = state
	s.rev[ref]++
	return s.rev[ref]
}

type ledgerHarness struct {
	t     *testing.T
	l     *erpledger.Registry
	clock *testClock
	sim   *simulatedERP
	obs   int
}

func (h *ledgerHarness) observe(key, ref string, revision int, state, source string) erpledger.ObservationView {
	h.t.Helper()
	h.obs++
	return must(h.l.Observe("w1", erpledger.Observation{ID: "o" + strconv.Itoa(h.obs), Connection: "odoo", Key: key, External: ref, Revision: revision, State: state, Source: source}, erpledger.Actor{ID: "connector", Kind: erpledger.KindSystem}))
}

func newHarness(t *testing.T) *ledgerHarness {
	t.Helper()
	numbers := must(decimal.NewProfile(decimal.ProfileSpec{ID: "rt", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80}))
	clock := &testClock{now: start}
	l := must(erpledger.NewRegistry(numbers, erpledger.Limits{Workspaces: 1, Connections: 2, Mappings: 20, Anchors: 5, Submissions: 10, Lines: 5, Documents: 10, Rows: 10, Observations: 100, Links: 20, Surveys: 5}, clock.read))
	admin := erpledger.Actor{ID: "admin", Kind: erpledger.KindHuman}
	caps := []string{"connect", "validate_mapping", "prepare", "submit_draft", "read_draft", "post", "get_status", "correct", "get_balance"}
	must(l.Connect("w1", erpledger.Connection{ID: "odoo", Entity: "seller-kr", System: "odoo-17", Transport: "api", Capabilities: caps, Currency: "KRW", Scale: 0, Interval: time.Hour, Owner: "erp-ops"}, admin))
	for _, m := range []erpledger.Mapping{{Connection: "odoo", Kind: "customer", Platform: "acme", External: "C-1"}, {Connection: "odoo", Kind: "product", Platform: "compute", External: "P-1"}, {Connection: "odoo", Kind: "tax", Platform: "vat10", External: "T-10"}} {
		must(l.Map("w1", m, admin))
	}
	must(l.Survey("w1", erpledger.Survey{Connection: "odoo", Customer: "acme", Period: "2026-10", Outcome: erpledger.SurveyComplete, Evidence: "survey-1"}, admin))
	return &ledgerHarness{t: t, l: l, clock: clock, sim: newSimulatedERP()}
}

func TestSimulatedRoundTripThroughLedger(t *testing.T) {
	h := newHarness(t)
	biller := erpledger.Actor{ID: "biller", Kind: erpledger.KindHuman}
	lines := []erpledger.Line{{ID: "l1", Source: "usage-1", Product: "compute", TaxCode: "vat10", Quantity: "3", Net: "300", Tax: "30"}}
	sub := erpledger.Submission{Key: "k1", Connection: "odoo", Purpose: erpledger.PurposeInvoice, Customer: "acme", Period: "2026-10", Path: erpledger.PathDirect, Lines: lines}
	must(h.l.Prepare("w1", sub, biller))
	must(h.l.Command("w1", "k1", "submit_draft", biller))
	ref := h.sim.draft(lines)
	h.observe("k1", ref, h.sim.rev[ref], erpledger.StateDraft, erpledger.SourceWebhook)
	must(h.l.Command("w1", "k1", "read_draft", biller))
	must(h.l.Document("w1", erpledger.Document{Connection: "odoo", External: ref, Revision: 1, Rows: h.sim.docs[ref]}, erpledger.Actor{ID: "connector", Kind: erpledger.KindSystem}))
	must(h.l.Link("w1", erpledger.Link{Connection: "odoo", External: ref, Row: "l1", Key: "k1", Line: "l1", Net: "300", Tax: "30", Quantity: "3"}, biller))
	rec := must(h.l.Reconcile("w1", "k1"))
	if !rec.Matched || rec.Documents[0].Total.Net != "315" || rec.Documents[0].Attributed.Net != "300" {
		t.Fatalf("amount recovery %+v", rec)
	}
	must(h.l.Command("w1", "k1", "post", biller))
	h.observe("", ref, h.sim.bump(ref, erpledger.StatePosted), erpledger.StatePosted, erpledger.SourceWebhook)
	if v := must(h.l.SubmissionOf("w1", "k1")); v.State != erpledger.StatePosted || v.External != ref {
		t.Fatalf("posted %+v", v)
	}
	if final := must(h.l.DocumentOf("w1", "odoo", ref)); final.Total.Net != "315" || len(final.Links) != 1 {
		t.Fatalf("final document %+v", final)
	}
	must(h.l.Command("w1", "k1", "correct", biller))
	corrected := append([]erpledger.Row(nil), h.sim.docs[ref]...)
	corrected[0].Net = "280"
	view := must(h.l.Document("w1", erpledger.Document{Connection: "odoo", External: ref, Revision: 2, Rows: corrected}, erpledger.Actor{ID: "connector", Kind: erpledger.KindSystem}))
	if !view.Links[0].Stale {
		t.Fatalf("correction not stale %+v", view.Links)
	}
	if rec := must(h.l.Reconcile("w1", "k1")); rec.Matched {
		t.Fatalf("matched after correction %+v", rec)
	}
}

func TestSimulatedTimeoutAndManualEdit(t *testing.T) {
	h := newHarness(t)
	biller := erpledger.Actor{ID: "biller", Kind: erpledger.KindHuman}
	lines := []erpledger.Line{{ID: "l1", Source: "usage-2", Product: "compute", TaxCode: "vat10", Quantity: "1", Net: "100", Tax: "10"}}
	must(h.l.Prepare("w1", erpledger.Submission{Key: "k2", Connection: "odoo", Purpose: erpledger.PurposeInvoice, Customer: "acme", Period: "2026-10", Path: erpledger.PathDirect, Lines: lines}, biller))
	must(h.l.Command("w1", "k2", "submit_draft", biller))
	ref := h.sim.draft(lines)
	h.clock.advance(time.Hour)
	must(h.l.Checked("w1", "k2", "timeout", erpledger.Actor{ID: "poller", Kind: erpledger.KindSystem}))
	if v := must(h.l.SubmissionOf("w1", "k2")); v.State != erpledger.StateUnknown {
		t.Fatalf("timeout %+v", v)
	}
	again := lines
	if kindLedger(second(h.l.Prepare("w1", erpledger.Submission{Key: "k3", Connection: "odoo", Purpose: erpledger.PurposeInvoice, Customer: "acme", Period: "2026-10", Path: erpledger.PathDirect, Lines: again}, biller))) != erpledger.ReasonAlreadySubmitted {
		t.Fatal("resend allowed while unknown")
	}
	h.observe("k2", ref, h.sim.rev[ref], erpledger.StateDraft, erpledger.SourcePoll)
	h.observe("", ref, h.sim.bump(ref, erpledger.StatePosted), erpledger.StatePosted, erpledger.SourcePoll)
	edit := h.observe("", ref, h.sim.bump(ref, erpledger.StateRejected), erpledger.StateRejected, erpledger.SourceWebhook)
	if edit.Outcome != "diverged" || !must(h.l.SubmissionOf("w1", "k2")).Diverged {
		t.Fatalf("manual edit %+v", edit)
	}
}

func TestLedgerAcceptsUnadmittedCapabilities(t *testing.T) {
	f := newFixture(t)
	numbers := must(decimal.NewProfile(decimal.ProfileSpec{ID: "gap", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80}))
	clock := &testClock{now: start}
	l := must(erpledger.NewRegistry(numbers, erpledger.Limits{Workspaces: 1, Connections: 1, Mappings: 1, Anchors: 1, Submissions: 1, Lines: 1, Documents: 1, Rows: 1, Observations: 1, Links: 1, Surveys: 1}, clock.read))
	caps := []string{"post", "correct"}
	if gaps := must(f.r.Admits("w1", "a1", LevelSandbox, caps)); len(gaps) != 2 {
		t.Fatalf("adapter admits declared commands %v", gaps)
	}
	_, err := l.Connect("w1", erpledger.Connection{ID: "c", Entity: "e", System: "odoo-17", Transport: "api", Capabilities: caps, Currency: "KRW", Interval: time.Hour, Owner: "o"}, erpledger.Actor{ID: "admin", Kind: erpledger.KindHuman})
	if err != nil {
		t.Fatalf("the ledger now refuses unadmitted capabilities (%v); close the adapter binding residual in docs/features/HB-227", err)
	}
}

func kindLedger(err error) string {
	var refusal erpledger.Refusal
	if ok := asRefusal(err, &refusal); ok {
		return refusal.Reason
	}
	if err == nil {
		return "ok"
	}
	return err.Error()
}

func asRefusal(err error, out *erpledger.Refusal) bool {
	r, ok := err.(erpledger.Refusal)
	if ok {
		*out = r
	}
	return ok
}
