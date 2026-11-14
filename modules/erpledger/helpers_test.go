package erpledger

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testNumbers(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "erpledger-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Connections: 4, Mappings: 20, Anchors: 6, Submissions: 12, Lines: 4, Documents: 6, Rows: 4, Observations: 60, Links: 6, Surveys: 6}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func kind(err error) string {
	var refusal Refusal
	switch {
	case err == nil:
		return "ok"
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "other: " + err.Error()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func second[T any](_ T, err error) error { return err }

func expect(t *testing.T, label string, err error, want string) {
	t.Helper()
	if got := kind(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type fixture struct {
	r     *Registry
	clock *testClock
}

var allCapabilities = []string{CommandConnect, CommandMapping, CommandPrepare, CommandSubmit, CommandReadDraft, CommandPost, CommandStatus, CommandCorrect, CommandBalance}

func apiConnection(id string) Connection {
	return Connection{ID: id, Entity: "seller-kr", System: "odoo-17", Transport: TransportAPI, Capabilities: allCapabilities, Currency: "KRW", Scale: 0, Interval: time.Hour, Owner: "erp-ops"}
}

func fileConnection(id string) Connection {
	return Connection{ID: id, Entity: "seller-kr", System: "sap-s4", Transport: TransportFile, Capabilities: []string{CommandPrepare, CommandMapping, CommandSubmit}, Currency: "KRW", Scale: 0, Interval: time.Hour, Owner: "erp-ops", Template: "standard_xlsx"}
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testNumbers(t), testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{r: r, clock: clock}
	f.connect(t, apiConnection("api"))
	return f
}

func (f fixture) connect(t *testing.T, c Connection) {
	t.Helper()
	must(f.r.Connect("w1", c, human("admin")))
	for _, m := range []Mapping{{c.ID, "customer", "acme", "C-100"}, {c.ID, "product", "compute", "P-1"}, {c.ID, "product", "storage", "P-2"}, {c.ID, "tax", "vat10", "T-10"}} {
		must(f.r.Map("w1", m, human("admin")))
	}
	must(f.r.Survey("w1", Survey{Connection: c.ID, Customer: "acme", Period: "2026-10", Outcome: SurveyComplete, Evidence: "survey-" + c.ID}, human("admin")))
}

func line(id, source, net string) Line {
	return Line{ID: id, Source: source, Product: "compute", TaxCode: "vat10", Quantity: "1", Net: net, Tax: "0"}
}

func invoice(key, connection string, lines ...Line) Submission {
	return Submission{Key: key, Connection: connection, Purpose: PurposeInvoice, Customer: "acme", Period: "2026-10", Path: PathDirect, Lines: lines}
}

func (f fixture) send(t *testing.T, sub Submission) SubmissionView {
	t.Helper()
	must(f.r.Prepare("w1", sub, human("biller")))
	return must(f.r.Command("w1", sub.Key, CommandSubmit, human("biller")))
}

var observationCount int

func (f fixture) observe(t *testing.T, connection, key, external string, revision int, state, source string) ObservationView {
	t.Helper()
	observationCount++
	return must(f.r.Observe("w1", Observation{ID: "o" + strconv.Itoa(observationCount), Connection: connection, Key: key, External: external, Revision: revision, State: state, Source: source}, system("connector")))
}
