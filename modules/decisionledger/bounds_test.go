package decisionledger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fileSize(t *testing.T, o Options) int64 {
	t.Helper()
	info, err := os.Stat(filepath.Join(o.Dir, DirName, FileName))
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

func TestABatchFitsTheBoundExactly(t *testing.T) {
	probe := options(t)
	l := must(t, probe)
	empty := fileSize(t, probe)
	if _, err := l.CommitBatch(batchOf(10, 0)); err != nil {
		t.Fatal(err)
	}
	l.Close()
	need := fileSize(t, probe)
	for _, c := range []struct {
		bound int64
		ok    bool
	}{{need, true}, {need - 1, false}} {
		o := options(t)
		o.MaxBytes = c.bound
		l := must(t, o)
		if got := fileSize(t, o); got != empty {
			t.Fatalf("genesis %d, probe %d", got, empty)
		}
		_, err := l.CommitBatch(batchOf(10, 0))
		if (err == nil) != c.ok {
			t.Errorf("bound %d of %d: %v", c.bound, need, err)
		}
		l.Close()
	}
}

func TestARecordOfExactlyTheLimitIsAccepted(t *testing.T) {
	sized := func(n int) Decision {
		d := identitySamples()[SuspendMembership]
		d.Target = strings.Repeat("u", n)
		return d
	}
	probe := options(t)
	l := must(t, probe)
	before := fileSize(t, probe)
	if _, err := l.CommitBatch([]Decision{sized(1)}); err != nil {
		t.Fatal(err)
	}
	l.Close()
	base := int(fileSize(t, probe)-before) - 1 - 1
	for _, c := range []struct {
		n  int
		ok bool
	}{{RecordLimit - base, true}, {RecordLimit - base + 1, false}} {
		l := must(t, options(t))
		_, err := l.CommitBatch([]Decision{sized(c.n)})
		if (err == nil) != c.ok {
			t.Errorf("target of %d bytes (record %d): %v", c.n, base+c.n, err)
		}
		l.Close()
	}
}

func TestIncreaseIntentShapes(t *testing.T) {
	l := must(t, options(t))
	defer l.Close()
	intent := func(kind, role, until string) Decision {
		return Decision{Workspace: "sample-workspace", Decision: IncreaseIntent, Target: "user-c", TargetVersion: 1, RequestedBy: "user-b", Increase: kind, Role: role, Until: until}
	}
	for name, c := range map[string]struct {
		d  Decision
		ok bool
	}{
		"reinstate user":               {intent(IncreaseReinstateUser, "", ""), true},
		"reinstate membership":         {intent(IncreaseReinstateMembership, "", ""), true},
		"an unknown increase":          {intent("grant_everything", "", ""), false},
		"reinstate user with a role":   {intent(IncreaseReinstateUser, "member", ""), false},
		"reinstate user with an end":   {intent(IncreaseReinstateUser, "", NoEnd), false},
		"reinstate member with an end": {intent(IncreaseReinstateMembership, "", NoEnd), false},
	} {
		if _, err := l.CommitBatch([]Decision{c.d}); (err == nil) != c.ok {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOnlyClockBaseDecisionsCarryAClockInTheScan(t *testing.T) {
	o := options(t)
	l := must(t, o)
	if _, err := l.Commit(identitySamples()[SuspendUser]); err != nil {
		t.Fatal(err)
	}
	l.Close()
	stamps, err := Scan(o.Dir)
	if err != nil || len(stamps) != 2 {
		t.Fatalf("%v %d", err, len(stamps))
	}
	for _, s := range stamps {
		if s.Clock != nil {
			t.Errorf("line %d %q carries a clock", s.Line, s.Decision)
		}
	}
}
