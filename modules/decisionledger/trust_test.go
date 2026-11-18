package decisionledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func digest(c string) string { return strings.Repeat(c, 64) }

func trustAt(epoch, policy uint64) Decision {
	return Decision{Workspace: "sample-workspace", Decision: TrustActivation, Trust: &Trust{Change: "trust-0a1b2c3d-" + strconv.FormatUint(epoch, 10) + "-" + strconv.FormatUint(policy, 10), ChangeRevision: 2, ChangeDigest: digest("c"),
		EvidenceDigest: digest("e"), OldEpoch: epoch - 1, NewEpoch: epoch, OldPolicy: policy - 1, NewPolicy: policy, OldDigest: digest("0"), NewDigest: digest("1"),
		Config:     Config{Methods: []string{"password"}, MFA: "none", MaxAgeSeconds: 3600, StepUp: []StepRule{{Action: "policy.change", MaxAgeSeconds: 600}}},
		ProposedBy: "user-a", ValidatedBy: "user-b", ApprovedBy: "user-b", ActivatedBy: "user-c", ApprovedAt: start, RunStartedAt: start, InvalidateBelow: epoch,
		AffectedSessions: 3, Origin: OriginActivation}}
}

func failedWith(t testing.TB, name string, err error, reason string) {
	t.Helper()
	var ce *CommitError
	if !errors.As(err, &ce) || ce.Reason != reason {
		t.Errorf("%s: %v", name, err)
	}
}

func TestTrustRecordsRoundTripAndReplayInOrder(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "1")
	first, err := l.Commit(trustAt(1, 1))
	if err != nil {
		t.Fatal(err)
	}
	commit(t, l, "2")
	if _, err := l.Commit(trustAt(2, 2)); err != nil {
		t.Fatal(err)
	}
	failedWith(t, "the same epoch", second(l.Commit(trustAt(2, 3))), ReasonCommitFailed)
	failedWith(t, "an older epoch", second(l.Commit(trustAt(1, 3))), ReasonCommitFailed)
	failedWith(t, "another generation", second(l.Commit(trustAt(1<<32+1, 3))), ReasonCommitFailed)
	other := trustAt(1, 1)
	other.Workspace = "other"
	if _, err := l.Commit(other); err != nil {
		t.Errorf("epochs are per workspace: %v", err)
	}
	for name, change := range map[string]func(*Decision){
		"no trust":            func(d *Decision) { d.Trust = nil },
		"a revocation field":  func(d *Decision) { d.Target = "grant-1" },
		"a short digest":      func(d *Decision) { d.Trust.ChangeDigest = "abc" },
		"no epoch number":     func(d *Decision) { d.Trust.NewEpoch, d.Trust.InvalidateBelow = 1<<32, 1<<32 },
		"no newer policy":     func(d *Decision) { d.Trust.NewPolicy = d.Trust.OldPolicy },
		"another floor":       func(d *Decision) { d.Trust.InvalidateBelow = 2 },
		"an actor missing":    func(d *Decision) { d.Trust.ActivatedBy = "" },
		"no methods":          func(d *Decision) { d.Trust.Config.Methods = nil },
		"an empty method":     func(d *Decision) { d.Trust.Config.Methods = []string{""} },
		"no mfa":              func(d *Decision) { d.Trust.Config.MFA = "" },
		"no age":              func(d *Decision) { d.Trust.Config.MaxAgeSeconds = 0 },
		"a bad rule":          func(d *Decision) { d.Trust.Config.StepUp = []StepRule{{Action: ""}} },
		"an unnamed rule":     func(d *Decision) { d.Trust.Config.StepUp = []StepRule{{MaxAgeSeconds: 60}} },
		"a rule without age":  func(d *Decision) { d.Trust.Config.StepUp = []StepRule{{Action: "x"}} },
		"no change revision":  func(d *Decision) { d.Trust.ChangeRevision = 0 },
		"no newer epoch":      func(d *Decision) { d.Trust.OldEpoch = d.Trust.NewEpoch },
		"an unknown origin":   func(d *Decision) { d.Trust.Origin = "restore" },
		"a carried no source": func(d *Decision) { d.Trust.Origin = OriginCarried },
		"no change":           func(d *Decision) { d.Trust.Change = "" },
		"a trust on a revoke": func(d *Decision) { d.Decision = RevokeGrant },
	} {
		d := trustAt(3, 3)
		change(&d)
		failedWith(t, name, second(l.Commit(d)), ReasonCommitFailed)
	}
	carried := trustAt(3, 3)
	carried.Trust.Origin, carried.Trust.CarriedFrom = OriginCarried, "abc:2"
	if _, err := l.Commit(carried); err != nil {
		t.Errorf("a carried record without a problem line %v", err)
	}
	big := trustAt(4, 4)
	for n := 0; n < 64; n++ {
		big.Trust.Config.StepUp = append(big.Trust.Config.StepUp, StepRule{Action: "action-" + strings.Repeat("x", 20), MaxAgeSeconds: 60})
	}
	failedWith(t, "an oversized record", second(l.Commit(big)), ReasonCommitFailed)
	if l.Status().State != StateOK {
		t.Errorf("a refused trust record stopped the ledger %+v", l.Status())
	}
	before := l.Decisions()
	before[1].Decision.Trust.Config.Methods[0] = "changed"
	if l.Decisions()[1].Decision.Trust.Config.Methods[0] != "password" {
		t.Error("the records share memory")
	}
	l.Close()
	again := must(t, o)
	defer again.Close()
	records := again.Decisions()
	if len(records) != 6 || records[1].Sequence != first.Sequence || records[0].Decision.Trust != nil {
		t.Fatalf("records %+v", records)
	}
	want := trustAt(1, 1)
	if got := records[1].Decision; !reflect.DeepEqual(got, want) {
		t.Errorf("round trip\n%+v\n%+v", got.Trust, want.Trust)
	}
	if !reflect.DeepEqual(again.Verified(), records) {
		t.Error("verified differs from decisions on a usable ledger")
	}
	failedWith(t, "after restart the same epoch", second(again.Commit(trustAt(2, 3))), ReasonCommitFailed)
	if _, err := again.Commit(trustAt(5, 5)); err != nil {
		t.Error(err)
	}
}

func rawTrust(t testing.TB, o Options, d Decision) {
	t.Helper()
	last, n := lastLine(t, o)
	rec := line{V: FormatV1, Seq: n + 1, Prev: hashOf(last), At: stamp(start), Kind: KindDecision, Workspace: d.Workspace, Decision: d.Decision, Trust: lineOf(d.Trust)}
	appendRaw(t, o, append(encode(rec), '\n'))
}

func TestEpochOrderIsolatesWithTheVerifiedPrefix(t *testing.T) {
	for name, bad := range map[string]Decision{"a lower epoch": trustAt(1, 3), "the same epoch": trustAt(2, 3), "another generation": trustAt(1<<32+5, 3)} {
		t.Run(name, func(t *testing.T) {
			o := options(t)
			l := must(t, o)
			l.Commit(trustAt(1, 1))
			l.Commit(trustAt(2, 2))
			l.Close()
			rawTrust(t, o, bad)
			rawTrust(t, o, trustAt(9, 9))
			iso := must(t, o)
			defer iso.Close()
			st := iso.Status()
			if st.State != StateIsolated || st.Isolation != IsolationOrder || st.ProblemLine != 4 {
				t.Fatalf("%+v", st)
			}
			v := iso.Verified()
			if len(v) != 2 || v[1].Decision.Trust.NewEpoch != 2 {
				t.Errorf("verified %+v", v)
			}
			failedWith(t, "commit while isolated", second(iso.Commit(trustAt(3, 3))), ReasonIsolated)
		})
	}
	o := options(t)
	l := must(t, o)
	l.Commit(trustAt(1, 1))
	l.Close()
	last, n := lastLine(t, o)
	rec := line{V: FormatV1, Seq: n + 1, Prev: hashOf(last), At: stamp(start), Kind: KindDecision, Workspace: "sample-workspace", Decision: TrustActivation, Trust: lineOf(trustAt(1, 2).Trust)}
	appendRaw(t, o, encode(rec))
	tail := must(t, o)
	defer tail.Close()
	if st := tail.Status(); st.State != StateIsolated || st.Isolation != IsolationOrder || st.ProblemLine != 3 || len(tail.Verified()) != 1 {
		t.Errorf("an unordered confirmed tail %+v", st)
	}
}

func TestGenerationInMarkerAndGenesis(t *testing.T) {
	o := options(t)
	l := must(t, o)
	if st := l.Status(); st.Generation != 0 {
		t.Errorf("first start %+v", st)
	}
	l.Close()
	marker := filepath.Join(o.Dir, MarkerName)
	data, _ := os.ReadFile(marker)
	if strings.Contains(string(data), `"g"`) || bytes.Contains(lines(t, o)[0], []byte(`"g"`)) {
		t.Errorf("generation 0 is written %s", data)
	}
	parts := lines(t, o)
	parts[0] = []byte("garbage\n")
	rewrite(t, o, parts)
	o.AcceptNew = true
	o.Successor = func(p Prior) (Successor, error) {
		if p.Isolation != IsolationIntegrity || p.ProblemLine != 1 || !p.MarkerUsable || p.LedgerID != "" {
			t.Errorf("prior %+v", p)
		}
		next, ok := p.Next()
		if !ok || next != 1 {
			t.Errorf("next %d %v", next, ok)
		}
		return Successor{Generation: 1, TrustReset: true, ReplacedLine: p.ProblemLine}, nil
	}
	first := must(t, o)
	st := first.Status()
	if st.Generation != 1 || !st.Genesis.TrustReset || st.Genesis.ReplacedLine != 1 || st.Genesis.Generation != 1 {
		t.Errorf("replaced %+v", st)
	}
	first.Close()
	data, _ = os.ReadFile(marker)
	if !strings.Contains(string(data), `"g":1`) || !bytes.Contains(lines(t, o)[0], []byte(`"g":1,"trust_reset":true,"replaced_line":1`)) {
		t.Errorf("marker %s genesis %s", data, lines(t, o)[0])
	}
	o.AcceptNew, o.Successor = false, nil
	again := must(t, o)
	if st := again.Status(); st.Generation != 1 || !st.Genesis.TrustReset || st.State != StateOK {
		t.Errorf("restart %+v", st)
	}
	if _, err := again.Commit(trustAt(1<<32+1, 1)); err != nil {
		t.Error(err)
	}
	failedWith(t, "an epoch of generation 0", second(again.Commit(trustAt(5, 2))), ReasonCommitFailed)
	again.Close()
	os.Remove(marker)
	unmarked := must(t, o)
	if st := unmarked.Status(); st.Generation != 1 || st.State != StateOK {
		t.Errorf("without marker %+v", st)
	}
	unmarked.Close()
	data, _ = os.ReadFile(marker)
	if !strings.Contains(string(data), `"g":1`) {
		t.Errorf("rewritten marker %s", data)
	}
	os.WriteFile(marker, []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"x","g":4}`), 0o600)
	mismatch := must(t, o)
	if st := mismatch.Status(); st.State != StateIsolated || st.Isolation != IsolationMismatch || st.Generation != 4 {
		t.Errorf("mismatch %+v", st)
	}
	if len(mismatch.Verified()) != 1 || len(mismatch.Decisions()) != 0 {
		t.Errorf("a mismatched ledger keeps its verified prefix for trust only %d %d", len(mismatch.Verified()), len(mismatch.Decisions()))
	}
	mismatch.Close()
	os.WriteFile(marker, []byte("not json"), 0o600)
	garbled := must(t, o)
	if st := garbled.Status(); st.State != StateIsolated || st.Generation != 1 {
		t.Errorf("an unreadable marker takes the generation from the genesis %+v", st)
	}
	garbled.Close()
}

func TestSuccessorDecides(t *testing.T) {
	setupReplaced := func(t *testing.T) Options {
		o := options(t)
		l := must(t, o)
		l.Commit(trustAt(1, 1))
		l.Commit(trustAt(2, 4))
		l.Close()
		rawTrust(t, o, trustAt(1, 5))
		o.AcceptNew = true
		return o
	}
	o := setupReplaced(t)
	listingBefore := listing(t, o.Dir)
	refuse := errors.New("refused by the operator rule")
	o.Successor = func(Prior) (Successor, error) { return Successor{}, refuse }
	if _, err := Open(o); !errors.Is(err, refuse) {
		t.Errorf("a refused successor %v", err)
	}
	if listing(t, o.Dir) != listingBefore {
		t.Error("a refused successor changed files")
	}
	o.Successor = func(Prior) (Successor, error) { return Successor{Generation: 0}, nil }
	if _, err := Open(o); !errors.Is(err, ErrInvalid) {
		t.Errorf("a generation at the floor %v", err)
	}
	var seen Prior
	o.Successor = func(p Prior) (Successor, error) {
		seen = p
		carried := *p.Verified[len(p.Verified)-1].Decision.Trust
		carried.NewEpoch, carried.InvalidateBelow, carried.OldEpoch = 1<<32+1, 1<<32+1, 2
		carried.Origin, carried.CarriedFrom, carried.CarriedLine = OriginCarried, p.LedgerID+":3", p.ProblemLine
		return Successor{Generation: 1, CarriedFrom: p.LedgerID, ReplacedLine: p.ProblemLine,
			Carry: []Decision{{Workspace: "sample-workspace", Decision: TrustActivation, Trust: &carried}}}, nil
	}
	l := must(t, o)
	if seen.Isolation != IsolationOrder || seen.ProblemLine != 4 || len(seen.Verified) != 2 || !seen.MarkerUsable || seen.MarkerID != seen.LedgerID {
		t.Errorf("prior %+v", seen)
	}
	if floor, known := seen.Floor(); !known || floor != 0 {
		t.Errorf("floor %d %v", floor, known)
	}
	st := l.Status()
	got := l.Decisions()
	if st.Generation != 1 || st.Genesis.CarriedFrom != seen.LedgerID || len(got) != 1 || got[0].Decision.Trust.Origin != OriginCarried || got[0].Decision.Trust.NewPolicy != 4 {
		t.Fatalf("carried %+v %+v", st, got)
	}
	failedWith(t, "the carried epoch again", second(l.Commit(trustAt(1<<32+1, 5))), ReasonCommitFailed)
	if _, err := l.Commit(trustAt(1<<32+2, 5)); err != nil {
		t.Error(err)
	}
	l.Close()
	os.WriteFile(ledgerPath(o), []byte("x\n"), 0o600)
	os.WriteFile(filepath.Join(o.Dir, MarkerName), []byte("{"), 0o600)
	o.Successor = func(p Prior) (Successor, error) {
		seen = p
		return Successor{Generation: 7, GenerationStated: true, TrustReset: true}, nil
	}
	o.Clock = func() time.Time { return start.Add(1) }
	stated := must(t, o)
	defer stated.Close()
	if seen.MarkerUsable || seen.LedgerID != "" || len(seen.Retained) != 1 || !seen.Retained[0].GenesisRead || seen.Retained[0].Generation != 0 || seen.Retained[0].HighestEpoch != 2 {
		t.Errorf("prior with retained %+v", seen)
	}
	if floor, known := seen.Floor(); !known || floor != 1 {
		t.Errorf("a retained generation 0 proves a successor of generation 1: floor %d %v", floor, known)
	}
	if _, ok := seen.Next(); ok {
		t.Error("next without marker or genesis")
	}
	if st := stated.Status(); st.Generation != 7 || !st.Genesis.GenerationStated {
		t.Errorf("stated %+v", st)
	}
}

func TestDefaultSuccessorAndFloors(t *testing.T) {
	o := options(t)
	existing(t, &o)
	os.WriteFile(filepath.Join(o.Dir, MarkerName), []byte(`{"ledger_id":"`+strings.Repeat("b", 32)+`","created_at":"x","g":3}`), 0o600)
	o.AcceptNew = true
	l := must(t, o)
	if st := l.Status(); st.Generation != 4 {
		t.Errorf("default successor %+v", st)
	}
	l.Close()
	p := Prior{Retained: []Retained{{File: "a", HighestEpoch: 9 << 32}, {File: "b", GenesisRead: true, Generation: 2}}}
	if floor, known := p.Floor(); floor != 10 || !known {
		t.Errorf("floor %d", floor)
	}
	if floor, known := (Prior{Verified: []Record{{Decision: trustAt(7<<32+1, 1)}, {Decision: sample("1")}}}).Floor(); !known || floor != 7 {
		t.Errorf("a verified trust record raises the floor %d", floor)
	}
	if _, known := (Prior{Retained: []Retained{{File: "x"}}}).Floor(); known {
		t.Error("an unreadable retained file has a floor")
	}
	if _, known := (Prior{}).Floor(); known {
		t.Error("an empty prior has a floor")
	}
	if next, ok := (Prior{LedgerID: "x", Generation: 5}).Next(); !ok || next != 6 {
		t.Errorf("next from genesis %d", next)
	}
	empty := options(t)
	os.Mkdir(filepath.Join(empty.Dir, DirName), 0o700)
	os.WriteFile(filepath.Join(empty.Dir, MarkerName), []byte("{"), 0o600)
	empty.AcceptNew = true
	m := must(t, empty)
	defer m.Close()
	if st := m.Status(); st.Generation != 1 || st.ReplaceReason != string(IsolationMissing) {
		t.Errorf("nothing known %+v", st)
	}
}

func second[T any](_ T, err error) error { return err }

func TestACarryThatCannotBeWrittenFailsClosed(t *testing.T) {
	o := options(t)
	existing(t, &o)
	os.WriteFile(ledgerPath(o), []byte("x\n"), 0o600)
	o.AcceptNew = true
	o.Successor = func(p Prior) (Successor, error) {
		return Successor{Generation: 1, Carry: []Decision{trustAt(5, 1)}}, nil
	}
	if _, err := Open(o); err == nil {
		t.Fatal("a carry outside the generation was accepted")
	}
	o.AcceptNew, o.Successor = false, nil
	l := must(t, o)
	defer l.Close()
	if st := l.Status(); st.State != StateIsolated || st.Isolation != IsolationMismatch {
		t.Errorf("after a failed carry %+v", st)
	}
	l.Close()
	for n := 2; n <= 3; n++ {
		o.AcceptNew = true
		o.Clock = func() time.Time { return start.Add(time.Duration(n) * time.Hour) }
		r := must(t, o)
		r.Close()
		os.WriteFile(ledgerPath(o), []byte("y\n"), 0o600)
	}
	o.Successor = func(p Prior) (Successor, error) {
		if len(p.Retained) != 3 || p.Retained[0].File >= p.Retained[1].File || p.Retained[1].File >= p.Retained[2].File {
			t.Errorf("retained %+v", p.Retained)
		}
		return Successor{Generation: 9}, nil
	}
	last := must(t, o)
	last.Close()
}

func TestFitsAndInjectedSyncFailure(t *testing.T) {
	if !Fits(trustAt(5, 5)) {
		t.Error("an ordinary record does not fit")
	}
	big := trustAt(5, 5)
	for n := 0; n < 64; n++ {
		big.Trust.Config.StepUp = append(big.Trust.Config.StepUp, StepRule{Action: "action-" + strings.Repeat("x", 20), MaxAgeSeconds: 60})
	}
	bad := trustAt(5, 5)
	bad.Trust.ChangeDigest = "x"
	if Fits(big) || Fits(bad) {
		t.Error("an oversized or misshaped record fits")
	}
	o := options(t)
	fail := false
	o.FailSync = func() bool { return fail }
	l := must(t, o)
	defer l.Close()
	if _, err := l.Commit(trustAt(1, 1)); err != nil {
		t.Fatal(err)
	}
	fail = true
	failedWith(t, "an injected sync failure", second(l.Commit(trustAt(2, 2))), ReasonCommitUnknown)
	if st := l.Status(); st.State != StateStopped {
		t.Errorf("%+v", st)
	}
	if n := len(lines(t, o)); n != 4 {
		t.Errorf("the record was written in full before the sync failed: %d", n)
	}
}

func TestADuplicateActivationIsolatesAndCanBeReplaced(t *testing.T) {
	o := options(t)
	l := must(t, o)
	first := trustAt(1, 1)
	if _, err := l.Commit(first); err != nil {
		t.Fatal(err)
	}
	again := trustAt(2, 2)
	again.Trust.Change = first.Trust.Change
	failedWith(t, "a second activation of the same change", second(l.Commit(again)), ReasonCommitFailed)
	l.Close()
	rawTrust(t, o, again)
	iso := must(t, o)
	if st := iso.Status(); st.State != StateIsolated || st.Isolation != IsolationOrder || st.ProblemLine != 3 || len(iso.Verified()) != 1 {
		t.Fatalf("a duplicate change isolates at its line %+v", st)
	}
	iso.Close()
	o.AcceptNew = true
	o.Successor = func(p Prior) (Successor, error) {
		carried := *p.Verified[0].Decision.Trust
		carried.OldEpoch, carried.NewEpoch, carried.InvalidateBelow = carried.NewEpoch, 1<<32+1, 1<<32+1
		carried.Origin, carried.CarriedFrom, carried.CarriedLine = OriginCarried, p.LedgerID+":2", p.ProblemLine
		return Successor{Generation: 1, CarriedFrom: p.LedgerID, ReplacedLine: p.ProblemLine, Carry: []Decision{{Workspace: "sample-workspace", Decision: TrustActivation, Trust: &carried}}}, nil
	}
	fresh := must(t, o)
	defer fresh.Close()
	if st := fresh.Status(); st.State != StateOK || st.Generation != 1 || len(fresh.Decisions()) != 1 {
		t.Errorf("accept new after a duplicate %+v", st)
	}
}

func TestTheUnverifiedCurrentFileRaisesTheFloor(t *testing.T) {
	o := options(t)
	l := must(t, o)
	l.Commit(trustAt(1, 1))
	l.Close()
	parts := lines(t, o)
	parts[0] = []byte("garbage\n")
	rewrite(t, o, parts)
	os.WriteFile(filepath.Join(o.Dir, MarkerName), []byte("{"), 0o600)
	o.AcceptNew = true
	var seen Prior
	o.Successor = func(p Prior) (Successor, error) {
		seen = p
		return Successor{Generation: 5, GenerationStated: true, TrustReset: true}, nil
	}
	r := must(t, o)
	r.Close()
	if !seen.UnverifiedRead || seen.Unverified != 0 || seen.LedgerID != "" {
		t.Errorf("unverified scan %+v", seen)
	}
	if floor, known := seen.Floor(); !known || floor != 0 {
		t.Errorf("floor %d %v", floor, known)
	}
	g := Prior{Unverified: 4, UnverifiedRead: true}
	if floor, _ := g.Floor(); floor != 4 {
		t.Errorf("an unverified generation 4 %d", floor)
	}
	if h, ok := unverifiedHighest(filepath.Join(o.Dir, "missing"), 1<<20); ok || h != 0 {
		t.Error("a missing file")
	}
	os.WriteFile(filepath.Join(o.Dir, "noise"), []byte("x\n{}\n"+`{"kind":"genesis","g":3}`+"\n"), 0o600)
	if _, ok := unverifiedHighest(filepath.Join(o.Dir, "noise"), 1<<20); ok {
		t.Error("a file without canonical, shaped genesis or trust lines states a generation")
	}
	genesisLine := func(g uint64) []byte {
		return append(encode(line{V: FormatV1, Seq: 1, Prev: genesisPrev, At: stamp(start), Kind: KindGenesis, LedgerID: strings.Repeat("a", 32), CreatedAt: stamp(start), G: g}), '\n')
	}
	os.WriteFile(filepath.Join(o.Dir, "probe"), append(append([]byte("x\n"), genesisLine(3)...), []byte("tail")...), 0o600)
	if h, ok := unverifiedHighest(filepath.Join(o.Dir, "probe"), 1<<20); !ok || h != 3 {
		t.Errorf("genesis generations %d %v", h, ok)
	}
}

func TestGenerationBounds(t *testing.T) {
	genesisLine := func(g uint64) []byte {
		return append(encode(line{V: FormatV1, Seq: 1, Prev: genesisPrev, At: stamp(start), Kind: KindGenesis, LedgerID: strings.Repeat("a", 32), CreatedAt: stamp(start), G: g}), '\n')
	}
	for _, c := range []struct {
		g  uint64
		ok bool
	}{{MaxGeneration - 1, true}, {MaxGeneration, false}, {MaxGeneration + 1, false}, {^uint64(0), false}} {
		o := options(t)
		os.Mkdir(filepath.Join(o.Dir, DirName), 0o700)
		os.WriteFile(ledgerPath(o), genesisLine(c.g), 0o600)
		l := must(t, o)
		st := l.Status()
		l.Close()
		os.WriteFile(filepath.Join(l.dir, FileName+ReplacedTag+"bound"), genesisLine(c.g), 0o600)
		if r := l.retained(); len(r) != 1 || r[0].GenesisRead != c.ok || c.ok && r[0].Generation != c.g {
			t.Errorf("retained g %d: %+v", c.g, r)
		}
		if (st.State != StateIsolated) != c.ok || c.ok && st.Generation != c.g {
			t.Errorf("genesis g %d: %+v", c.g, st)
		}
		probe := filepath.Join(o.Dir, "probe")
		os.WriteFile(probe, genesisLine(c.g), 0o600)
		if h, found := unverifiedHighest(probe, 1<<20); found != c.ok || c.ok && h != c.g {
			t.Errorf("unverified scan g %d: %d %v", c.g, h, found)
		}
		m := options(t)
		existing(t, &m)
		data, _ := os.ReadFile(filepath.Join(m.Dir, MarkerName))
		var mark marker
		json.Unmarshal(data, &mark)
		mark.G = c.g
		raw, _ := json.Marshal(mark)
		os.WriteFile(filepath.Join(m.Dir, MarkerName), raw, 0o600)
		ml := must(t, m)
		if st := ml.Status(); st.MarkerUsable != c.ok {
			t.Errorf("marker g %d: %+v", c.g, st)
		}
		ml.Close()
	}
	crafted := options(t)
	os.Mkdir(filepath.Join(crafted.Dir, DirName), 0o700)
	os.WriteFile(ledgerPath(crafted), []byte(`{"kind":"genesis","g":18446744073709551615}`+"\n"), 0o600)
	os.WriteFile(filepath.Join(crafted.Dir, MarkerName), []byte("{"), 0o600)
	crafted.AcceptNew = true
	var seen Prior
	crafted.Successor = func(p Prior) (Successor, error) {
		seen = p
		return Successor{Generation: 2, GenerationStated: true, TrustReset: true}, nil
	}
	cl := must(t, crafted)
	cl.Close()
	if seen.UnverifiedRead {
		t.Errorf("a crafted non-canonical genesis was counted %+v", seen)
	}
	top := options(t)
	os.Mkdir(filepath.Join(top.Dir, DirName), 0o700)
	os.WriteFile(ledgerPath(top), genesisLine(MaxGeneration-1), 0o600)
	os.WriteFile(filepath.Join(top.Dir, MarkerName), []byte("{"), 0o600)
	top.AcceptNew = true
	if _, err := Open(top); !errors.Is(err, ErrGenerations) {
		t.Errorf("the default successor past the last generation %v", err)
	}
	top.Successor = func(Prior) (Successor, error) {
		return Successor{Generation: MaxGeneration, GenerationStated: true}, nil
	}
	if _, err := Open(top); !errors.Is(err, ErrGenerations) {
		t.Errorf("a stated successor at the bound %v", err)
	}
	high := trustAt(MaxGeneration<<32+1, 1)
	if Fits(high) {
		t.Error("a trust record of an unbounded generation is shaped")
	}
	d := trustAt(5<<32+1, 1)
	trustLine := encode(line{V: FormatV1, Seq: 2, Prev: genesisPrev, At: stamp(start), Kind: KindDecision, Workspace: d.Workspace, Decision: d.Decision, Trust: lineOf(d.Trust)})
	probe := filepath.Join(t.TempDir(), "mixed")
	os.WriteFile(probe, append(genesisLine(2), append(trustLine, '\n')...), 0o600)
	if h, found := unverifiedHighest(probe, 1<<20); !found || h != 5 {
		t.Errorf("a trust record above the genesis generation %d %v", h, found)
	}
	big := options(t)
	os.Mkdir(filepath.Join(big.Dir, DirName), 0o700)
	huge := append(append(bytes.Repeat([]byte("y"), LineLimit+10), '\n'), genesisLine(9)...)
	os.WriteFile(ledgerPath(big), huge, 0o600)
	if h, found := unverifiedHighest(ledgerPath(big), 1<<20); !found || h != 9 {
		t.Errorf("an over-long line is skipped %d %v", h, found)
	}
	if _, found := unverifiedHighest(ledgerPath(big), LineLimit); found {
		t.Error("the scan reads past its byte limit")
	}
	os.WriteFile(ledgerPath(big), bytes.Repeat([]byte("z"), LineLimit+10), 0o600)
	if _, found := unverifiedHighest(ledgerPath(big), 1<<20); found {
		t.Error("an over-long final line")
	}
}
