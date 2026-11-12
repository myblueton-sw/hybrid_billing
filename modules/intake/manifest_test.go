package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestRefusalsRepeatNothingTheCallerSent(t *testing.T) {
	h := newHarness(t, nil)
	h.register(base("a1", "d1", 1))
	secret := "customer-text"
	spec := base("a1", "d1", 2)
	spec.Location = secret
	_, _, conflict := h.r.Register(spec)
	spec.ID = secret + " with a space"
	_, _, invalid := h.r.Register(spec)
	_, missing := h.r.Stage("ws1", secret, 1, []Record{rec(secret, "1", "USD")})
	_, _, absent := h.r.Seal(secret, "a1")
	wrong := h.r.Verify("ws1", "a1", digestOf(secret), secret, secret)
	_, unpublished := h.r.Publish(Subject{WorkspaceID: "ws1", Provider: secret, Scope: secret, Period: secret}, 0)
	for index, pair := range []struct{ got, want error }{{conflict, ErrConflict}, {invalid, ErrInvalid}, {missing, ErrNotFound}, {absent, ErrNotFound}, {wrong, ErrRejected}, {unpublished, ErrNotFound}} {
		if pair.got != pair.want || strings.Contains(pair.got.Error(), "customer") {
			t.Fatalf("%d: %v", index, pair.got)
		}
	}
}

func (h *harness) blocked(at Subject, expected uint64, total int, want ...Blocker) {
	h.t.Helper()
	before := h.r.dump()
	out, err := h.r.Publish(at, expected)
	if err != ErrRejected || out.Number != 0 || out.Written || out.BlockerTotal != total || !reflect.DeepEqual(out.Blockers, want) {
		h.t.Fatalf("publish: %+v %v", out, err)
	}
	h.unchanged(before, "blocked publication")
	status := h.status(at)
	if status.Differs != Blocked || status.BlockerTotal != total || !reflect.DeepEqual(status.Blockers, want) {
		h.t.Fatalf("status: %+v", status)
	}
}

func (h *harness) written(at Subject, expected uint64, want string) {
	h.t.Helper()
	out := h.publish(at, expected)
	if out.Number != expected+1 || !out.Written || out.Blockers != nil {
		h.t.Fatalf("publish: %+v", out)
	}
	if got := brief(h.all(at, out.Number)); got != want {
		h.t.Fatalf("manifest %d:\n got %s\nwant %s", out.Number, got, want)
	}
	if status := h.status(at); status.Differs != Same || status.Latest != out.Number {
		h.t.Fatalf("status after publication: %+v", status)
	}
}

func (h *harness) manifest(at Subject, number uint64) ManifestView {
	h.t.Helper()
	view, err := h.r.Manifest(at, number)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func TestLaterVersionReplacesTheEarlierAsAWhole(t *testing.T) {
	h := newHarness(t, nil)
	h.sealRows("a1", "d1", 1, []Record{rec("k1", "10", "USD"), rec("k2", "20", "USD")}, []Record{rec("k3", "30", "USD")})
	if status := h.status(october); status.Differs != DiffersContent || status.Latest != 0 {
		t.Fatalf("%+v", status)
	}
	h.clock.advance(time.Minute)
	h.written(october, 0, "event//k1=10USD@a1 event//k2=20USD@a1 event//k3=30USD@a1")
	first := h.manifest(october, 1)
	if first.Records != 3 || !reflect.DeepEqual(first.Sums, map[string]string{"USD": "60"}) || !first.At.Equal(start.Add(time.Minute)) || first.Number != 1 || first.Subject != october {
		t.Fatalf("%+v", first)
	}
	h.sealRows("a2", "d1", 2, []Record{rec("k1", "10", "USD"), rec("k3", "35", "USD"), rec("k4", "5.5", "EUR")})
	if status := h.status(october); status.Differs != DiffersContent || status.Latest != 1 {
		t.Fatalf("%+v", status)
	}
	h.written(october, 1, "event//k1=10USD@a2 event//k3=35USD@a2 event//k4=5.5EUR@a2")
	second := h.manifest(october, 2)
	if !reflect.DeepEqual(second.Sums, map[string]string{"USD": "45", "EUR": "5.5"}) || second.Content == first.Content {
		t.Fatalf("%+v", second)
	}
	if again := h.manifest(october, 1); !reflect.DeepEqual(again, first) {
		t.Fatal("a published manifest changed")
	}
	if err := h.r.Retire("ws1", "a2", Reissued); err != nil {
		t.Fatal(err)
	}
	if status := h.status(october); status.Differs != DiffersContent || !reflect.DeepEqual(status.Behind, Listed{Total: 1, Items: []string{"d1"}}) {
		t.Fatalf("%+v", status)
	}
	h.written(october, 2, "event//k1=10USD@a1 event//k2=20USD@a1 event//k3=30USD@a1")
	third := h.manifest(october, 3)
	if third.Content != first.Content || third.Attribution != first.Attribution || third.Behind.Total != 1 || first.Behind.Total != 0 {
		t.Fatalf("%+v", third)
	}
	h.sealRows("a0", "d1", 1, []Record{rec("k1", "10", "USD"), rec("k2", "20", "USD"), rec("k3", "30", "USD")})
	if out := h.publish(october, 3); out.Number != 3 || out.Written {
		t.Fatalf("%+v", out)
	}
}

func TestLocatorRowsShiftAndStayInsideTheirDocument(t *testing.T) {
	h := newHarness(t, nil)
	locator := func(id, document string, revision uint64, rows ...Record) {
		s := original(t, id, document, revision, rows)
		s.Kind = LocatorKeys
		h.seal(s, rows)
	}
	same := Record{Key: "r2", Digest: digestOf("same"), Amount: "2", Currency: "USD"}
	twin := Record{Key: "r3", Digest: digestOf("same"), Amount: "2", Currency: "USD"}
	locator("a1", "L1", 1, rec("r1", "1", "USD"), same, twin)
	locator("b1", "L2", 1, rec("r1", "7", "USD"))
	h.sealRows("c1", "E1", 1, []Record{rec("r1", "100", "USD")})
	h.written(october, 0, "event//r1=100USD@c1 locator/L1/r1=1USD@a1 locator/L1/r2=2USD@a1 locator/L1/r3=2USD@a1 locator/L2/r1=7USD@b1")
	locator("a2", "L1", 2, rec("r1", "2", "USD"), rec("r2", "2", "USD"))
	h.written(october, 1, "event//r1=100USD@c1 locator/L1/r1=2USD@a2 locator/L1/r2=2USD@a2 locator/L2/r1=7USD@b1")
	view := h.manifest(october, 2)
	if !reflect.DeepEqual(view.Sums, map[string]string{"USD": "111"}) || view.Records != 4 {
		t.Fatalf("%+v", view)
	}
	records := h.all(october, 2)
	var content, attribution []byte
	for _, record := range records {
		fields := []string{string(record.Kind), record.Document, record.Key, record.Amount, record.Currency, record.Digest}
		content = append(content, secondEncoding(fields...)...)
		attribution = append(attribution, secondEncoding(append(fields, record.Artifact)...)...)
	}
	c, a := sha256.Sum256(content), sha256.Sum256(attribution)
	if view.Content != hex.EncodeToString(c[:]) || view.Attribution != hex.EncodeToString(a[:]) {
		t.Fatal("manifest digests do not follow the stated encoding")
	}
	if records[0].Document != "" || records[1].Document != "L1" || records[0].Digest != rec("r1", "100", "USD").Digest {
		t.Fatalf("%+v", records)
	}
}

func TestTwoReceiptsOfOneVersion(t *testing.T) {
	h := newHarness(t, nil)
	rows := []Record{rec("k1", "10", "USD"), rec("k2", "20", "USD")}
	h.sealRows("a1", "d1", 1, rows)
	h.sealRows("a2", "d1", 1, []Record{rows[1]}, []Record{rows[0]})
	h.written(october, 0, "event//k1=10USD@a1 event//k2=20USD@a1")
	if err := h.r.Retire("ws1", "a1", Reissued); err != nil {
		t.Fatal(err)
	}
	if status := h.status(october); status.Differs != DiffersAttribution || status.Behind.Total != 0 {
		t.Fatalf("%+v", status)
	}
	h.written(october, 1, "event//k1=10USD@a2 event//k2=20USD@a2")
	one, two := h.manifest(october, 1), h.manifest(october, 2)
	if one.Content != two.Content || one.Attribution == two.Attribution {
		t.Fatal("a change of attribution must keep the content digest")
	}
	h.sealRows("a3", "d1", 1, []Record{rec("k1", "10", "USD"), rec("k2", "20.0001", "USD")})
	h.blocked(october, 2, 1, Blocker{What: InConflict, Document: "d1"})
	h.seal(base("a4", "d2", 1))
	h.sealRows("a5", "d2", 1, []Record{rec("z1", "1", "USD")})
	h.blocked(october, 2, 2, Blocker{What: InConflict, Document: "d1"}, Blocker{What: InConflict, Document: "d2"})
	if err := h.r.Retire("ws1", "a3", WrongSource); err != nil {
		t.Fatal(err)
	}
	h.blocked(october, 2, 1, Blocker{What: InConflict, Document: "d2"})
	if err := h.r.Retire("ws1", "a5", WrongSource); err != nil {
		t.Fatal(err)
	}
	if out := h.publish(october, 0); out.Number != 2 || out.Written {
		t.Fatalf("%+v", out)
	}
	h.seal(base("a6", "d2", 1))
	if out := h.publish(october, 2); out.Number != 2 || out.Written {
		t.Fatalf("%+v", out)
	}
}

func TestRowsAreAttributedToTheEarliestSealedReceipt(t *testing.T) {
	h := newHarness(t, nil)
	rows := []Record{rec("k1", "10", "USD")}
	first, second := original(t, "a1", "d1", 1, rows), original(t, "a2", "d1", 1, rows)
	h.register(first)
	h.register(second)
	h.stage(first, rows)
	h.seal(second, rows)
	h.seal(first, rows)
	h.written(october, 0, "event//k1=10USD@a2")
	if h.view("a2").Sequence != 1 || h.view("a1").Sequence != 2 {
		t.Fatal("sequence")
	}
}

func TestScannedOriginalIsUnreadableUntilItsExtractionIsSealed(t *testing.T) {
	h := newHarness(t, nil)
	h.sealRows("a1", "d1", 1, []Record{rec("k1", "10", "USD")})
	h.written(october, 0, "event//k1=10USD@a1")
	h.seal(scan("o2", "d1", 2))
	h.blocked(october, 1, 1, Blocker{What: Unreadable, Document: "d1"})
	good := []Record{rec("k1", "12.50", "USD")}
	x := derived(t, "x2", "d1", 2, "o2", good)
	h.register(x)
	h.stage(x, good)
	h.blocked(october, 1, 1, Blocker{What: Unreadable, Document: "d1"})
	h.seal(x, good)
	h.written(october, 1, "event//k1=12.5USD@x2")
	misread := []Record{rec("k1", "125.0", "USD")}
	h.seal(derived(t, "y2", "d1", 2, "o2", misread), misread)
	h.blocked(october, 2, 1, Blocker{What: InConflict, Document: "d1"})
	if err := h.r.Retire("ws1", "y2", VerificationWithdrawn); err != nil {
		t.Fatal(err)
	}
	h.seal(derived(t, "z2", "d1", 2, "o2", good), good)
	if out := h.publish(october, 2); out.Number != 2 || out.Written {
		t.Fatalf("%+v", out)
	}
	if err := h.r.Retire("ws1", "o2", WrongSource); err != nil {
		t.Fatal(err)
	}
	if h.r.workspaces["ws1"].records != 3 {
		t.Fatal(h.r.workspaces["ws1"].records)
	}
	if status := h.status(october); status.Differs != DiffersContent || !reflect.DeepEqual(status.Behind, Listed{Total: 1, Items: []string{"d1"}}) {
		t.Fatalf("%+v", status)
	}
	h.written(october, 2, "event//k1=10USD@a1")
	if err := h.r.Retire("ws1", "x2", VerificationWithdrawn); err != nil {
		t.Fatal(err)
	}
	if h.r.workspaces["ws1"].records != 2 {
		t.Fatal(h.r.workspaces["ws1"].records)
	}
	late := derived(t, "w2", "d1", 2, "o2", good)
	_, _, err := h.r.Register(late)
	wantErr(t, err, ErrRejected, "extraction of a retired original")
}

func TestMixedArtifactsAtOneRevision(t *testing.T) {
	rows := []Record{rec("k1", "10", "USD")}
	h := newHarness(t, nil)
	h.seal(scan("o1", "d1", 1))
	h.seal(base("e1", "d1", 1))
	h.sealRows("b1", "d2", 1, rows)
	h.written(october, 0, "event//k1=10USD@b1")
	h.seal(derived(t, "x1", "d1", 1, "o1", rows), rows)
	h.blocked(october, 1, 1, Blocker{What: InConflict, Document: "d1"})
	if err := h.r.Retire("ws1", "e1", WrongSource); err != nil {
		t.Fatal(err)
	}
	if status := h.status(october); status.Differs != Same {
		t.Fatalf("%+v", status)
	}
	h.sealRows("a1", "d1", 1, rows)
	if err := h.r.Retire("ws1", "b1", WrongSource); err != nil {
		t.Fatal(err)
	}
	h.written(october, 1, "event//k1=10USD@x1")
	if err := h.r.Retire("ws1", "x1", VerificationWithdrawn); err != nil {
		t.Fatal(err)
	}
	h.written(october, 2, "event//k1=10USD@a1")
	if err := h.r.Retire("ws1", "a1", WrongSource); err != nil {
		t.Fatal(err)
	}
	h.blocked(october, 3, 1, Blocker{What: Unreadable, Document: "d1"})
	if err := h.r.Retire("ws1", "o1", WrongSource); err != nil {
		t.Fatal(err)
	}
	h.written(october, 3, "")
	view := h.manifest(october, 4)
	if view.Records != 0 || len(view.Sums) != 0 || !reflect.DeepEqual(view.Behind, Listed{Total: 2, Items: []string{"d1", "d2"}}) {
		t.Fatalf("%+v", view)
	}
	if out := h.publish(october, 4); out.Number != 4 || out.Written {
		t.Fatalf("%+v", out)
	}
}

func TestEventKeysAcrossDocuments(t *testing.T) {
	h := newHarness(t, nil)
	h.sealRows("a1", "d1", 1, []Record{rec("k1", "10", "USD"), rec("k3", "1", "USD")})
	h.sealRows("b1", "d2", 1, []Record{rec("k1", "10", "USD"), rec("k2", "5", "USD")})
	h.written(october, 0, "event//k1=10USD@a1 event//k2=5USD@b1 event//k3=1USD@a1")
	if view := h.manifest(october, 1); !reflect.DeepEqual(view.Sums, map[string]string{"USD": "16"}) {
		t.Fatalf("%+v", view)
	}
	if err := h.r.Retire("ws1", "a1", Reissued); err != nil {
		t.Fatal(err)
	}
	h.sealRows("a2", "d1", 1, []Record{rec("k1", "10", "USD"), rec("k3", "1", "USD")})
	h.written(october, 1, "event//k1=10USD@b1 event//k2=5USD@b1 event//k3=1USD@a2")
	if one, two := h.manifest(october, 1), h.manifest(october, 2); one.Content != two.Content || one.Attribution == two.Attribution {
		t.Fatal("attribution")
	}
	otherDigest := rec("k2", "5", "USD")
	otherDigest.Digest = digestOf("other")
	otherAmount, otherCurrency := rec("k2", "5", "USD"), rec("k2", "5", "USD")
	otherAmount.Amount, otherCurrency.Currency = "5.01", "EUR"
	for index, differing := range []Record{otherAmount, otherCurrency, otherDigest} {
		id := fmt.Sprintf("c%d", index)
		h.sealRows(id, "d3", uint64(index+1), []Record{differing, rec("k1", "10", "USD")})
		h.blocked(october, 2, 1, Blocker{What: KeyConflict, Document: "d3", Key: "k2", HeldBy: "d2"})
	}
	h.sealRows("e1", "d4", 1, []Record{rec("k2", "6", "USD"), rec("k3", "2", "USD"), rec("k1", "11", "USD"), rec("k0", "11", "USD")})
	h.blocked(october, 2, 3, Blocker{What: KeyConflict, Document: "d4", Key: "k1", HeldBy: "d2"}, Blocker{What: KeyConflict, Document: "d3", Key: "k2", HeldBy: "d2"}, Blocker{What: KeyConflict, Document: "d4", Key: "k3", HeldBy: "d1"})
	other := original(t, "f1", "d5", 1, []Record{rec("k1", "99", "USD")})
	other.Subject = november
	h.seal(other, []Record{rec("k1", "99", "USD")})
	h.written(november, 0, "event//k1=99USD@f1")
	foreign := original(t, "f1", "d1", 1, []Record{rec("k1", "77", "USD")})
	foreign.Subject.WorkspaceID = "ws2"
	h.seal(foreign, []Record{rec("k1", "77", "USD")})
	elsewhere := october
	elsewhere.WorkspaceID = "ws2"
	h.written(elsewhere, 0, "event//k1=77USD@f1")
	if view, err := h.r.Artifact("ws2", "f1"); err != nil || view.Sequence != 1 {
		t.Fatal("the sealing sequence of one workspace shows the seals of another")
	}
	if _, err := h.r.Manifest(november, 2); err != ErrNotFound {
		t.Fatal(err)
	}
}

func TestSubjectSumOutsideTheProfileBlocksPublication(t *testing.T) {
	small, err := decimal.NewProfile(decimal.ProfileSpec{ID: "small", Revision: "1", InputIntegerDigits: 3, InputScale: 2, ResultIntegerDigits: 3, ResultScale: 2, IntermediateDigits: 10})
	if err != nil {
		t.Fatal(err)
	}
	clock := &testClock{at: start}
	r, err := NewRegistry(testLimits(), small, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, r: r, clock: clock}
	h.sealRows("a1", "d1", 1, []Record{rec("k1", "600", "USD"), rec("k2", "1", "EUR")})
	h.sealRows("b1", "d2", 1, []Record{rec("k3", "399.99", "USD"), rec("k4", "-600", "KRW")})
	h.written(october, 0, "event//k1=600USD@a1 event//k2=1EUR@a1 event//k3=399.99USD@b1 event//k4=-600KRW@b1")
	h.sealRows("c1", "d3", 1, []Record{rec("k5", "0.01", "USD"), rec("k6", "-400", "KRW")})
	h.blocked(october, 1, 2, Blocker{What: SumOutside, Currency: "KRW"}, Blocker{What: SumOutside, Currency: "USD"})
	if err := h.r.Retire("ws1", "c1", WrongSource); err != nil {
		t.Fatal(err)
	}
	if status := h.status(october); status.Differs != Same || status.BlockerTotal != 0 {
		t.Fatalf("%+v", status)
	}
}

func TestBlockersAreBoundedAndCounted(t *testing.T) {
	h := newHarness(t, nil)
	for index := 0; index < 5; index++ {
		document := fmt.Sprintf("d%d", index)
		h.sealRows(fmt.Sprintf("a%d", index), document, 1, []Record{rec(fmt.Sprintf("k%d", index), "1", "USD")})
		h.sealRows(fmt.Sprintf("b%d", index), document, 1, []Record{rec(fmt.Sprintf("k%d", index), "2", "USD")})
	}
	h.seal(scan("o1", "d9", 1))
	h.blocked(october, 0, 6, Blocker{What: InConflict, Document: "d0"}, Blocker{What: InConflict, Document: "d1"}, Blocker{What: InConflict, Document: "d2"})
	wide := newHarness(t, func(l *Limits) { l.Listed = 6 })
	wide.seal(scan("o1", "d0", 1))
	wide.sealRows("a1", "d1", 1, []Record{rec("k1", "1", "USD")})
	wide.sealRows("b1", "d1", 1, []Record{rec("k1", "2", "USD")})
	wide.sealRows("c1", "d2", 1, []Record{rec("k7", "1", "USD")})
	wide.sealRows("e1", "d3", 1, []Record{rec("k7", "2", "USD")})
	wide.blocked(october, 0, 3, Blocker{What: InConflict, Document: "d1"}, Blocker{What: KeyConflict, Document: "d3", Key: "k7", HeldBy: "d2"}, Blocker{What: Unreadable, Document: "d0"})
}

func TestBehindDocumentsAreReported(t *testing.T) {
	rows := []Record{rec("k1", "10", "USD")}
	later := []Record{rec("k1", "11", "USD")}
	steps := []struct {
		name string
		make func(h *harness)
	}{
		{"rejected", func(h *harness) {
			s := original(t, "a2", "d1", 2, later)
			s.Bodies = Failed
			h.register(s)
		}},
		{"abandoned", func(h *harness) {
			s := original(t, "a2", "d1", 2, later)
			h.register(s)
			if err := h.r.Abandon("ws1", "a2", TransferFailed); err != nil {
				t.Fatal(err)
			}
		}},
		{"inconsistent", func(h *harness) {
			s := original(t, "a2", "d1", 2, later)
			h.register(s)
			h.stage(s, later)
			if state, err := h.r.Stage("ws1", "a2", 1, rows); err != nil || state != Inconsistent {
				t.Fatal(state, err)
			}
		}},
		{"retired", func(h *harness) {
			h.sealRows("a2", "d1", 2, later)
			if err := h.r.Retire("ws1", "a2", WrongSource); err != nil {
				t.Fatal(err)
			}
		}},
		{"staging", func(h *harness) {
			s := original(t, "a2", "d1", 2, later)
			h.register(s)
			h.stage(s, later)
		}},
		{"mistaken high revision", func(h *harness) {
			s := base("a2", "d1", 900)
			s.Scan = NotRun
			h.register(s)
		}},
	}
	for _, step := range steps {
		h := newHarness(t, nil)
		h.sealRows("a1", "d1", 1, rows)
		h.sealRows("b1", "d2", 1, []Record{rec("k2", "1", "USD")})
		h.written(october, 0, "event//k1=10USD@a1 event//k2=1USD@b1")
		step.make(h)
		status := h.status(october)
		staging := 0
		if step.name == "staging" {
			staging = 1
		}
		if status.Differs != Same || status.BlockerTotal != 0 || status.Staging != staging || !reflect.DeepEqual(status.Behind, Listed{Total: 1, Items: []string{"d1"}}) {
			t.Fatalf("%s: %+v", step.name, status)
		}
		before := h.r.dump()
		if out := h.publish(october, 1); out.Number != 1 || out.Written {
			t.Fatalf("%s: %+v", step.name, out)
		}
		h.unchanged(before, step.name)
		if view := h.manifest(october, 1); view.Behind.Total != 0 || len(view.Behind.Items) != 0 || view.Staging != 0 {
			t.Fatalf("%s: %+v", step.name, view)
		}
		h.sealRows("b2", "d2", 2, []Record{rec("k2", "2", "USD")})
		h.written(october, 1, "event//k1=10USD@a1 event//k2=2USD@b2")
		if view := h.manifest(october, 2); !reflect.DeepEqual(view.Behind, Listed{Total: 1, Items: []string{"d1"}}) || view.Staging != staging {
			t.Fatalf("%s: %+v", step.name, view)
		}
	}
	h := newHarness(t, nil)
	h.sealRows("a1", "d1", 1, rows)
	high := base("a2", "d1", 900)
	high.Scan = NotRun
	h.register(high)
	h.sealRows("a3", "d1", 2, later)
	h.sealRows("a4", "d1", 899, rows)
	h.written(october, 0, "event//k1=10USD@a4")
	if view := h.manifest(october, 1); !reflect.DeepEqual(view.Behind, Listed{Total: 1, Items: []string{"d1"}}) {
		t.Fatalf("%+v", view)
	}
	h.sealRows("a5", "d1", 900, later)
	if status := h.status(october); status.Behind.Total != 0 || len(status.Behind.Items) != 0 || status.Differs != DiffersContent {
		t.Fatalf("%+v", status)
	}
}

func TestDocumentsWithoutALiveArtifact(t *testing.T) {
	h := newHarness(t, nil)
	bad := base("a1", "d1", 1)
	bad.Syntax = Failed
	h.register(bad)
	if status := h.status(october); status.Differs != Same || status.Latest != 0 || !reflect.DeepEqual(status.Behind, Listed{Total: 1, Items: []string{"d1"}}) {
		t.Fatalf("%+v", status)
	}
	before := h.r.dump()
	_, err := h.r.Publish(october, 0)
	wantErr(t, err, ErrRejected, "nothing counts yet")
	_, err = h.r.Publish(october, 3)
	wantErr(t, err, ErrConflict, "the expected number is checked before the empty manifest")
	h.unchanged(before, "empty first manifest")
	h.sealRows("b1", "d2", 1, []Record{rec("k1", "1", "USD")})
	h.sealRows("c1", "d3", 1, []Record{rec("k2", "1", "USD")})
	h.sealRows("e1", "d5", 1, []Record{rec("k4", "1", "USD")})
	failed := base("c2", "d3", 2)
	failed.Scan = Failed
	h.register(failed)
	if err := h.r.Retire("ws1", "c1", WrongSource); err != nil {
		t.Fatal(err)
	}
	h.register(base("f1", "d4", 1))
	h.register(base("g1", "d0", 4))
	if err := h.r.Retire("ws1", "e1", WrongSource); err != nil {
		t.Fatal(err)
	}
	h.written(october, 0, "event//k1=1USD@b1")
	view := h.manifest(october, 1)
	if !reflect.DeepEqual(view.Behind, Listed{Total: 5, Items: []string{"d0", "d1", "d3"}}) || view.Staging != 2 {
		t.Fatalf("%+v", view)
	}
	if status := h.status(october); !reflect.DeepEqual(status.Behind, view.Behind) || status.Staging != 2 {
		t.Fatalf("%+v", status)
	}
	view.Behind.Items[0] = "changed"
	view.Sums["USD"] = "9"
	if again := h.manifest(october, 1); again.Behind.Items[0] != "d0" || again.Sums["USD"] != "1" {
		t.Fatal("manifest view shares state")
	}
	status := h.status(october)
	status.Behind.Items[0] = "changed"
	if again := h.manifest(october, 1); again.Behind.Items[0] != "d0" {
		t.Fatal("status view shares state")
	}
}

func TestEmptiedDocumentIsPublished(t *testing.T) {
	h := newHarness(t, nil)
	h.seal(base("a0", "d1", 1))
	before := h.r.dump()
	_, err := h.r.Publish(october, 0)
	wantErr(t, err, ErrRejected, "an empty first manifest")
	h.unchanged(before, "empty first manifest")
	if status := h.status(october); status.Differs != Same || status.Behind.Total != 0 {
		t.Fatalf("%+v", status)
	}
	h.sealRows("a1", "d1", 2, []Record{rec("k1", "10", "USD")})
	h.written(october, 0, "event//k1=10USD@a1")
	h.seal(base("a2", "d1", 3))
	if status := h.status(october); status.Differs != DiffersContent {
		t.Fatalf("%+v", status)
	}
	h.written(october, 1, "")
	view := h.manifest(october, 2)
	sum := sha256.Sum256(nil)
	if view.Records != 0 || len(view.Sums) != 0 || view.Content != hex.EncodeToString(sum[:]) || view.Attribution != view.Content {
		t.Fatalf("%+v", view)
	}
	h.seal(base("a3", "d1", 4))
	before = h.r.dump()
	if out := h.publish(october, 2); out.Number != 2 || out.Written {
		t.Fatalf("%+v", out)
	}
	h.unchanged(before, "nothing to nothing")
	if page, err := h.r.Records(october, 2, 0, 10); err != nil || page != nil {
		t.Fatal(page, err)
	}
}

func TestSameBytesUnderTwoDocumentsAreReported(t *testing.T) {
	h := newHarness(t, nil)
	shared := digestOf("one file")
	make := func(id, document string, at Subject, kind KeyKind, key string) Spec {
		rows := []Record{rec(key, "1", "USD")}
		s := original(t, id, document, 1, rows)
		s.Bytes, s.Subject, s.Kind = shared, at, kind
		h.seal(s, rows)
		return s
	}
	make("a1", "d1", october, LocatorKeys, "r1")
	make("a2", "d1", october, LocatorKeys, "r1")
	if status := h.status(october); status.SameBytes.Total != 0 || len(status.SameBytes.Items) != 0 {
		t.Fatalf("%+v", status)
	}
	make("b1", "d2", october, LocatorKeys, "r1")
	make("n1", "d7", november, LocatorKeys, "r1")
	foreign := october
	foreign.WorkspaceID = "ws2"
	make("b1", "d8", foreign, LocatorKeys, "r1")
	staging := original(t, "s1", "d6", 1, []Record{rec("r1", "1", "USD")})
	staging.Bytes = shared
	h.register(staging)
	if status := h.status(october); !reflect.DeepEqual(status.SameBytes, Listed{Total: 1, Items: []string{"d1+d2"}}) || status.BlockerTotal != 0 {
		t.Fatalf("%+v", status)
	}
	h.written(october, 0, "locator/d1/r1=1USD@a1 locator/d2/r1=1USD@b1")
	if view := h.manifest(october, 1); !reflect.DeepEqual(view.SameBytes, Listed{Total: 1, Items: []string{"d1+d2"}}) || view.Staging != 1 {
		t.Fatalf("%+v", view)
	}
	make("c1", "d3", october, EventKeys, "r1")
	make("e1", "d0", october, EventKeys, "r2")
	want := Listed{Total: 6, Items: []string{"d0+d1", "d0+d2", "d0+d3"}}
	if status := h.status(october); !reflect.DeepEqual(status.SameBytes, want) {
		t.Fatalf("%+v", status)
	}
	if view := h.manifest(october, 1); view.SameBytes.Total != 1 {
		t.Fatal("a published manifest received a later annotation")
	}
	if err := h.r.Retire("ws1", "e1", WrongSource); err != nil {
		t.Fatal(err)
	}
	if err := h.r.Retire("ws1", "a1", WrongSource); err != nil {
		t.Fatal(err)
	}
	if status := h.status(october); !reflect.DeepEqual(status.SameBytes, Listed{Total: 3, Items: []string{"d1+d2", "d1+d3", "d2+d3"}}) {
		t.Fatalf("%+v", status)
	}
	if status := h.status(november); status.SameBytes.Total != 0 {
		t.Fatalf("%+v", status)
	}
	shared = digestOf("another file")
	make("b7", "d2", october, LocatorKeys, "r1")
	make("c7", "d3", october, EventKeys, "r1")
	if status := h.status(october); !reflect.DeepEqual(status.SameBytes, Listed{Total: 3, Items: []string{"d1+d2", "d1+d3", "d2+d3"}}) {
		t.Fatalf("%+v", status)
	}
	view := h.manifest(october, 1)
	view.SameBytes.Items[0] = "changed"
	if again := h.manifest(october, 1); again.SameBytes.Items[0] != "d1+d2" {
		t.Fatal("manifest view shares state")
	}
}

func TestPublicationChecksInOrder(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Manifests = 3 })
	_, err := h.r.Publish(october, 0)
	wantErr(t, err, ErrNotFound, "no document")
	for _, at := range []Subject{{}, {WorkspaceID: "ws1", Provider: "aws", Scope: "payer-1"}, {WorkspaceID: "ws1", Provider: "a b", Scope: "payer-1", Period: "p"}, {WorkspaceID: "ws1", Provider: "aws", Period: "p"}, {Provider: "aws", Scope: "payer-1", Period: "p"}} {
		_, err = h.r.Publish(at, 0)
		wantErr(t, err, ErrInvalid, "subject")
		_, err = h.r.Status(at)
		wantErr(t, err, ErrInvalid, "subject")
		_, err = h.r.Manifest(at, 1)
		wantErr(t, err, ErrInvalid, "subject")
		_, err = h.r.Records(at, 1, 0, 1)
		wantErr(t, err, ErrInvalid, "subject")
	}
	_, err = h.r.Status(november)
	wantErr(t, err, ErrNotFound, "no document")
	h.sealRows("a1", "d1", 1, []Record{rec("k1", "1", "USD")})
	elsewhere := october
	elsewhere.WorkspaceID = "ws2"
	_, err = h.r.Publish(elsewhere, 0)
	wantErr(t, err, ErrNotFound, "other workspace")
	_, err = h.r.Status(elsewhere)
	wantErr(t, err, ErrNotFound, "other workspace")
	before := h.r.dump()
	_, err = h.r.Publish(october, 1)
	wantErr(t, err, ErrConflict, "expected ahead")
	h.unchanged(before, "expected ahead")
	h.clock.advance(time.Minute)
	h.written(october, 0, "event//k1=1USD@a1")
	h.clock.advance(time.Minute)
	before = h.r.dump()
	for _, expected := range []uint64{0, 1, 7} {
		if out := h.publish(october, expected); out.Number != 1 || out.Written {
			t.Fatalf("%+v", out)
		}
	}
	h.unchanged(before, "repeat after a lost response")
	h.register(original(t, "s1", "d2", 1, []Record{rec("k9", "1", "USD")}))
	before = h.r.dump()
	if out := h.publish(october, 1); out.Number != 1 || out.Written {
		t.Fatalf("%+v", out)
	}
	h.unchanged(before, "a change of annotations alone")
	if status := h.status(october); status.Staging != 1 || status.Differs != Same {
		t.Fatalf("%+v", status)
	}
	h.sealRows("a2", "d1", 2, []Record{rec("k1", "2", "USD")})
	before = h.r.dump()
	for _, expected := range []uint64{0, 2} {
		_, err = h.r.Publish(october, expected)
		wantErr(t, err, ErrConflict, "stale or ahead")
	}
	h.unchanged(before, "expected number")
	h.sealRows("x2", "d1", 2, []Record{rec("k1", "3", "USD")})
	h.blocked(october, 0, 1, Blocker{What: InConflict, Document: "d1"})
	if err := h.r.Retire("ws1", "x2", WrongSource); err != nil {
		t.Fatal(err)
	}
	h.written(october, 1, "event//k1=2USD@a2")
	if view := h.manifest(october, 2); !view.At.Equal(start.Add(2*time.Minute)) || view.Staging != 1 {
		t.Fatalf("%+v", view)
	}
	h.sealRows("a3", "d1", 3, []Record{rec("k1", "3", "USD")})
	h.written(october, 2, "event//k1=3USD@a3")
	h.sealRows("a4", "d1", 4, []Record{rec("k1", "4", "USD")})
	before = h.r.dump()
	_, err = h.r.Publish(october, 2)
	wantErr(t, err, ErrConflict, "the expected number is checked before the limit")
	_, err = h.r.Publish(october, 3)
	wantErr(t, err, ErrLimit, "manifest limit")
	h.unchanged(before, "limit")
	if err := h.r.Retire("ws1", "a4", WrongSource); err != nil {
		t.Fatal(err)
	}
	if out := h.publish(october, 3); out.Number != 3 || out.Written {
		t.Fatalf("%+v", out)
	}
	other := original(t, "n1", "d7", 1, []Record{rec("k1", "1", "USD")})
	other.Subject = november
	h.seal(other, []Record{rec("k1", "1", "USD")})
	h.written(november, 0, "event//k1=1USD@n1")
}

func TestManifestsAreReadInPages(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Page = 4 })
	var rows []Record
	for index := 0; index < 10; index++ {
		rows = append(rows, rec(fmt.Sprintf("k%02d", index), "1", "USD"))
	}
	h.sealRows("a1", "d1", 1, rows[:5], rows[5:])
	h.publish(october, 0)
	for _, c := range []struct {
		number       uint64
		after, limit int
		want         error
	}{
		{1, 0, 0, ErrInvalid}, {1, 0, 5, ErrInvalid}, {1, -1, 1, ErrInvalid}, {0, 0, 1, ErrInvalid}, {2, 0, 1, ErrNotFound},
	} {
		_, err := h.r.Records(october, c.number, c.after, c.limit)
		wantErr(t, err, c.want, fmt.Sprint(c))
	}
	_, err := h.r.Records(november, 1, 0, 1)
	wantErr(t, err, ErrNotFound, "another subject")
	elsewhere := october
	elsewhere.WorkspaceID = "ws2"
	_, err = h.r.Records(elsewhere, 1, 0, 1)
	wantErr(t, err, ErrNotFound, "another workspace")
	_, err = h.r.Manifest(elsewhere, 1)
	wantErr(t, err, ErrNotFound, "another workspace")
	_, err = h.r.Manifest(october, 0)
	wantErr(t, err, ErrInvalid, "number zero")
	_, err = h.r.Manifest(october, 2)
	wantErr(t, err, ErrNotFound, "number ahead")
	for _, c := range []struct {
		after, limit int
		first        string
		count        int
	}{{0, 4, "k00", 4}, {4, 4, "k04", 4}, {8, 4, "k08", 2}, {9, 1, "k09", 1}, {0, 1, "k00", 1}, {7, 3, "k07", 3}} {
		page, err := h.r.Records(october, 1, c.after, c.limit)
		if err != nil || len(page) != c.count || page[0].Key != c.first {
			t.Fatalf("%+v: %v %v", c, page, err)
		}
	}
	for _, after := range []int{10, 11} {
		if page, err := h.r.Records(october, 1, after, 4); err != nil || page != nil {
			t.Fatal(page, err)
		}
	}
	if got := len(h.all(october, 1)); got != 10 {
		t.Fatal(got)
	}
	page, _ := h.r.Records(october, 1, 0, 1)
	page[0].Amount = "999"
	if again, _ := h.r.Records(october, 1, 0, 1); again[0].Amount != "1" {
		t.Fatal("records share state")
	}
}
