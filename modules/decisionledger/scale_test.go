package decisionledger

import (
	"strconv"
	"testing"
	"time"
)

func TestScaleOneHundredThousandItemRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	o := options(t)
	o.MaxBytes = 256 << 20
	s := &fakeSys{}
	l := faulted(t, o, s)
	began, syncs := time.Now(), 0
	if _, err := l.Commit(identitySamples()[BulkRevocationJob]); err != nil {
		t.Fatal(err)
	}
	syncs++
	for start := 0; start < 100000; start += BatchLimit {
		if _, err := l.CommitBatch(batchOf(min(BatchLimit, 100000-start), start)); err != nil {
			t.Fatal(err)
		}
		syncs++
	}
	t.Logf("IAM-07 ledger: 100000 item records and a header in %v with %d syncs and %d writes", time.Since(began), syncs, s.writes)
	l.Close()
	began = time.Now()
	again := must(t, o)
	defer again.Close()
	if st := again.Status(); st.Records != 100002 || st.State != StateOK {
		t.Fatalf("reopened %+v", st)
	}
	t.Logf("IAM-07 ledger: reopen and verify of %s records took %v", strconv.Itoa(len(again.Decisions())), time.Since(began))
}
