package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const noRecords = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func TestStageRefusals(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.ChunkRecords = 3 })
	c1 := []Record{rec("k1", "1", "USD"), rec("k2", "2", "USD")}
	c2 := []Record{rec("k3", "3", "USD")}
	s := original(t, "a1", "d1", 1, c1, c2)
	h.register(s)
	h.register(base("e1", "d1", 1))
	good := rec("k9", "1", "USD")
	cases := []struct {
		name    string
		ws, id  string
		number  int
		records []Record
		want    error
	}{
		{"number zero", "ws1", "a1", 0, c1, ErrInvalid},
		{"number negative", "ws1", "a1", -1, c1, ErrInvalid},
		{"number over count", "ws1", "a1", 3, c1, ErrInvalid},
		{"no record", "ws1", "a1", 1, nil, ErrInvalid},
		{"over chunk limit", "ws1", "a1", 1, []Record{rec("a", "1", "USD"), rec("b", "1", "USD"), rec("c", "1", "USD"), rec("d", "1", "USD")}, ErrInvalid},
		{"key", "ws1", "a1", 1, []Record{{Key: "a b", Digest: good.Digest, Amount: "1", Currency: "USD"}}, ErrInvalid},
		{"key empty", "ws1", "a1", 1, []Record{{Key: "", Digest: good.Digest, Amount: "1", Currency: "USD"}}, ErrInvalid},
		{"digest", "ws1", "a1", 1, []Record{{Key: "k", Digest: "xyz", Amount: "1", Currency: "USD"}}, ErrInvalid},
		{"digest long", "ws1", "a1", 1, []Record{{Key: "k", Digest: digestOf("a") + digestOf("b") + "0", Amount: "1", Currency: "USD"}}, ErrInvalid},
		{"amount", "ws1", "a1", 1, []Record{{Key: "k", Digest: good.Digest, Amount: "one", Currency: "USD"}}, ErrInvalid},
		{"amount scale", "ws1", "a1", 1, []Record{{Key: "k", Digest: good.Digest, Amount: "0.00001", Currency: "USD"}}, ErrInvalid},
		{"currency", "ws1", "a1", 1, []Record{{Key: "k", Digest: good.Digest, Amount: "1", Currency: "US"}}, ErrInvalid},
		{"currency case", "ws1", "a1", 1, []Record{{Key: "k", Digest: good.Digest, Amount: "1", Currency: "USd"}}, ErrInvalid},
		{"second record bad", "ws1", "a1", 1, []Record{good, {Key: "k", Digest: good.Digest, Amount: "1", Currency: "1SD"}}, ErrInvalid},
		{"workspace", "", "a1", 1, c1, ErrInvalid},
		{"id", "ws1", "a 1", 1, c1, ErrInvalid},
		{"missing", "ws1", "a9", 1, c1, ErrNotFound},
		{"other workspace", "ws2", "a1", 1, c1, ErrNotFound},
		{"empty artifact", "ws1", "e1", 1, c1, ErrInvalid},
	}
	for _, c := range cases {
		before := h.r.dump()
		_, err := h.r.Stage(c.ws, c.id, c.number, c.records)
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	three := []Record{rec("a", "1", "USD"), rec("b", "1", "USD"), rec("c", "1", "USD")}
	full := original(t, "a2", "d1", 1, three)
	h.register(full)
	h.stage(full, three)
	long := digestOf("a") + digestOf("b")
	wide := []Record{{Key: "k", Digest: long, Amount: "1", Currency: "USD"}}
	edge := original(t, "a3", "d1", 1, wide)
	h.seal(edge, wide)
	short := []Record{{Key: "k", Digest: "0123abcd", Amount: "1", Currency: "USD"}}
	h.seal(original(t, "a4", "d1", 1, short), short)
}

func TestStageRepeatsInAnyOrder(t *testing.T) {
	h := newHarness(t, nil)
	c1 := []Record{rec("k1", "1", "USD"), rec("k2", "2", "USD")}
	c2 := []Record{rec("k3", "3", "USD")}
	c3 := []Record{rec("k4", "4.5", "EUR")}
	s := original(t, "a1", "d1", 1, c1, c2, c3)
	h.register(s)
	for _, step := range []struct {
		number int
		chunk  []Record
		rows   int
	}{{3, c3, 1}, {1, c1, 3}, {3, c3, 3}, {1, c1, 3}, {2, c2, 4}, {2, c2, 4}} {
		h.clock.advance(time.Second)
		state, err := h.r.Stage("ws1", "a1", step.number, step.chunk)
		if err != nil || state != Staging {
			t.Fatal(state, err)
		}
		staged, _ := h.r.Staged("ws1", "a1")
		if staged.Rows != step.rows {
			t.Fatalf("%+v after %d", staged, step.number)
		}
	}
	before := h.r.dump()
	sameValue := []Record{c2[0]}
	sameValue[0].Amount = "3.00"
	if state, err := h.r.Stage("ws1", "a1", 2, sameValue); err != nil || state != Staging {
		t.Fatal(state, err)
	}
	h.unchanged(before, "the same amount written otherwise")
	staged, err := h.r.Staged("ws1", "a1")
	var encoded []byte
	for _, record := range []Record{c1[0], c1[1], c2[0], c3[0]} {
		encoded = append(encoded, secondEncoding(record.Key, record.Amount, record.Currency, record.Digest)...)
	}
	stagedSum := sha256.Sum256(encoded)
	want := StagedView{Chunks: 3, Rows: 4, Sums: map[string]string{"USD": "6", "EUR": "4.5"}, Complete: true, Digest: hex.EncodeToString(stagedSum[:])}
	if err != nil || !reflect.DeepEqual(staged, want) {
		t.Fatalf("%+v %v", staged, err)
	}
	if view := h.view("a1"); !view.Changed.Equal(start) || view.Rows != 4 {
		t.Fatalf("%+v", view)
	}
	if h.r.workspaces["ws1"].records != 4 {
		t.Fatal(h.r.workspaces["ws1"].records)
	}
	if state, _, err := h.r.Seal("ws1", "a1"); err != nil || state != Sealed {
		t.Fatal(state, err)
	}
	before = h.r.dump()
	if state, err := h.r.Stage("ws1", "a1", 2, c2); err != nil || state != Sealed {
		t.Fatal(state, err)
	}
	_, err = h.r.Stage("ws1", "a1", 2, c3)
	wantErr(t, err, ErrRejected, "other content after sealing")
	h.unchanged(before, "stage after sealing")
}

func TestContradictionsMakeAnArtifactInconsistent(t *testing.T) {
	c1 := []Record{rec("k1", "1", "USD"), rec("k2", "2", "USD")}
	c2 := []Record{rec("k3", "3", "USD"), rec("k4", "4", "USD")}
	c3 := []Record{rec("k5", "5", "USD")}
	otherDigest := c1[0]
	otherDigest.Digest = digestOf("other")
	cases := []struct {
		name   string
		number int
		chunk  []Record
	}{
		{"other order", 1, []Record{c1[1], c1[0]}},
		{"other amount", 1, []Record{rec("k1", "1.0001", "USD"), c1[1]}},
		{"other currency", 1, []Record{rec("k1", "1", "EUR"), c1[1]}},
		{"other row digest", 1, []Record{otherDigest, c1[1]}},
		{"other key", 1, []Record{rec("k0", "1", "USD"), c1[1]}},
		{"other keys entirely", 1, []Record{rec("k8", "1", "USD"), rec("k9", "2", "USD")}},
		{"one record fewer", 1, []Record{c1[0]}},
		{"one record more", 1, []Record{c1[0], c1[1], rec("k9", "1", "USD")}},
		{"key twice in one chunk", 2, []Record{rec("k3", "3", "USD"), rec("k3", "3", "USD")}},
		{"key of another chunk", 2, []Record{rec("k3", "3", "USD"), c1[1]}},
		{"key of another chunk first", 2, []Record{c1[0], rec("k4", "4", "USD")}},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		s := original(t, "a1", "d1", 1, c1, c2, c3)
		h.register(s)
		h.stage(s, c1)
		if _, err := h.r.Stage("ws1", "a1", 3, c3); err != nil {
			t.Fatal(err)
		}
		h.clock.advance(time.Minute)
		state, err := h.r.Stage("ws1", "a1", c.number, c.chunk)
		if err != nil || state != Inconsistent {
			t.Fatalf("%s: %v %v", c.name, state, err)
		}
		view := h.view("a1")
		if view.State != Inconsistent || view.Contradiction != c.number || !view.Changed.Equal(start.Add(time.Minute)) || !view.Registered.Equal(start) {
			t.Fatalf("%s: %+v", c.name, view)
		}
		if h.r.workspaces["ws1"].records != 0 {
			t.Fatalf("%s: records kept", c.name)
		}
		h.clock.advance(time.Minute)
		before := h.r.dump()
		if state, err := h.r.Stage("ws1", "a1", 1, c1); err != nil || state != Inconsistent {
			t.Fatalf("%s repeat: %v %v", c.name, state, err)
		}
		if state, err := h.r.Stage("ws1", "a1", 3, c3); err != nil || state != Inconsistent {
			t.Fatalf("%s repeat: %v %v", c.name, state, err)
		}
		if c.number == 2 {
			_, err = h.r.Stage("ws1", "a1", 2, c.chunk)
			wantErr(t, err, ErrRejected, c.name+" contradicting chunk again")
			_, err = h.r.Stage("ws1", "a1", 2, c2)
			wantErr(t, err, ErrRejected, c.name+" new chunk")
		} else {
			_, err = h.r.Stage("ws1", "a1", 1, c.chunk)
			wantErr(t, err, ErrRejected, c.name+" contradicting chunk again")
		}
		_, _, err = h.r.Seal("ws1", "a1")
		wantErr(t, err, ErrRejected, c.name+" seal")
		wantErr(t, h.r.Abandon("ws1", "a1", Unsealable), ErrRejected, c.name+" abandon")
		wantErr(t, h.r.Retire("ws1", "a1", Reissued), ErrRejected, c.name+" retire")
		_, err = h.r.Staged("ws1", "a1")
		wantErr(t, err, ErrRejected, c.name+" staged")
		if state, _, err := h.r.Register(s); err != nil || state != Inconsistent {
			t.Fatalf("%s registered again: %v %v", c.name, state, err)
		}
		h.unchanged(before, c.name)
		if status := h.status(october); status.Staging != 0 || status.Behind.Total != 1 {
			t.Fatalf("%s: %+v", c.name, status)
		}
	}
}

func TestSealListsEveryGap(t *testing.T) {
	usd := []Record{rec("k1", "1", "USD"), rec("k2", "5", "USD")}
	cases := []struct {
		name   string
		first  []Record
		second []Record
		want   []Gap
	}{
		{"nothing staged", nil, nil, []Gap{{What: GapChunks}, {What: GapRows}, {What: GapNoRows, Currency: "EUR"}, {What: GapNoRows, Currency: "USD"}}},
		{"one chunk", usd, nil, []Gap{{What: GapChunks}, {What: GapRows}, {What: GapNoRows, Currency: "EUR"}}},
		{"other chunk", nil, []Record{rec("k3", "2.5", "EUR"), rec("k4", "-2.5", "EUR")}, []Gap{{What: GapChunks}, {What: GapRows}, {What: GapNoRows, Currency: "USD"}}},
		{"a row missing", usd, []Record{rec("k3", "0", "EUR")}, []Gap{{What: GapRows}}},
		{"a row more", usd, []Record{rec("k3", "0", "EUR"), rec("k4", "0", "EUR"), rec("k5", "0", "EUR")}, []Gap{{What: GapRows}}},
		{"sum differs", usd, []Record{rec("k3", "2.5", "EUR"), rec("k4", "-2.4999", "EUR")}, []Gap{{What: GapSum, Currency: "EUR"}}},
		{"both sums differ", []Record{rec("k1", "1", "USD"), rec("k2", "5.0001", "USD")}, []Record{rec("k3", "2.5", "EUR"), rec("k4", "2.5", "EUR")}, []Gap{{What: GapSum, Currency: "EUR"}, {What: GapSum, Currency: "USD"}}},
		{"currency not declared", usd, []Record{rec("k3", "0", "EUR"), rec("k4", "1", "KRW")}, []Gap{{What: GapCurrency, Currency: "KRW"}}},
		{"declared currency without rows", usd, []Record{rec("k3", "1", "KRW"), rec("k4", "-1", "AUD")}, []Gap{{What: GapCurrency, Currency: "AUD"}, {What: GapNoRows, Currency: "EUR"}, {What: GapCurrency, Currency: "KRW"}}},
		{"sum differs by a power of ten", []Record{rec("k1", "0.1", "USD"), rec("k2", "0.5", "USD")}, []Record{rec("k3", "2.5", "EUR"), rec("k4", "-2.5", "EUR")}, []Gap{{What: GapSum, Currency: "USD"}}},
		{"sealable", usd, []Record{rec("k3", "2.5", "EUR"), rec("k4", "-2.5", "EUR")}, nil},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		s := original(t, "a1", "d1", 1, usd)
		s.Chunks, s.Rows, s.Sums = 2, 4, map[string]string{"USD": "6", "EUR": "0"}
		h.register(s)
		if c.first != nil {
			h.stage(s, c.first)
		}
		if c.second != nil {
			if _, err := h.r.Stage("ws1", "a1", 2, c.second); err != nil {
				t.Fatal(err)
			}
		}
		h.clock.advance(time.Minute)
		before := h.r.dump()
		state, gaps, err := h.r.Seal("ws1", "a1")
		if c.want == nil {
			if err != nil || state != Sealed || gaps != nil {
				t.Fatalf("%s: %v %v %v", c.name, state, gaps, err)
			}
			view := h.view("a1")
			if view.Sequence != 1 || view.Rows != 4 || !reflect.DeepEqual(view.Sums, map[string]string{"USD": "6", "EUR": "0"}) || !view.Changed.Equal(start.Add(time.Minute)) {
				t.Fatalf("%+v", view)
			}
			continue
		}
		if err != ErrRejected || state != "" || !reflect.DeepEqual(gaps, c.want) {
			t.Fatalf("%s: %v %v %v", c.name, state, gaps, err)
		}
		h.unchanged(before, c.name)
		if err := h.r.Abandon("ws1", "a1", Unsealable); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSealedDigestFollowsTheStatedEncoding(t *testing.T) {
	h := newHarness(t, nil)
	c1 := []Record{rec("b", "2.50", "USD"), rec("a", "-1", "EUR")}
	c2 := []Record{rec("B", "0.0001", "USD")}
	h.sealRows("a1", "d1", 1, c1, c2)
	var encoded []byte
	encoded = append(encoded, secondEncoding("B", "0.0001", "USD", c2[0].Digest)...)
	encoded = append(encoded, secondEncoding("a", "-1", "EUR", c1[1].Digest)...)
	encoded = append(encoded, secondEncoding("b", "2.5", "USD", c1[0].Digest)...)
	sum := sha256.Sum256(encoded)
	view := h.view("a1")
	if view.Digest != hex.EncodeToString(sum[:]) || view.Sequence != 1 {
		t.Fatalf("%+v", view)
	}
	other := newHarness(t, nil)
	other.sealRows("a1", "d1", 1, []Record{c2[0], c1[1]}, []Record{c1[0]})
	if got := other.view("a1").Digest; got != view.Digest {
		t.Fatal("digest depends on chunking")
	}
	h.clock.advance(time.Minute)
	before := h.r.dump()
	if state, gaps, err := h.r.Seal("ws1", "a1"); err != nil || state != Sealed || gaps != nil {
		t.Fatal(state, gaps, err)
	}
	h.unchanged(before, "repeated seal")
	h.seal(base("e1", "d1", 2))
	h.seal(scan("o1", "d1", 3))
	for id, sequence := range map[string]uint64{"e1": 2, "o1": 3} {
		if view := h.view(id); view.Digest != noRecords || view.Sequence != sequence || view.Rows != 0 || len(view.Sums) != 0 {
			t.Fatalf("%s: %+v", id, view)
		}
	}
	if err := h.r.Retire("ws1", "a1", WrongSource); err != nil {
		t.Fatal(err)
	}
	before = h.r.dump()
	if state, gaps, err := h.r.Seal("ws1", "a1"); err != nil || state != Retired || gaps != nil {
		t.Fatal(state, gaps, err)
	}
	h.unchanged(before, "seal after retirement")
	if view := h.view("a1"); view.Digest == "" || view.Rows != 3 || view.RetireReason != WrongSource {
		t.Fatalf("%+v", view)
	}
	for _, c := range []struct{ ws, id string }{{"ws1", "a9"}, {"ws2", "e1"}} {
		_, _, err := h.r.Seal(c.ws, c.id)
		wantErr(t, err, ErrNotFound, c.id)
	}
	_, _, err := h.r.Seal("", "a1")
	wantErr(t, err, ErrInvalid, "workspace")
	_, _, err = h.r.Seal("ws1", "")
	wantErr(t, err, ErrInvalid, "id")
}

func TestSumsDoNotDependOnTheOrderOfArrival(t *testing.T) {
	small, err := decimal.NewProfile(decimal.ProfileSpec{ID: "small", Revision: "1", InputIntegerDigits: 3, InputScale: 2, ResultIntegerDigits: 3, ResultScale: 2, IntermediateDigits: 10})
	if err != nil {
		t.Fatal(err)
	}
	up := [][]Record{{rec("k1", "900", "USD")}, {rec("k2", "900.5", "USD")}, {rec("k3", "-900", "USD")}, {rec("k4", "-900.45", "USD")}}
	digests := map[string]bool{}
	for _, order := range [][]int{{0, 1, 2, 3}, {2, 3, 0, 1}, {3, 1, 0, 2}} {
		clock := &testClock{at: start}
		r, err := NewRegistry(testLimits(), small, clock.read)
		if err != nil {
			t.Fatal(err)
		}
		s := base("a1", "d1", 1)
		s.Holds, s.Chunks, s.Rows, s.Sums, s.Totals = Rows, 4, 4, map[string]string{"USD": "0.05"}, SourceStated
		if _, _, err := r.Register(s); err != nil {
			t.Fatal(err)
		}
		for step, index := range order {
			if _, err := r.Stage("ws1", "a1", index+1, up[index]); err != nil {
				t.Fatal(err)
			}
			staged, _ := r.Staged("ws1", "a1")
			if step == 1 && order[0] == 0 && staged.Sums["USD"] != "1800.5" {
				t.Fatalf("%+v", staged)
			}
			if step == 1 && order[0] == 2 && staged.Sums["USD"] != "-1800.45" {
				t.Fatalf("%+v", staged)
			}
		}
		state, gaps, err := r.Seal("ws1", "a1")
		if err != nil || state != Sealed {
			t.Fatal(order, state, gaps, err)
		}
		view, _ := r.Artifact("ws1", "a1")
		if view.Sums["USD"] != "0.05" {
			t.Fatalf("%+v", view)
		}
		digests[view.Digest] = true
	}
	if len(digests) != 1 {
		t.Fatal(digests)
	}
	h := newHarness(t, nil)
	rows := []Record{rec("a", "-0.05", "USD"), rec("b", "0.5", "EUR"), rec("c", "0.5", "EUR"), rec("d", "100", "KRW"), rec("e", "0.0001", "KRW"), rec("f", "-0.5", "AUD"), rec("g", "0.5", "AUD"), rec("h", "-12.3", "JPY"), rec("i", "120", "CAD"), rec("j", "0.007", "CHF")}
	s := original(t, "a1", "d1", 1, rows)
	h.register(s)
	h.stage(s, rows)
	staged, _ := h.r.Staged("ws1", "a1")
	want := map[string]string{"USD": "-0.05", "EUR": "1", "KRW": "100.0001", "AUD": "0", "JPY": "-12.3", "CAD": "120", "CHF": "0.007"}
	if !reflect.DeepEqual(staged.Sums, want) {
		t.Fatalf("%v", staged.Sums)
	}
	if !reflect.DeepEqual(s.Sums, want) {
		t.Fatalf("test totals %v", s.Sums)
	}
	if state, gaps, err := h.r.Seal("ws1", "a1"); err != nil || state != Sealed {
		t.Fatal(gaps, err)
	}
}

func TestVerificationBindsAnExtractionToItsContent(t *testing.T) {
	h := newHarness(t, nil)
	c1 := []Record{rec("p1:l1", "10.5", "USD")}
	c2 := []Record{rec("p1:l2", "2", "USD")}
	h.register(scan("o1", "d1", 1))
	x := derived(t, "x1", "d1", 1, "o1", c1, c2)
	h.register(x)
	h.stage(x, c1)
	partial, _ := h.r.Staged("ws1", "x1")
	before := h.r.dump()
	wantErr(t, h.r.Verify("ws1", "x1", partial.Digest, "v:1", "e:1"), ErrRejected, "before every chunk")
	h.unchanged(before, "early verification")
	h.stage(x, c1, c2)
	staged, _ := h.r.Staged("ws1", "x1")
	if !staged.Complete || partial.Complete || staged.Digest == partial.Digest {
		t.Fatalf("%+v %+v", staged, partial)
	}
	before = h.r.dump()
	state, gaps, err := h.r.Seal("ws1", "x1")
	if err != ErrRejected || state != "" || !reflect.DeepEqual(gaps, []Gap{{What: GapOriginal}, {What: GapVerified}}) {
		t.Fatal(state, gaps, err)
	}
	for _, c := range []struct {
		name                           string
		ws, id, digest, verifier, note string
		want                           error
	}{
		{"other digest", "ws1", "x1", partial.Digest, "v:1", "e:1", ErrConflict},
		{"digest form", "ws1", "x1", "zz", "v:1", "e:1", ErrInvalid},
		{"verifier", "ws1", "x1", staged.Digest, "", "e:1", ErrInvalid},
		{"evidence", "ws1", "x1", staged.Digest, "v:1", "see page 2", ErrInvalid},
		{"workspace form", "", "x1", staged.Digest, "v:1", "e:1", ErrInvalid},
		{"id form", "ws1", "", staged.Digest, "v:1", "e:1", ErrInvalid},
		{"missing", "ws1", "x9", staged.Digest, "v:1", "e:1", ErrNotFound},
		{"other workspace", "ws2", "x1", staged.Digest, "v:1", "e:1", ErrNotFound},
		{"an original", "ws1", "o1", noRecords, "v:1", "e:1", ErrRejected},
	} {
		wantErr(t, h.r.Verify(c.ws, c.id, c.digest, c.verifier, c.note), c.want, c.name)
		h.unchanged(before, c.name)
	}
	h.clock.advance(time.Minute)
	if err := h.r.Verify("ws1", "x1", staged.Digest, "v:1", "e:1"); err != nil {
		t.Fatal(err)
	}
	view := h.view("x1")
	if view.Verification == nil || *view.Verification != (Verification{Digest: staged.Digest, Verifier: "v:1", Evidence: "e:1", At: start.Add(time.Minute)}) || !view.Changed.Equal(start) {
		t.Fatalf("%+v", view.Verification)
	}
	view.Verification.Verifier = "changed"
	if again := h.view("x1"); again.Verification.Verifier != "v:1" {
		t.Fatal("view shares the verification")
	}
	h.clock.advance(time.Minute)
	before = h.r.dump()
	if err := h.r.Verify("ws1", "x1", staged.Digest, "v:1", "e:1"); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Verify("ws1", "x1", staged.Digest, "v:2", "e:1"), ErrConflict, "other verifier")
	wantErr(t, h.r.Verify("ws1", "x1", staged.Digest, "v:1", "e:2"), ErrConflict, "other evidence")
	wantErr(t, h.r.Verify("ws1", "x1", partial.Digest, "v:1", "e:1"), ErrConflict, "other digest afterwards")
	state, gaps, err = h.r.Seal("ws1", "x1")
	if err != ErrRejected || !reflect.DeepEqual(gaps, []Gap{{What: GapOriginal}}) {
		t.Fatal(state, gaps, err)
	}
	h.unchanged(before, "verification repeats")
	h.seal(scan("o1", "d1", 1))
	if state, gaps, err := h.r.Seal("ws1", "x1"); err != nil || state != Sealed {
		t.Fatal(state, gaps, err)
	}
	if h.view("x1").Digest != staged.Digest {
		t.Fatal("sealed content is not the verified content")
	}
	before = h.r.dump()
	if err := h.r.Verify("ws1", "x1", staged.Digest, "v:1", "e:1"); err != nil {
		t.Fatal(err)
	}
	h.unchanged(before, "verification repeated after sealing")

	y := derived(t, "x2", "d1", 1, "o1", c1, c2)
	h.register(y)
	h.stage(y, c1, c2)
	if state, gaps, err := h.r.Seal("ws1", "x2"); err != ErrRejected || !reflect.DeepEqual(gaps, []Gap{{What: GapVerified}}) {
		t.Fatal(state, gaps, err)
	}
	if err := h.r.Verify("ws1", "x2", staged.Digest, "v:1", "e:1"); err != nil {
		t.Fatal(err)
	}
	misread := []Record{rec("p1:l2", "20", "USD")}
	if state, err := h.r.Stage("ws1", "x2", 2, misread); err != nil || state != Inconsistent {
		t.Fatal(state, err)
	}
	_, _, err = h.r.Seal("ws1", "x2")
	wantErr(t, err, ErrRejected, "other content after verification")

	z := derived(t, "x3", "d1", 1, "o1", c1, c2)
	h.register(z)
	h.stage(z, c1, c2)
	if err := h.r.Abandon("ws1", "x3", Superseded); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Verify("ws1", "x3", staged.Digest, "v:1", "e:1"), ErrRejected, "abandoned")
}

func TestAbandonAndRetire(t *testing.T) {
	chunk := []Record{rec("k1", "1", "USD"), rec("k2", "1", "USD")}
	for _, reason := range []AbandonReason{TransferFailed, Superseded, Unsealable, AbandonedBy} {
		h := newHarness(t, func(l *Limits) { l.Records = 2 })
		s := original(t, "a1", "d1", 1, chunk)
		h.register(s)
		h.stage(s, chunk)
		before := h.r.dump()
		for _, bad := range []AbandonReason{"", "wrong_source", "other"} {
			wantErr(t, h.r.Abandon("ws1", "a1", bad), ErrInvalid, "reason")
		}
		wantErr(t, h.r.Abandon("ws1", "a9", reason), ErrNotFound, "missing")
		wantErr(t, h.r.Abandon("ws2", "a1", reason), ErrNotFound, "other workspace")
		wantErr(t, h.r.Abandon("", "a1", reason), ErrInvalid, "workspace")
		wantErr(t, h.r.Abandon("ws1", "", reason), ErrInvalid, "id")
		wantErr(t, h.r.Retire("ws1", "a1", Reissued), ErrRejected, "retire while staging")
		h.unchanged(before, "refusals")
		h.clock.advance(time.Minute)
		if err := h.r.Abandon("ws1", "a1", reason); err != nil {
			t.Fatal(err)
		}
		view := h.view("a1")
		if view.State != Abandoned || view.AbandonReason != reason || !view.Changed.Equal(start.Add(time.Minute)) || h.r.workspaces["ws1"].records != 0 {
			t.Fatalf("%+v", view)
		}
		h.clock.advance(time.Minute)
		before = h.r.dump()
		if err := h.r.Abandon("ws1", "a1", reason); err != nil {
			t.Fatal(err)
		}
		other := Superseded
		if reason == Superseded {
			other = AbandonedBy
		}
		wantErr(t, h.r.Abandon("ws1", "a1", other), ErrConflict, "other reason")
		if state, err := h.r.Stage("ws1", "a1", 1, chunk); err != nil || state != Abandoned {
			t.Fatal(state, err)
		}
		_, err := h.r.Stage("ws1", "a1", 1, []Record{chunk[0]})
		wantErr(t, err, ErrRejected, "other chunk after abandonment")
		_, _, err = h.r.Seal("ws1", "a1")
		wantErr(t, err, ErrRejected, "seal")
		wantErr(t, h.r.Retire("ws1", "a1", Reissued), ErrRejected, "retire")
		_, err = h.r.Staged("ws1", "a1")
		wantErr(t, err, ErrRejected, "staged")
		if state, _, err := h.r.Register(s); err != nil || state != Abandoned {
			t.Fatal(state, err)
		}
		h.unchanged(before, "after abandonment")
		again := original(t, "a2", "d1", 1, chunk)
		h.seal(again, chunk)
	}
	for _, reason := range []RetireReason{WrongSource, Reissued, VerificationWithdrawn, RetiredBy} {
		h := newHarness(t, func(l *Limits) { l.Records = 2 })
		h.sealRows("a1", "d1", 1, chunk)
		before := h.r.dump()
		for _, bad := range []RetireReason{"", "superseded", "other"} {
			wantErr(t, h.r.Retire("ws1", "a1", bad), ErrInvalid, "reason")
		}
		wantErr(t, h.r.Retire("ws1", "a9", reason), ErrNotFound, "missing")
		wantErr(t, h.r.Retire("ws2", "a1", reason), ErrNotFound, "other workspace")
		wantErr(t, h.r.Retire("", "a1", reason), ErrInvalid, "workspace")
		wantErr(t, h.r.Retire("ws1", "", reason), ErrInvalid, "id")
		wantErr(t, h.r.Abandon("ws1", "a1", Superseded), ErrRejected, "abandon a sealed one")
		h.unchanged(before, "refusals")
		h.clock.advance(time.Minute)
		if err := h.r.Retire("ws1", "a1", reason); err != nil {
			t.Fatal(err)
		}
		view := h.view("a1")
		if view.State != Retired || view.RetireReason != reason || !view.Changed.Equal(start.Add(time.Minute)) || h.r.workspaces["ws1"].records != 0 || view.Sequence != 1 {
			t.Fatalf("%+v", view)
		}
		h.clock.advance(time.Minute)
		before = h.r.dump()
		if err := h.r.Retire("ws1", "a1", reason); err != nil {
			t.Fatal(err)
		}
		other := Reissued
		if reason == Reissued {
			other = RetiredBy
		}
		wantErr(t, h.r.Retire("ws1", "a1", other), ErrConflict, "other reason")
		wantErr(t, h.r.Abandon("ws1", "a1", Superseded), ErrRejected, "abandon")
		if state, err := h.r.Stage("ws1", "a1", 1, chunk); err != nil || state != Retired {
			t.Fatal(state, err)
		}
		h.unchanged(before, "after retirement")
		h.sealRows("a2", "d1", 1, chunk)
	}
}

func TestRecordLimitCountsWhatCanBeUsed(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Records = 5 })
	three := []Record{rec("k1", "1", "USD"), rec("k2", "1", "USD"), rec("k3", "1", "USD")}
	two := []Record{rec("k4", "1", "USD"), rec("k5", "1", "USD")}
	one := []Record{rec("k6", "1", "USD")}
	a := original(t, "a1", "d1", 1, three, two)
	h.register(a)
	h.stage(a, three)
	b := original(t, "b1", "d2", 1, three)
	h.register(b)
	before := h.r.dump()
	_, err := h.r.Stage("ws1", "b1", 1, three)
	wantErr(t, err, ErrLimit, "six of five")
	h.unchanged(before, "limit")
	if state, err := h.r.Stage("ws1", "a1", 1, three); err != nil || state != Staging {
		t.Fatal(state, err)
	}
	h.stage(a, three, two)
	if h.r.workspaces["ws1"].records != 5 {
		t.Fatal("five of five")
	}
	c := original(t, "c1", "d3", 1, one)
	h.register(c)
	_, err = h.r.Stage("ws1", "c1", 1, one)
	wantErr(t, err, ErrLimit, "full")
	if state, _, err := h.r.Seal("ws1", "a1"); err != nil || state != Sealed {
		t.Fatal(state, err)
	}
	_, err = h.r.Stage("ws1", "c1", 1, one)
	wantErr(t, err, ErrLimit, "sealed records stay counted")
	elsewhere := original(t, "c1", "d3", 1, three)
	elsewhere.Subject.WorkspaceID = "ws2"
	h.register(elsewhere)
	if _, err := h.r.Stage("ws2", "c1", 1, three); err != nil {
		t.Fatal(err)
	}
	if err := h.r.Retire("ws1", "a1", Reissued); err != nil {
		t.Fatal(err)
	}
	h.stage(b, three)
	h.stage(c, one)
	if h.r.workspaces["ws1"].records != 4 || h.r.workspaces["ws2"].records != 3 {
		t.Fatal(h.r.workspaces["ws1"].records)
	}
	if state, err := h.r.Stage("ws1", "b1", 1, two); err != nil || state != Inconsistent {
		t.Fatal(state, err)
	}
	if h.r.workspaces["ws1"].records != 1 {
		t.Fatal(h.r.workspaces["ws1"].records)
	}
	if err := h.r.Abandon("ws1", "c1", TransferFailed); err != nil {
		t.Fatal(err)
	}
	if h.r.workspaces["ws1"].records != 0 || h.r.workspaces["ws2"].records != 3 {
		t.Fatal(h.r.workspaces["ws1"].records)
	}
	full := newHarness(t, func(l *Limits) { l.Records = 3 })
	s := original(t, "a1", "d1", 1, three)
	full.register(s)
	full.stage(s, three)
	contradiction := []Record{three[0], three[1], three[2], rec("k9", "1", "USD")}
	if state, err := full.r.Stage("ws1", "a1", 1, contradiction); err != nil || state != Inconsistent {
		t.Fatal(state, err)
	}
}

func TestClockNeverGoesBackwards(t *testing.T) {
	h := newHarness(t, nil)
	h.clock.advance(time.Hour)
	h.register(base("a1", "d1", 1))
	h.clock.advance(-30 * time.Minute)
	h.register(base("a2", "d1", 2))
	if state, _, err := h.r.Seal("ws1", "a1"); err != nil || state != Sealed {
		t.Fatal(err)
	}
	for _, id := range []string{"a1", "a2"} {
		if view := h.view(id); !view.Registered.Equal(start.Add(time.Hour)) || !view.Changed.Equal(start.Add(time.Hour)) {
			t.Fatalf("%s %+v", id, view)
		}
	}
	h.clock.advance(31 * time.Minute)
	if state, _, err := h.r.Seal("ws1", "a2"); err != nil || state != Sealed {
		t.Fatal(err)
	}
	if view := h.view("a2"); !view.Changed.Equal(start.Add(time.Hour+time.Minute)) || !view.Registered.Equal(start.Add(time.Hour)) {
		t.Fatalf("%+v", view)
	}
	if _, err := h.r.Artifact("ws2", "a1"); err != ErrNotFound {
		t.Fatal(err)
	}
	if _, err := h.r.Artifact("ws1", "zz"); err != ErrNotFound {
		t.Fatal(err)
	}
	if _, err := h.r.Artifact("", "a1"); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := h.r.Artifact("ws1", ""); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := h.r.Staged("ws2", "a1"); err != ErrNotFound {
		t.Fatal(err)
	}
	if _, err := h.r.Staged("", "a1"); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := h.r.Staged("ws1", ""); err != ErrInvalid {
		t.Fatal(err)
	}
	view := h.view("a1")
	view.Spec.Sums["USD"] = "9"
	view.Reasons = append(view.Reasons, "x")
	if again := h.view("a1"); len(again.Spec.Sums) != 0 || len(again.Reasons) != 0 {
		t.Fatal("view shares state")
	}
}
