package decisionledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

var hash64 = strings.Repeat("b", 64)

func identitySamples() map[string]Decision {
	ws := "sample-workspace"
	return map[string]Decision{
		BulkRevocationJob: {Workspace: ws, Decision: BulkRevocationJob, Target: "bulk-revocation-1", RequestDigest: hash64, RequestedBy: "user-a", ApprovedBy: "user-b",
			Effect: SuspendMembership, Targets: 3, Held: 1, Excluded: 2, DenyGeneration: 9},
		SuspendMembership: {Workspace: ws, Decision: SuspendMembership, Target: "user-c", TargetVersion: 2, Revocation: "bulk-revocation-1:0", Request: "bulk-revocation-1",
			RequestDigest: hash64, RequestedBy: "user-b", DenyGeneration: 9},
		SuspendUser: {Workspace: ws, Decision: SuspendUser, Target: "user-c", TargetVersion: 1, Revocation: "bulk-revocation-2:0", Request: "bulk-revocation-2",
			RequestDigest: hash64, RequestedBy: "user-b", DenyGeneration: 10},
		RolePeriodEnded: {Workspace: ws, Decision: RolePeriodEnded, Target: "user-c", TargetVersion: 3, Role: "member", Until: "2026-11-18T10:00:00Z",
			Revocation: "role-end-1", RequestedBy: RoleSystem},
		RolePeriodSet: {Workspace: ws, Decision: RolePeriodSet, Target: "user-c", TargetVersion: 3, Role: "administrator", Until: "2026-11-19T10:00:00Z", RequestedBy: "user-a"},
		IncreaseIntent: {Workspace: ws, Decision: IncreaseIntent, Target: "user-c", TargetVersion: 3, Increase: IncreaseRolePeriod, Role: "member", Until: NoEnd,
			RequestedBy: "user-a"},
		IncreaseConfirmed: {Workspace: ws, Decision: IncreaseConfirmed, Target: "user-c", IntentSequence: 1},
		IncreaseAborted:   {Workspace: ws, Decision: IncreaseAborted, Target: "user-c", IntentSequence: 1},
		ClockBaseChoice: {Workspace: ws, Decision: ClockBaseChoice, Clock: &ClockBase{LedgerTime: start.Add(48 * time.Hour), WallTime: start, Choice: ChoiceLedger,
			Offset: 48 * time.Hour}},
	}
}

func TestIdentityKindsAreValidatedPerKind(t *testing.T) {
	samples := identitySamples()
	if len(samples) != len(identityKinds) || !IdentityKind(SuspendUser) || IdentityKind(RevokeGrant) {
		t.Fatalf("%d samples for %d kinds", len(samples), len(identityKinds))
	}
	for kind, d := range samples {
		rec := lineFor(d)
		rec.Seq = 5
		if !rec.shaped() || rec.format() != FormatV2 {
			t.Errorf("%s sample is not shaped", kind)
		}
	}
	bad := map[string]func(*Decision){
		"a trust part":            func(d *Decision) { d.Trust = &Trust{} },
		"ended delegations":       func(d *Decision) { d.EndedDelegations = []string{"x"} },
		"a clock on a non-clock":  func(d *Decision) { d.Clock = &ClockBase{Choice: ChoiceWall} },
		"no target":               func(d *Decision) { d.Target = "" },
		"an unknown role":         func(d *Decision) { d.Role = "owner" },
		"an unknown kind":         func(d *Decision) { d.Decision = "suspend_everything" },
		"a rationale-like reason": func(d *Decision) { d.ApprovedBy = "someone"; d.Request = "r" },
	}
	for kind, d := range samples {
		for name, mutate := range bad {
			if kind == ClockBaseChoice && (name == "a clock on a non-clock" || name == "no target") {
				continue
			}
			copied := d
			mutate(&copied)
			rec := lineFor(copied)
			rec.Seq = 5
			if rec.shaped() {
				t.Errorf("%s with %s is shaped", kind, name)
			}
		}
	}
	specific := map[string]func(*Decision){
		"header without digest":           func(d *Decision) { *d = samples[BulkRevocationJob]; d.RequestDigest = "x" },
		"header without executor":         func(d *Decision) { *d = samples[BulkRevocationJob]; d.ApprovedBy = "" },
		"header with an unknown effect":   func(d *Decision) { *d = samples[BulkRevocationJob]; d.Effect = "remove" },
		"header without targets":          func(d *Decision) { *d = samples[BulkRevocationJob]; d.Targets, d.Held = 0, 0 },
		"header with more held":           func(d *Decision) { *d = samples[BulkRevocationJob]; d.Held = 4 },
		"header with a version":           func(d *Decision) { *d = samples[BulkRevocationJob]; d.TargetVersion = 1 },
		"header with a revocation":        func(d *Decision) { *d = samples[BulkRevocationJob]; d.Revocation = "x" },
		"header with a request":           func(d *Decision) { *d = samples[BulkRevocationJob]; d.Request = "x" },
		"header with an intent":           func(d *Decision) { *d = samples[BulkRevocationJob]; d.IntentSequence = 1 },
		"item without version":            func(d *Decision) { *d = samples[SuspendMembership]; d.TargetVersion = 0 },
		"item without revocation":         func(d *Decision) { *d = samples[SuspendUser]; d.Revocation = "" },
		"item without manifest":           func(d *Decision) { *d = samples[SuspendUser]; d.Request = "" },
		"item with an approver":           func(d *Decision) { *d = samples[SuspendUser]; d.ApprovedBy = "x" },
		"item with counts":                func(d *Decision) { *d = samples[SuspendUser]; d.Targets = 1 },
		"item with an until":              func(d *Decision) { *d = samples[SuspendUser]; d.Until = NoEnd },
		"item without actor":              func(d *Decision) { *d = samples[SuspendUser]; d.RequestedBy = "" },
		"end by a person":                 func(d *Decision) { *d = samples[RolePeriodEnded]; d.RequestedBy = "user-a" },
		"end with no end":                 func(d *Decision) { *d = samples[RolePeriodEnded]; d.Until = NoEnd },
		"end with an unknown role":        func(d *Decision) { *d = samples[RolePeriodEnded]; d.Role = "owner" },
		"end with a request":              func(d *Decision) { *d = samples[RolePeriodEnded]; d.Request = "x" },
		"end without revocation":          func(d *Decision) { *d = samples[RolePeriodEnded]; d.Revocation = "" },
		"end with an increase":            func(d *Decision) { *d = samples[RolePeriodEnded]; d.Increase = IncreaseRolePeriod },
		"set with no end":                 func(d *Decision) { *d = samples[RolePeriodSet]; d.Until = NoEnd },
		"set with a revocation":           func(d *Decision) { *d = samples[RolePeriodSet]; d.Revocation = "x" },
		"set without actor":               func(d *Decision) { *d = samples[RolePeriodSet]; d.RequestedBy = "" },
		"set with an intent":              func(d *Decision) { *d = samples[RolePeriodSet]; d.IntentSequence = 2 },
		"intent of an unknown increase":   func(d *Decision) { *d = samples[IncreaseIntent]; d.Increase = "promote" },
		"intent of a period without role": func(d *Decision) { *d = samples[IncreaseIntent]; d.Role = "" },
		"intent of a period, bad until":   func(d *Decision) { *d = samples[IncreaseIntent]; d.Until = "soon" },
		"intent of a reinstate with role": func(d *Decision) { *d = samples[IncreaseIntent]; d.Increase = IncreaseReinstateUser },
		"intent without version":          func(d *Decision) { *d = samples[IncreaseIntent]; d.TargetVersion = 0 },
		"intent with a sequence":          func(d *Decision) { *d = samples[IncreaseIntent]; d.IntentSequence = 1 },
		"intent with a revocation":        func(d *Decision) { *d = samples[IncreaseIntent]; d.Revocation = "x" },
		"confirm without intent":          func(d *Decision) { *d = samples[IncreaseConfirmed]; d.IntentSequence = 0 },
		"confirm of a later intent":       func(d *Decision) { *d = samples[IncreaseConfirmed]; d.IntentSequence = 5 },
		"confirm with a version":          func(d *Decision) { *d = samples[IncreaseConfirmed]; d.TargetVersion = 1 },
		"confirm with an actor":           func(d *Decision) { *d = samples[IncreaseAborted]; d.RequestedBy = "x" },
		"confirm with a role":             func(d *Decision) { *d = samples[IncreaseAborted]; d.Role = "member" },
		"confirm with a revocation":       func(d *Decision) { *d = samples[IncreaseAborted]; d.Revocation = "x" },
		"clock with a target":             func(d *Decision) { *d = samples[ClockBaseChoice]; d.Target = "x" },
		"clock with an unknown choice": func(d *Decision) {
			*d = samples[ClockBaseChoice]
			d.Clock = &ClockBase{LedgerTime: start, WallTime: start, Choice: "both"}
		},
		"wall clock with an offset": func(d *Decision) {
			*d = samples[ClockBaseChoice]
			d.Clock = &ClockBase{LedgerTime: start, WallTime: start, Choice: ChoiceWall, Offset: 1}
		},
		"clock without times": func(d *Decision) { *d = samples[ClockBaseChoice]; d.Clock = nil },
		"clock with an actor": func(d *Decision) { *d = samples[ClockBaseChoice]; d.RequestedBy = "x" },
	}
	for name, mutate := range specific {
		var d Decision
		mutate(&d)
		rec := lineFor(d)
		rec.Seq = 5
		if rec.shaped() {
			t.Errorf("%s is shaped", name)
		}
	}
	plain := sample("1")
	plain.Role = "member"
	if rec := lineFor(plain); rec.shaped() {
		t.Error("a grant revocation with a role is shaped")
	}
	if rec := lineFor(Decision{}); rec.format() != FormatV1 {
		t.Error("an empty decision is written as the new format")
	}
}

func TestIdentityRecordsAreWrittenAsVersionTwoAndReopen(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "a")
	samples := identitySamples()
	order := []string{IncreaseIntent, IncreaseConfirmed, BulkRevocationJob, SuspendMembership, SuspendUser, RolePeriodEnded, RolePeriodSet, IncreaseAborted, ClockBaseChoice}
	for _, kind := range order {
		d := samples[kind]
		if kind == IncreaseConfirmed || kind == IncreaseAborted {
			d.IntentSequence = 3
		}
		if _, err := l.Commit(d); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	if _, err := l.Commit(samples[IncreaseConfirmed]); err != nil {
		t.Fatalf("a confirm naming an earlier intent: %v", err)
	}
	before := l.Decisions()
	l.Close()
	versions := []int{}
	for _, raw := range lines(t, o) {
		if len(raw) == 0 {
			continue
		}
		var head struct {
			V int `json:"v"`
		}
		json.Unmarshal(raw, &head)
		versions = append(versions, head.V)
	}
	want := []int{1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	if len(versions) != len(want) {
		t.Fatalf("versions %v", versions)
	}
	for i := range want {
		if versions[i] != want[i] {
			t.Fatalf("versions %v", versions)
		}
	}
	again := must(t, o)
	defer again.Close()
	if st := again.Status(); st.State != StateOK || st.Records != 12 {
		t.Fatalf("reopened %+v", st)
	}
	after := again.Decisions()
	if len(after) != len(before) || after[len(after)-2].Decision.Clock == nil || after[len(after)-2].Decision.Clock.Offset != 48*time.Hour ||
		after[3].Decision.Targets != 3 || after[6].Decision.Until != "2026-11-18T10:00:00Z" {
		t.Fatalf("decisions %+v", after)
	}
	if before[len(before)-2].Decision.Clock == nil || !before[len(before)-2].Decision.Clock.LedgerTime.Equal(start.Add(48*time.Hour)) {
		t.Error("the committed clock record is not returned")
	}
}

func TestAWrongVersionIsolatesAndANewerVersionRefusesToStart(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "a")
	l.Close()
	parts := lines(t, o)
	var rec line
	json.Unmarshal(bytes.TrimSuffix(parts[1], []byte("\n")), &rec)
	rec.V = FormatV2
	rewrite(t, o, [][]byte{parts[0], append(encode(rec), '\n')})
	wrong := must(t, o)
	if st := wrong.Status(); st.State != StateIsolated {
		t.Errorf("a version two grant revocation %+v", st)
	}
	wrong.Close()
	o = options(t)
	l = must(t, o)
	commit(t, l, "a")
	l.Close()
	parts = lines(t, o)
	json.Unmarshal(bytes.TrimSuffix(parts[1], []byte("\n")), &rec)
	rec.V = 3
	rewrite(t, o, [][]byte{parts[0], append(encode(rec), '\n')})
	data, _ := os.ReadFile(ledgerPath(o))
	marker, _ := os.ReadFile(filepath.Join(o.Dir, MarkerName))
	before := listing(t, o.Dir)
	if _, err := Open(o); !errors.Is(err, ErrFormatNewer) || err.Error() != "the decision ledger was written by a newer version" {
		t.Fatalf("a newer record: %v", err)
	}
	after, _ := os.ReadFile(ledgerPath(o))
	markerAfter, _ := os.ReadFile(filepath.Join(o.Dir, MarkerName))
	if !bytes.Equal(after, data) || !bytes.Equal(marker, markerAfter) || listing(t, o.Dir) != before {
		t.Error("refusing a newer ledger changed a file")
	}
	rewrite(t, o, [][]byte{parts[0], []byte(`{"v":7,"later":true}`)})
	if _, err := Open(o); !errors.Is(err, ErrFormatNewer) {
		t.Errorf("a newer torn tail: %v", err)
	}
}

func batchOf(n int, from int) []Decision {
	out := []Decision{}
	for i := 0; i < n; i++ {
		d := identitySamples()[SuspendMembership]
		d.Target, d.Revocation = "user-"+strconv.Itoa(from+i), "bulk-revocation-1:"+strconv.Itoa(from+i)
		out = append(out, d)
	}
	return out
}

func TestCommitBatch(t *testing.T) {
	o := options(t)
	s := &fakeSys{}
	l := faulted(t, o, s)
	writes := s.writes
	receipts, err := l.CommitBatch(batchOf(BatchLimit, 0))
	if err != nil || len(receipts) != BatchLimit || s.writes != writes+1 {
		t.Fatalf("%v %d writes %d", err, len(receipts), s.writes-writes)
	}
	st := l.Status()
	if receipts[BatchLimit-1].Watermark != st.Watermark || receipts[0].Sequence != 2 || st.Records != BatchLimit+1 {
		t.Fatalf("receipts %+v status %+v", receipts[BatchLimit-1], st)
	}
	records := l.Decisions()
	for i, r := range receipts {
		if records[i].Sequence != r.Sequence || records[i].Watermark != r.Watermark {
			t.Fatalf("record %d %+v vs %+v", i, records[i], r)
		}
	}
	for name, batch := range map[string][]Decision{"an empty batch": nil, "a batch above the limit": batchOf(BatchLimit+1, 0),
		"a trust record": {{Workspace: "w", Decision: TrustActivation, Trust: &Trust{}}}, "a malformed record": {{Workspace: "w", Decision: SuspendUser}}} {
		if _, err := l.CommitBatch(batch); err.(*CommitError).Reason != ReasonCommitFailed || l.Status().State != StateOK {
			t.Errorf("%s: %v", name, err)
		}
	}
	l.Close()
	again := must(t, o)
	if again.Status().Watermark != st.Watermark {
		t.Error("the batch does not reopen to its watermark")
	}
	again.Close()
	for name, c := range map[string]struct {
		set    func(*faultyFile)
		reason string
	}{
		"write error": {func(f *faultyFile) { f.writeErr = true }, ReasonCommitFailed},
		"short write": {func(f *faultyFile) { f.short = true }, ReasonCommitFailed},
		"seek error":  {func(f *faultyFile) { f.seekErr = true }, ReasonCommitFailed},
		"sync error":  {func(f *faultyFile) { f.syncErr = true }, ReasonCommitUnknown},
	} {
		t.Run(name, func(t *testing.T) {
			s := &fakeSys{}
			l := faulted(t, options(t), s)
			defer l.Close()
			c.set(s.file)
			if _, err := l.CommitBatch(batchOf(3, 0)); err.(*CommitError).Reason != c.reason || l.Status().State != StateStopped {
				t.Fatalf("%v %+v", err, l.Status())
			}
			if _, err := l.CommitBatch(batchOf(1, 0)); err.(*CommitError).Reason != ReasonStopped {
				t.Errorf("a stopped ledger %v", err)
			}
			if len(l.Decisions()) != 0 || l.Status().Records != 1 {
				t.Error("a failed batch was counted")
			}
		})
	}
	small := options(t)
	small.MaxBytes = 4096
	l = must(t, small)
	if _, err := l.CommitBatch(batchOf(20, 0)); err.(*CommitError).Reason != ReasonFull || l.Status().State != StateFull {
		t.Errorf("a batch past the bound: %v %+v", err, l.Status())
	}
	if _, err := l.CommitBatch(batchOf(1, 0)); err.(*CommitError).Reason != ReasonFull {
		t.Errorf("a full ledger %v", err)
	}
	l.Close()
	big := identitySamples()[SuspendMembership]
	big.Target = strings.Repeat("u", RecordLimit)
	l = must(t, options(t))
	defer l.Close()
	if _, err := l.CommitBatch([]Decision{big}); err.(*CommitError).Reason != ReasonCommitFailed || l.Status().State != StateOK {
		t.Errorf("an oversized record %v", err)
	}
	if _, err := l.Commit(big); err.(*CommitError).Reason != ReasonCommitFailed {
		t.Errorf("an oversized single record %v", err)
	}
	isolated := options(t)
	os.MkdirAll(filepath.Join(isolated.Dir, DirName), 0o700)
	os.WriteFile(filepath.Join(isolated.Dir, MarkerName), []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"2026-11-18T09:00:00Z"}`), 0o600)
	l2 := must(t, isolated)
	defer l2.Close()
	if _, err := l2.CommitBatch(batchOf(1, 0)); err.(*CommitError).Reason != ReasonIsolated {
		t.Errorf("an isolated ledger %v", err)
	}
}

func TestATornBatchTailIsVerifiedAndReplayed(t *testing.T) {
	o := options(t)
	l := must(t, o)
	if _, err := l.CommitBatch(batchOf(4, 0)); err != nil {
		t.Fatal(err)
	}
	l.Close()
	parts := lines(t, o)
	parts[len(parts)-2] = bytes.TrimSuffix(parts[len(parts)-2], []byte("\n"))
	rewrite(t, o, parts)
	again := must(t, o)
	defer again.Close()
	if st := again.Status(); st.State != StateIsolated || len(again.Decisions()) != 4 || st.NotDurable {
		t.Errorf("a decodable tail without its newline %+v %d", st, len(again.Decisions()))
	}
	o = options(t)
	l = must(t, o)
	l.CommitBatch(batchOf(4, 0))
	l.Close()
	parts = lines(t, o)
	last := parts[len(parts)-2]
	parts[len(parts)-2] = last[:len(last)/2]
	rewrite(t, o, parts)
	torn := must(t, o)
	defer torn.Close()
	if st := torn.Status(); st.State != StateRecovered || len(torn.Decisions()) != 3 {
		t.Errorf("a torn last line %+v %d", st, len(torn.Decisions()))
	}
}

func TestPropertyBatchesReopenToTheSameWatermark(t *testing.T) {
	r := rand.New(rand.NewPCG(299, 1))
	for round := 0; round < 15; round++ {
		o := options(t)
		l := must(t, o)
		total := 1
		for b := r.IntN(5); b >= 0; b-- {
			n := 1 + r.IntN(30)
			if r.IntN(3) == 0 {
				commit(t, l, strconv.Itoa(total))
				total++
			}
			if _, err := l.CommitBatch(batchOf(n, total)); err != nil {
				t.Fatal(err)
			}
			total += n
		}
		before := l.Status()
		l.Close()
		again := must(t, o)
		after := again.Status()
		again.Close()
		if after.Watermark != before.Watermark || after.Records != uint64(total) {
			t.Fatalf("round %d: %+v vs %+v", round, after, before)
		}
	}
}

func TestScanReadsEveryDecodableLine(t *testing.T) {
	o := options(t)
	if got, err := Scan(o.Dir); err != nil || len(got) != 0 {
		t.Fatalf("a missing ledger %v %v", got, err)
	}
	l := must(t, o)
	commit(t, l, "a")
	l.Commit(identitySamples()[ClockBaseChoice])
	l.Close()
	parts := lines(t, o)
	parts = append(parts[:len(parts)-1], []byte("{not json}\n"), []byte(`{"at":"soon"}`+"\n"), []byte(strings.Repeat("x", LineLimit+10)+"\n"),
		[]byte(`{"at":"2027-01-01T00:00:00Z","kind":"decision","decision":"revoke_grant"}`))
	rewrite(t, o, parts)
	got, err := Scan(o.Dir)
	if err != nil || len(got) != 4 || got[0].Line != 1 || got[2].Clock == nil || got[2].Clock.Choice != ChoiceLedger || got[2].Clock.Offset != 48*time.Hour ||
		got[3].Line != 7 || !got[3].At.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) || got[1].Decision != RevokeGrant {
		t.Fatalf("scan %+v %v", got, err)
	}
	if _, err := Scan(ledgerPath(o)); err == nil {
		t.Error("a ledger below a file")
	}
	folder := options(t)
	os.MkdirAll(ledgerPath(folder), 0o700)
	if _, err := Scan(folder.Dir); err == nil {
		t.Error("a ledger that is a directory")
	}
}

func TestStopKeepsIsolation(t *testing.T) {
	l := must(t, options(t))
	defer l.Close()
	l.Stop()
	if _, err := l.Commit(sample("1")); err.(*CommitError).Reason != ReasonStopped || l.Status().State != StateStopped {
		t.Errorf("stopped %v", err)
	}
	isolated := options(t)
	os.MkdirAll(filepath.Join(isolated.Dir, DirName), 0o700)
	os.WriteFile(filepath.Join(isolated.Dir, MarkerName), []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"2026-11-18T09:00:00Z"}`), 0o600)
	i := must(t, isolated)
	defer i.Close()
	i.Stop()
	if i.Status().State != StateIsolated {
		t.Error("stop changed an isolated ledger")
	}
}

func TestAnyClockFieldOnAnotherKindIsRefused(t *testing.T) {
	for name, set := range map[string]func(*line){
		"ledger time": func(l *line) { l.LedgerTime = stamp(start) }, "wall time": func(l *line) { l.WallTime = stamp(start) },
		"choice": func(l *line) { l.Choice = ChoiceWall }, "offset": func(l *line) { l.OffsetNanos = 1 },
	} {
		for _, kind := range []string{SuspendUser, RevokeGrant} {
			d := identitySamples()[SuspendUser]
			if kind == RevokeGrant {
				d = sample("1")
			}
			rec := lineFor(d)
			rec.Seq = 5
			set(&rec)
			if rec.shaped() {
				t.Errorf("%s with only the %s is shaped", kind, name)
			}
		}
	}
}

func TestAHeaderMayHoldEveryTarget(t *testing.T) {
	d := identitySamples()[BulkRevocationJob]
	d.Held = d.Targets
	if rec := lineFor(d); !rec.shaped() {
		t.Error("a header whose targets were all held is refused")
	}
}

func TestVersionedKindsNeedAVersion(t *testing.T) {
	for _, kind := range []string{SuspendMembership, SuspendUser, RolePeriodEnded, RolePeriodSet, IncreaseIntent} {
		d := identitySamples()[kind]
		d.TargetVersion = 0
		if rec := lineFor(d); rec.shaped() {
			t.Errorf("%s without a version is shaped", kind)
		}
	}
}
