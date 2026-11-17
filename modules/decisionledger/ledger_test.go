package decisionledger

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

func options(t testing.TB) Options {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return Options{Dir: dir, MaxBytes: 1 << 20, Clock: func() time.Time { return start }, Random: rand.Reader}
}

func must(t testing.TB, o Options) *Ledger {
	t.Helper()
	l, err := Open(o)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func sample(n string) Decision {
	return Decision{Workspace: "sample-workspace", Decision: RevokeGrant, Target: "grant-" + n, TargetVersion: 1, Revocation: "revocation-" + n,
		Request: "access-request-" + n, RequestDigest: strings.Repeat("a", 64), RequestedBy: "user-a", ApprovedBy: "user-b", DenyGeneration: 7}
}

func commit(t testing.TB, l *Ledger, n string) Receipt {
	t.Helper()
	r, err := l.Commit(sample(n))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func ledgerPath(o Options) string { return filepath.Join(o.Dir, DirName, FileName) }

func lines(t testing.TB, o Options) [][]byte {
	t.Helper()
	data, err := os.ReadFile(ledgerPath(o))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.SplitAfter(data, []byte("\n"))
}

func rewrite(t testing.TB, o Options, parts [][]byte) {
	t.Helper()
	if err := os.WriteFile(ledgerPath(o), bytes.Join(parts, nil), 0o600); err != nil {
		t.Fatal(err)
	}
}

func listing(t testing.TB, dir string) string {
	t.Helper()
	out := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, _ := d.Info()
		out = append(out, path+" "+info.ModTime().String()+" "+info.Mode().String())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}

func TestFirstStartCreatesGenesisAndMarker(t *testing.T) {
	o := options(t)
	l := must(t, o)
	defer l.Close()
	st := l.Status()
	if st.State != StateOK || st.NotDurable || st.Records != 1 || len(st.LedgerID) != 32 || !st.CreatedAt.Equal(start) || st.File != RelativeFile || !strings.HasPrefix(st.Watermark, "1:") ||
		st.Replaces != "" || st.Recovery != nil || !st.CheckedAt.Equal(start) {
		t.Fatalf("status %+v", st)
	}
	data, _ := os.ReadFile(filepath.Join(o.Dir, MarkerName))
	var m marker
	if json.Unmarshal(data, &m) != nil || m.LedgerID != st.LedgerID || m.CreatedAt != stamp(start) {
		t.Errorf("marker %s", data)
	}
	for path, want := range map[string]fs.FileMode{ledgerPath(o): 0o600, filepath.Join(o.Dir, MarkerName): 0o600, filepath.Join(o.Dir, DirName): 0o700 | fs.ModeDir} {
		if info, err := os.Lstat(path); err != nil || info.Mode() != want {
			t.Errorf("%s mode %v %v", path, info.Mode(), err)
		}
	}
	first := lines(t, o)[0]
	if !bytes.Contains(first, []byte(`"prev":"`+genesisPrev+`"`)) || !bytes.Contains(first, []byte(`"kind":"genesis"`)) || st.Watermark != "1:"+hashOf(bytes.TrimSuffix(first, []byte("\n"))) {
		t.Errorf("genesis %s", first)
	}
	if len(l.Decisions()) != 0 {
		t.Error("decisions on a new ledger")
	}
}

func TestCommitAndRestartReread(t *testing.T) {
	o := options(t)
	l := must(t, o)
	r1 := commit(t, l, "1")
	ended := sample("2")
	ended.Decision, ended.Request, ended.RequestDigest, ended.ApprovedBy = RevokeDelegation, "", "", ""
	ended.EndedDelegations = []string{"del-2", "del-3"}
	r2, err := l.Commit(ended)
	if err != nil || r1.Sequence != 2 || r2.Sequence != 3 || r2.Watermark == r1.Watermark || r2.Watermark != l.Status().Watermark {
		t.Fatalf("receipts %+v %+v %v", r1, r2, err)
	}
	before := l.Status()
	if live := l.Decisions(); live[0].Watermark != r1.Watermark || live[1].Watermark != r2.Watermark {
		t.Error("live watermarks")
	}
	got := l.Decisions()
	got[1].Decision.EndedDelegations[0] = "changed"
	if l.Decisions()[1].Decision.EndedDelegations[0] != "del-2" {
		t.Error("decisions alias the ledger")
	}
	if err := l.Close(); err != nil || l.Close() != nil {
		t.Fatal("close")
	}
	if _, err := l.Commit(sample("x")); err == nil {
		t.Error("commit after close")
	}
	again := must(t, o)
	defer again.Close()
	after := again.Status()
	if after.Records != before.Records || after.LedgerID != before.LedgerID || after.Watermark != before.Watermark || after.State != StateOK {
		t.Fatalf("reread %+v vs %+v", after, before)
	}
	decisions := again.Decisions()
	if len(decisions) != 2 || decisions[0].Sequence != 2 || decisions[0].Watermark != r1.Watermark || decisions[1].Watermark != r2.Watermark || decisions[0].Decision.Target != "grant-1" || !decisions[0].At.Equal(start) ||
		strings.Join(decisions[1].Decision.EndedDelegations, ",") != "del-2,del-3" || decisions[1].Decision.Decision != RevokeDelegation {
		t.Errorf("decisions %+v", decisions)
	}
	if r := commit(t, again, "3"); r.Sequence != 4 {
		t.Errorf("sequence continues %+v", r)
	}
	raw, _ := os.ReadFile(ledgerPath(o))
	for _, forbidden := range []string{"sample reason", "@example"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Errorf("ledger holds %s", forbidden)
		}
	}
}

func TestLedgerWithoutMarkerWritesTheMarker(t *testing.T) {
	o := options(t)
	l := must(t, o)
	id := l.Status().LedgerID
	commit(t, l, "1")
	l.Close()
	if err := os.Remove(filepath.Join(o.Dir, MarkerName)); err != nil {
		t.Fatal(err)
	}
	again := must(t, o)
	defer again.Close()
	data, _ := os.ReadFile(filepath.Join(o.Dir, MarkerName))
	if again.Status().State != StateOK || !strings.Contains(string(data), id) {
		t.Errorf("marker %s %+v", data, again.Status())
	}
}

func TestIsolationTouchesNoFile(t *testing.T) {
	for name, damage := range map[string]func(t *testing.T, o Options){
		"missing": func(t *testing.T, o Options) { os.Remove(ledgerPath(o)) },
		"integrity": func(t *testing.T, o Options) {
			parts := lines(t, o)
			parts[1] = bytes.Replace(parts[1], []byte("grant-1"), []byte("grant-9"), 1)
			rewrite(t, o, parts)
		},
		"mismatch": func(t *testing.T, o Options) {
			os.WriteFile(filepath.Join(o.Dir, MarkerName), []byte(`{"ledger_id":"`+strings.Repeat("b", 32)+`","created_at":"x"}`), 0o600)
		},
		"undecodable marker": func(t *testing.T, o Options) {
			os.WriteFile(filepath.Join(o.Dir, MarkerName), []byte(`nope`), 0o600)
		},
	} {
		t.Run(name, func(t *testing.T) {
			o := options(t)
			l := must(t, o)
			commit(t, l, "1")
			commit(t, l, "2")
			l.Close()
			damage(t, o)
			before := listing(t, o.Dir)
			iso := must(t, o)
			st := iso.Status()
			want := map[string]Isolation{"missing": IsolationMissing, "integrity": IsolationIntegrity, "mismatch": IsolationMismatch, "undecodable marker": IsolationMismatch}[name]
			if st.State != StateIsolated || st.Isolation != want || len(iso.Decisions()) != 0 {
				t.Fatalf("status %+v", st)
			}
			if name == "integrity" && st.ProblemLine != 2 {
				t.Errorf("problem line %d", st.ProblemLine)
			}
			for i := 0; i < 2; i++ {
				if _, err := iso.Commit(sample("3")); err == nil || err.(*CommitError).Reason != ReasonIsolated {
					t.Errorf("commit in isolation %v", err)
				}
			}
			iso.Close()
			if after := listing(t, o.Dir); after != before {
				t.Errorf("isolation touched files\n%s\n%s", before, after)
			}
			if iso.Status().State != StateIsolated {
				t.Error("close changed the isolation state")
			}
		})
	}
}

func TestAcceptNewLedger(t *testing.T) {
	o := options(t)
	l := must(t, o)
	old := l.Status().LedgerID
	commit(t, l, "1")
	l.Close()
	parts := lines(t, o)
	parts[1] = bytes.Replace(parts[1], []byte(`"seq":2`), []byte(`"seq":3`), 1)
	rewrite(t, o, parts)
	o.AcceptNew = true
	o.Clock = func() time.Time { return start.Add(time.Hour) }
	fresh := must(t, o)
	st := fresh.Status()
	if st.State != StateOK || st.Replaces != old || st.ReplaceReason != string(IsolationIntegrity) || st.LedgerID == old || st.Records != 1 ||
		st.ReplacedFile != FileName+ReplacedTag+"20261118T100000.000000000Z" || st.Isolation != "" {
		t.Fatalf("replaced %+v", st)
	}
	if kept, err := os.ReadFile(filepath.Join(o.Dir, DirName, st.ReplacedFile)); err != nil || !bytes.Equal(kept, bytes.Join(parts, nil)) {
		t.Error("the replaced file is not kept")
	}
	data, _ := os.ReadFile(filepath.Join(o.Dir, MarkerName))
	if !strings.Contains(string(data), st.LedgerID) {
		t.Error("marker")
	}
	if !bytes.Contains(lines(t, o)[0], []byte(`"replaces":"`+old+`","replace_reason":"integrity_failed"`)) {
		t.Errorf("genesis %s", lines(t, o)[0])
	}
	fresh.Close()
	if _, err := Open(o); !errors.Is(err, ErrNothingToReplace) {
		t.Errorf("the flag left on %v", err)
	}
	o.AcceptNew = false
	again := must(t, o)
	if again.Status().LedgerID != st.LedgerID || again.Status().Replaces != old {
		t.Errorf("restart after replacement %+v", again.Status())
	}
	again.Close()
	os.Remove(ledgerPath(o))
	o.AcceptNew = true
	missing := must(t, o)
	defer missing.Close()
	if st := missing.Status(); st.Replaces != UnknownLedger || st.ReplaceReason != string(IsolationMissing) || st.ReplacedFile != "" {
		t.Errorf("replacing a missing ledger %+v", st)
	}
	fresh2 := options(t)
	fresh2.AcceptNew = true
	if _, err := Open(fresh2); !errors.Is(err, ErrNothingToReplace) {
		t.Errorf("first start with the flag %v", err)
	}
	if _, err := os.Lstat(ledgerPath(fresh2)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("a refused first start created a ledger")
	}
}

func TestTornTailRecovery(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "1")
	l.Close()
	full, _ := os.ReadFile(ledgerPath(o))
	f, _ := os.OpenFile(ledgerPath(o), os.O_WRONLY|os.O_APPEND, 0o600)
	f.Write([]byte(`{"v":1,"seq":3,"prev":"`))
	f.Close()
	o.Clock = func() time.Time { return start.Add(time.Minute) }
	again := must(t, o)
	st := again.Status()
	if st.State != StateRecovered || st.Recovery == nil || st.Recovery.TruncatedBytes != 23 || !st.Recovery.At.Equal(start.Add(time.Minute)) || st.Records != 3 {
		t.Fatalf("recovered %+v", st)
	}
	st.Recovery.TruncatedBytes = 0
	if again.Status().Recovery.TruncatedBytes != 23 {
		t.Error("status aliases the recovery")
	}
	parts := lines(t, o)
	if !bytes.HasPrefix(bytes.Join(parts, nil), full) || !bytes.Contains(parts[2], []byte(`"kind":"recovery"`)) || !bytes.Contains(parts[2], []byte(`"truncated_bytes":23`)) {
		t.Errorf("recovery record %s", parts[2])
	}
	if r := commit(t, again, "2"); r.Sequence != 4 {
		t.Errorf("after recovery %+v", r)
	}
	again.Close()
	if next := must(t, o); next.Status().State != StateOK || next.Status().Records != 4 {
		t.Errorf("next start %+v", next.Status())
	} else {
		next.Close()
	}
}

func TestTornTailAtTheBoundStartsFull(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "1")
	l.Close()
	size := int64(len(bytes.Join(lines(t, o), nil)))
	f, _ := os.OpenFile(ledgerPath(o), os.O_WRONLY|os.O_APPEND, 0o600)
	f.Write([]byte(`{"v":`))
	f.Close()
	o.MaxBytes = size + RecordLimit - 1
	again := must(t, o)
	defer again.Close()
	st := again.Status()
	if st.State != StateFull || st.Recovery == nil || st.Recovery.TruncatedBytes != 5 || st.Records != 2 {
		t.Fatalf("full with recovery %+v", st)
	}
	if info, _ := os.Stat(ledgerPath(o)); info.Size() != size {
		t.Error("the torn tail was not truncated")
	}
	if _, err := again.Commit(sample("2")); err.(*CommitError).Reason != ReasonFull {
		t.Errorf("commit when full %v", err)
	}
}

func TestEmptyLedger(t *testing.T) {
	o := options(t)
	l := must(t, o)
	l.Close()
	os.WriteFile(ledgerPath(o), []byte(`{"v":1`), 0o600)
	os.Remove(filepath.Join(o.Dir, MarkerName))
	again := must(t, o)
	if st := again.Status(); st.State != StateOK || st.Records != 1 {
		t.Errorf("an interrupted first start %+v", st)
	}
	again.Close()
	os.WriteFile(ledgerPath(o), nil, 0o600)
	iso := must(t, o)
	defer iso.Close()
	if st := iso.Status(); st.State != StateIsolated || st.Isolation != IsolationIntegrity || st.ProblemLine != 1 {
		t.Errorf("an empty ledger with a marker %+v", st)
	}
}

func TestCorruptionsAreFoundWithTheirLine(t *testing.T) {
	type damage func(parts [][]byte) [][]byte
	base := func(t *testing.T) (Options, [][]byte) {
		o := options(t)
		l := must(t, o)
		for _, n := range []string{"1", "2", "3"} {
			commit(t, l, n)
		}
		l.Close()
		return o, lines(t, o)
	}
	edit := func(i int, from, to string) damage {
		return func(p [][]byte) [][]byte { p[i] = bytes.Replace(p[i], []byte(from), []byte(to), 1); return p }
	}
	for name, c := range map[string]struct {
		change damage
		line   int
	}{
		"byte flip":         {edit(2, "grant-2", "grant-x"), 3},
		"deleted line":      {func(p [][]byte) [][]byte { return append(p[:1], p[2:]...) }, 2},
		"reordered":         {func(p [][]byte) [][]byte { p[1], p[2] = p[2], p[1]; return p }, 2},
		"duplicated":        {func(p [][]byte) [][]byte { return append(p[:3], append([][]byte{p[2]}, p[3:]...)...) }, 4},
		"unknown field":     {edit(1, `"v":1,`, `"v":1,"x":1,`), 2},
		"missing genesis":   {func(p [][]byte) [][]byte { return p[1:] }, 1},
		"misplaced genesis": {func(p [][]byte) [][]byte { return append(p[:2], append([][]byte{p[0]}, p[2:]...)...) }, 3},
		"oversize":          {func(p [][]byte) [][]byte { return append(p, []byte(strings.Repeat("x", LineLimit+1)+"\n")) }, 5},
		"altered last line": {edit(3, "grant-3", "grant-y"), 4},
		"spacing":           {edit(3, `"v":1,`, `"v": 1,`), 4},
		"wrong format":      {edit(3, `"v":1,`, `"v":2,`), 4},
		"not json":          {edit(3, `{`, `[`), 4},
	} {
		t.Run(name, func(t *testing.T) {
			o, parts := base(t)
			rewrite(t, o, c.change(parts))
			l := must(t, o)
			defer l.Close()
			if st := l.Status(); st.State != StateIsolated || st.Isolation != IsolationIntegrity || st.ProblemLine != c.line {
				t.Errorf("%+v", st)
			}
		})
	}
}

func TestRecordShapes(t *testing.T) {
	ok := line{Kind: KindDecision, Workspace: "w", Decision: RevokeGrant, Target: "g", Revocation: "r", RequestedBy: "u"}
	if !ok.shaped() {
		t.Fatal("decision shape")
	}
	for name, l := range map[string]line{
		"decision with genesis fields":  {Kind: KindDecision, Workspace: "w", Decision: RevokeGrant, Target: "g", Revocation: "r", RequestedBy: "u", LedgerID: "x"},
		"decision with recovery fields": {Kind: KindDecision, Workspace: "w", Decision: RevokeGrant, Target: "g", Revocation: "r", RequestedBy: "u", TruncatedBytes: 1},
		"decision without workspace":    {Kind: KindDecision, Decision: RevokeGrant, Target: "g", Revocation: "r", RequestedBy: "u"},
		"unknown decision":              {Kind: KindDecision, Workspace: "w", Decision: "other", Target: "g", Revocation: "r", RequestedBy: "u"},
		"genesis with a decision field": {Kind: KindGenesis, LedgerID: strings.Repeat("a", 32), CreatedAt: stamp(start), Target: "g"},
		"genesis with a bad id":         {Kind: KindGenesis, LedgerID: strings.Repeat("A", 32), CreatedAt: stamp(start)},
		"genesis with half a replace":   {Kind: KindGenesis, LedgerID: strings.Repeat("a", 32), CreatedAt: stamp(start), Replaces: "x"},
		"genesis with recovery fields":  {Kind: KindGenesis, LedgerID: strings.Repeat("a", 32), CreatedAt: stamp(start), RecoveredAt: stamp(start)},
		"recovery without bytes":        {Kind: KindRecovery, RecoveredAt: stamp(start)},
		"recovery with a genesis field": {Kind: KindRecovery, TruncatedBytes: 1, RecoveredAt: stamp(start), LedgerID: "x"},
		"unknown kind":                  {Kind: "other"},
		"bad time":                      {Kind: KindRecovery, TruncatedBytes: 1, RecoveredAt: "2026-11-18T09:00:00+09:00"},
		"short id":                      {Kind: KindGenesis, LedgerID: "ab", CreatedAt: stamp(start)},
	} {
		if l.shaped() {
			t.Errorf("%s accepted", name)
		}
	}
	if (&CommitError{Reason: ReasonFull}).Error() != "decision not committed: ledger_full" {
		t.Error("error text")
	}
}

func TestBound(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "1")
	l.Close()
	size := int64(len(bytes.Join(lines(t, o), nil)))
	o.MaxBytes = size - 1
	if _, err := Open(o); !errors.Is(err, ErrBound) {
		t.Errorf("a bound below the file %v", err)
	}
	o.MaxBytes = size + RecordLimit - 1
	near := must(t, o)
	if near.Status().State != StateFull {
		t.Errorf("within one record %+v", near.Status())
	}
	near.Close()
	o.MaxBytes = size + RecordLimit + 100
	room := must(t, o)
	defer room.Close()
	if room.Status().State != StateOK {
		t.Fatalf("more than one record of room %+v", room.Status())
	}
	commit(t, room, "2")
	if room.Status().State != StateOK {
		t.Fatal("a fitting record made the ledger full")
	}
	size = int64(len(bytes.Join(lines(t, o), nil)))
	big := sample("3")
	for len(big.EndedDelegations) < 38 {
		big.EndedDelegations = append(big.EndedDelegations, "delegation-with-a-long-identifier")
	}
	if _, err := room.Commit(big); err.(*CommitError).Reason != ReasonFull {
		t.Errorf("past the bound %v", err)
	}
	if after := int64(len(bytes.Join(lines(t, o), nil))); after != size || room.Status().State != StateFull {
		t.Error("a record past the bound was written")
	}
}

func TestRecordLimitAndShape(t *testing.T) {
	o := options(t)
	l := must(t, o)
	defer l.Close()
	big := sample("1")
	for len(big.EndedDelegations) < 100 {
		big.EndedDelegations = append(big.EndedDelegations, "delegation-with-a-long-identifier")
	}
	if _, err := l.Commit(big); err.(*CommitError).Reason != ReasonCommitFailed || l.Status().State != StateOK {
		t.Errorf("an oversize record %v %+v", err, l.Status())
	}
	if _, err := l.Commit(Decision{Workspace: "w"}); err.(*CommitError).Reason != ReasonCommitFailed || l.Status().State != StateOK {
		t.Errorf("an unshaped decision %v", err)
	}
	if l.Status().Records != 1 {
		t.Error("a refused record was written")
	}
}

func TestSecondOpenIsLocked(t *testing.T) {
	o := options(t)
	l := must(t, o)
	defer l.Close()
	before := l.Status().Watermark
	if _, err := Open(o); !errors.Is(err, ErrLocked) {
		t.Errorf("second open %v", err)
	}
	if l.Status().Watermark != before {
		t.Error("the second open changed the ledger")
	}
	commit(t, l, "1")
}

func TestConcurrentCommits(t *testing.T) {
	o := options(t)
	l := must(t, o)
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := []uint64{}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				r, err := l.Commit(sample("c"))
				if err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				seen = append(seen, r.Sequence)
				mu.Unlock()
				l.Status()
				l.Decisions()
			}
		}()
	}
	wg.Wait()
	sort.Slice(seen, func(a, b int) bool { return seen[a] < seen[b] })
	for i, s := range seen {
		if s != uint64(i)+2 {
			t.Fatalf("sequence %d at %d", s, i)
		}
	}
	l.Close()
	again := must(t, o)
	defer again.Close()
	if again.Status().State != StateOK || again.Status().Records != 81 {
		t.Errorf("chain after concurrent commits %+v", again.Status())
	}
}

func TestOptionsAndLocation(t *testing.T) {
	o := options(t)
	for name, change := range map[string]func(*Options){
		"dir":    func(o *Options) { o.Dir = "" },
		"bound":  func(o *Options) { o.MaxBytes = 0 },
		"clock":  func(o *Options) { o.Clock = nil },
		"random": func(o *Options) { o.Random = nil },
	} {
		bad := o
		change(&bad)
		if _, err := Open(bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	missing := o
	missing.Dir = filepath.Join(o.Dir, "absent")
	if _, err := Open(missing); err == nil {
		t.Error("a missing data directory")
	}
	file := o
	file.Dir = filepath.Join(o.Dir, "file")
	os.WriteFile(file.Dir, nil, 0o600)
	if _, err := Open(file); err == nil {
		t.Error("a data directory that is a file")
	}
	marked := options(t)
	os.Mkdir(filepath.Join(marked.Dir, MarkerName), 0o700)
	if _, err := Open(marked); !errors.Is(err, ErrUnsafe) {
		t.Errorf("a marker that is a directory %v", err)
	}
	huge := options(t)
	os.WriteFile(filepath.Join(huge.Dir, MarkerName), bytes.Repeat([]byte("x"), LineLimit+1), 0o600)
	if _, err := Open(huge); !errors.Is(err, ErrUnsafe) {
		t.Errorf("an oversize marker %v", err)
	}
}

func TestAConfirmedFinalRecordIsNeverTruncated(t *testing.T) {
	for name, damage := range map[string]func([]byte) []byte{
		"newline removed":  func(d []byte) []byte { return d[:len(d)-1] },
		"newline replaced": func(d []byte) []byte { d[len(d)-1] = 'x'; return d },
	} {
		t.Run(name, func(t *testing.T) {
			o := options(t)
			l := must(t, o)
			commit(t, l, "1")
			commit(t, l, "2")
			l.Close()
			data, _ := os.ReadFile(ledgerPath(o))
			damaged := damage(append([]byte{}, data...))
			os.WriteFile(ledgerPath(o), damaged, 0o600)
			iso := must(t, o)
			defer iso.Close()
			st := iso.Status()
			if st.State != StateIsolated || st.Isolation != IsolationIntegrity || st.ProblemLine != 3 || st.Recovery != nil {
				t.Fatalf("%+v", st)
			}
			if after, _ := os.ReadFile(ledgerPath(o)); !bytes.Equal(after, damaged) {
				t.Error("a confirmed record was truncated")
			}
			if got := iso.Decisions(); len(got) != 2 || got[0].Decision.Target != "grant-1" || got[1].Decision.Target != "grant-2" {
				t.Errorf("verified prefix %+v", got)
			}
		})
	}
}

func TestIsolationKeepsTheVerifiedPrefix(t *testing.T) {
	o := options(t)
	l := must(t, o)
	for _, n := range []string{"1", "2", "3"} {
		commit(t, l, n)
	}
	l.Close()
	parts := lines(t, o)
	parts[3] = bytes.Replace(parts[3], []byte("grant-3"), []byte("grant-z"), 1)
	rewrite(t, o, parts)
	before := listing(t, o.Dir)
	iso := must(t, o)
	got := iso.Decisions()
	if st := iso.Status(); st.State != StateIsolated || st.ProblemLine != 4 || len(got) != 2 || got[0].Decision.Target != "grant-1" || got[1].Decision.Target != "grant-2" {
		t.Fatalf("%+v %+v", st, got)
	}
	if _, err := iso.Commit(sample("4")); err.(*CommitError).Reason != ReasonIsolated {
		t.Errorf("commit %v", err)
	}
	iso.Close()
	if after := listing(t, o.Dir); after != before {
		t.Error("isolation wrote")
	}
}

func TestAForeignMarkerWithADamagedLineReplaysNothing(t *testing.T) {
	o := options(t)
	l := must(t, o)
	for _, n := range []string{"1", "2", "3"} {
		commit(t, l, n)
	}
	l.Close()
	parts := lines(t, o)
	parts[3] = bytes.Replace(parts[3], []byte("grant-3"), []byte("grant-z"), 1)
	rewrite(t, o, parts)
	os.WriteFile(filepath.Join(o.Dir, MarkerName), []byte(`{"ledger_id":"`+strings.Repeat("c", 32)+`","created_at":"x"}`), 0o600)
	iso := must(t, o)
	defer iso.Close()
	if st := iso.Status(); st.State != StateIsolated || st.Isolation != IsolationMismatch || st.ProblemLine != 0 || len(iso.Decisions()) != 0 {
		t.Errorf("%+v %d", st, len(iso.Decisions()))
	}
	iso.Close()
	parts[0] = []byte("damaged genesis\n")
	rewrite(t, o, parts)
	again := must(t, o)
	defer again.Close()
	if st := again.Status(); st.Isolation != IsolationIntegrity || st.ProblemLine != 1 || len(again.Decisions()) != 0 {
		t.Errorf("a damaged genesis cannot name a ledger %+v", st)
	}
}
