package issuance

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

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

var start = time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC)

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func reason(err error) string {
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
	if got := reason(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type fields struct {
	mu      sync.Mutex
	entries map[string][]protected.Resolution
	fail    bool
}

func (f *fields) set(subject, field, change, ref string, effective time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := subject + "|" + field
	f.entries[key] = append(f.entries[key], protected.Resolution{Change: change, Ref: protected.Ref{ID: ref, Fingerprint: "fp-" + ref}, Effective: effective})
}

func (f *fields) clear(subject, field string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.entries, subject+"|"+field)
}

func (f *fields) Resolve(workspace, subject, field string, at time.Time) (protected.Resolution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return protected.Resolution{}, errors.New("unavailable")
	}
	var found *protected.Resolution
	for index, entry := range f.entries[subject+"|"+field] {
		if !entry.Effective.After(at) {
			found = &f.entries[subject+"|"+field][index]
		}
	}
	if found == nil {
		return protected.Resolution{}, protected.ErrNotFound
	}
	return *found, nil
}

type terms struct {
	mu        sync.Mutex
	contracts map[string]receivables.Coverage
	err       error
	skew      int
}

func (f *terms) Cover(workspace, contract string, at time.Time) (receivables.Coverage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return receivables.Coverage{}, f.err
	}
	c, ok := f.contracts[contract]
	if !ok {
		return receivables.Coverage{}, receivables.Refusal{Reason: receivables.ReasonTermMissing}
	}
	if location, err := time.LoadLocation(c.Zone); err == nil {
		c.Date, c.Due = dates(c.NetDays, at, location)
	}
	switch f.skew {
	case 1:
		c.Date = "2000-01-01"
	case 2:
		c.Due = "2000-01-01"
	}
	return c, nil
}

func (f *terms) put(c receivables.Coverage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.contracts[c.Contract] = c
}

type details struct {
	mu    sync.Mutex
	facts map[string]DetailFacts
}

func (f *details) Detail(workspace, artifact string) (DetailFacts, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	facts, ok := f.facts[workspace+"|"+artifact]
	if !ok {
		return DetailFacts{}, errors.New("unknown")
	}
	return facts, nil
}

type renderer struct {
	mu    sync.Mutex
	fn    func(Snapshot) ([]byte, error)
	calls int
	seen  []Snapshot
}

func (f *renderer) Render(v Snapshot) ([]byte, error) {
	f.mu.Lock()
	fn := f.fn
	f.calls++
	f.seen = append(f.seen, v)
	f.mu.Unlock()
	return fn(v)
}

func (f *renderer) use(fn func(Snapshot) ([]byte, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fn = fn
}

func stream(dict, body string) string {
	return fmt.Sprintf("<< %s /Length %d >>\nstream\n%s\nendstream", dict, len(body), body)
}

func build(extra string, rest ...string) []byte {
	return assemble("", append([]string{"<< /Type /Catalog /Pages 2 0 R " + extra + " >>", "<< /Type /Pages /Kids [] /Count 0 >>"}, rest...)...)
}

func assemble(trailer string, objects ...string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects))
	for index, o := range objects {
		offsets[index] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", index+1, o)
	}
	x := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	if trailer == "" {
		trailer = fmt.Sprintf("<< /Size %d /Root 1 0 R >>", len(objects)+1)
	}
	fmt.Fprintf(&b, "trailer\n%s\nstartxref\n%d\n%%%%EOF\n", trailer, x)
	return b.Bytes()
}

func page(content string) []byte {
	return build("", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>", stream("/Filter /FlateDecode", content))
}

func good(v Snapshot) ([]byte, error) {
	return page("BT (" + v.Number + " " + v.Digest + ") Tj ET"), nil
}

type fixture struct {
	r        *Registry
	clock    *testClock
	fields   *fields
	terms    *terms
	details  *details
	renderer *renderer
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Kinds: 8, Drafts: 40, Documents: 30, Numbers: 60, Attempts: 6, Downloads: 3, Lease: 10 * time.Minute}
}

const digestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func open(t testing.TB, limits Limits) fixture {
	t.Helper()
	clock := &testClock{now: start}
	f := fixture{clock: clock, fields: &fields{entries: map[string][]protected.Resolution{}}, terms: &terms{contracts: map[string]receivables.Coverage{}},
		details: &details{facts: map[string]DetailFacts{}}, renderer: &renderer{fn: good}}
	r, err := NewRegistry(Config{Fields: f.fields, Terms: f.terms, Details: f.details, Renderer: f.renderer, Zones: time.LoadLocation}, limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f.r = r
	return f
}

var kinds = []Kind{
	{ID: "inv", Class: ClassSelf, Region: "KR", Tax: "kr_vat", Statutory: true, Zone: "Asia/Seoul", Languages: []string{"ko", "en", "ar"}},
	{ID: "crn", Class: ClassSelf, Region: "KR", Tax: "kr_vat", Statutory: true, Credit: true, Zone: "Asia/Seoul", Languages: []string{"ko", "en"}},
	{ID: "int", Class: ClassInternal, Region: "KR", Tax: "none", Zone: "Asia/Seoul", Languages: []string{"en"}},
	{ID: "erp", Class: ClassERP, Region: "US", Tax: "us_sales", Zone: "America/New_York", Languages: []string{"en"}},
	{ID: "seq", Class: ClassSelf, Region: "AE", Tax: "ae_vat", Sequential: true, Zone: "Asia/Dubai", Languages: []string{"ar", "en"}},
}

var prefixes = map[string]string{"inv": "INV-", "crn": "CRN-", "int": "INT-", "seq": "SEQ-"}

func standard(t testing.TB) fixture {
	t.Helper()
	f := open(t, testLimits())
	f.setup(t, "ws")
	return f
}

func (f fixture) setup(t testing.TB, workspace string) {
	t.Helper()
	before := f.clock.read().Add(-time.Hour)
	for _, k := range kinds {
		must(f.r.Define(workspace, k, human("definer")))
		must(f.r.ApproveKind(workspace, k.ID, human("approver")))
		if k.Statutory {
			must(f.r.VerifyTax(workspace, k.ID, "tax-review-"+k.ID, human("taxer")))
		}
		if prefix, ok := prefixes[k.ID]; ok {
			must(f.r.ProposeSeries(workspace, Series{ID: "s-" + k.ID, Kind: k.ID, Prefix: prefix, Width: 6, First: 1}, human("definer")))
			must(f.r.ApproveSeries(workspace, "s-"+k.ID, human("approver")))
		}
		f.fields.set(SubjectKind+k.ID, protected.FieldDocumentTemplate, "tpl-"+k.ID, "template-"+k.ID, before)
	}
	for _, s := range []string{"en1", "en2"} {
		f.fields.set(SubjectEntity+s, protected.FieldPaymentAccount, "acct-"+s, "account-"+s, before)
		f.fields.set(SubjectEntity+s, protected.FieldTaxIdentifier, "etax-"+s, "entity-tax-"+s, before)
	}
	for _, c := range []string{"cu1", "cu2"} {
		f.fields.set(SubjectCustomer+c, protected.FieldDocumentRecipient, "rcpt-"+c, "recipient-"+c, before)
		f.fields.set(SubjectCustomer+c, protected.FieldTaxIdentifier, "ctax-"+c, "customer-tax-"+c, before)
	}
	f.terms.put(receivables.Coverage{Contract: "k1", Binding: "b1", Term: "net30", Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"})
	f.terms.put(receivables.Coverage{Contract: "k2", Binding: "b2", Term: "net45", Revision: 2, NetDays: 45, Mode: receivables.ModeCalendar, Zone: "America/New_York"})
}

func draftOf(id, kindID string, lines ...Line) Draft {
	if len(lines) == 0 {
		lines = []Line{{Text: "Cloud usage", Amount: "100"}, {Text: "VAT", Amount: "10", Tax: true}}
	}
	total := sumOf(lines, 2)
	return Draft{ID: id, Kind: kindID, Customer: "cu1", Entity: "en1", Contract: "k1", Currency: "KRW", Scale: 2, Language: "ko", Lines: lines, Total: total}
}

func sumOf(lines []Line, scale int) string {
	d := Draft{Lines: lines, Scale: scale}
	total, _ := sums(d)
	return render(total.String(), scale)
}

func render(digits string, scale int) string {
	for len(digits) <= scale {
		digits = "0" + digits
	}
	whole, fraction := digits[:len(digits)-scale], strings.TrimRight(digits[len(digits)-scale:], "0")
	if fraction == "" {
		return whole
	}
	return whole + "." + fraction
}

func (f fixture) ready(t testing.TB, workspace string, d Draft) DraftView {
	t.Helper()
	v := must(f.r.Submit(workspace, d, human("clerk")))
	return must(f.r.Approve(workspace, d.ID, v.Digest, human("lead")))
}

func (f fixture) issued(t testing.TB, d Draft) DocumentView {
	t.Helper()
	f.ready(t, "ws", d)
	return must(f.r.Issue("ws", d.ID, human("issuer")))
}

var errFake = errors.New("renderer unavailable")

func receivablesProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "issuance-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}
