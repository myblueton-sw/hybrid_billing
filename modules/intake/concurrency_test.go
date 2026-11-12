package intake

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentStagingCountsEachChunkOnce(t *testing.T) {
	h := newHarness(t, nil)
	var chunks [][]Record
	for number := 0; number < 20; number++ {
		var chunk []Record
		for index := 0; index < 25; index++ {
			chunk = append(chunk, rec(fmt.Sprintf("k%02d-%02d", number, index), "1.25", "USD"))
		}
		chunks = append(chunks, chunk)
	}
	s := original(t, "a1", "d1", 1, chunks...)
	h.register(s)
	var wait sync.WaitGroup
	var failures atomic.Int64
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for step := 0; step < len(chunks); step++ {
				number := (step*7+worker*3)%len(chunks) + 1
				if state, err := h.r.Stage("ws1", "a1", number, chunks[number-1]); err != nil || (state != Staging && state != Sealed) {
					failures.Add(1)
				}
				h.r.Staged("ws1", "a1")
				h.r.Seal("ws1", "a1")
			}
		}(worker)
	}
	wait.Wait()
	if failures.Load() != 0 {
		t.Fatal(failures.Load())
	}
	view := h.view("a1")
	if view.State != Sealed || view.Rows != 500 || view.Sums["USD"] != "625" || h.r.workspaces["ws1"].records != 500 || view.Sequence != 1 {
		t.Fatalf("%+v %d", view, h.r.workspaces["ws1"].records)
	}
}

func TestConcurrentPublicationWritesOneManifest(t *testing.T) {
	h := newHarness(t, nil)
	h.sealRows("a1", "d1", 1, []Record{rec("k1", "1", "USD")})
	var wait sync.WaitGroup
	var written, answered atomic.Int64
	for worker := 0; worker < 16; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			out, err := h.r.Publish(october, 0)
			if err == nil && out.Number == 1 {
				answered.Add(1)
			}
			if out.Written {
				written.Add(1)
			}
		}()
	}
	wait.Wait()
	if written.Load() != 1 || answered.Load() != 16 || h.status(october).Latest != 1 {
		t.Fatal(written.Load(), answered.Load())
	}
}

func TestSealingAndRetiringRacePublication(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Manifests = 1000 })
	var wait sync.WaitGroup
	var unexpected atomic.Int64
	for worker := 0; worker < 6; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			document := fmt.Sprintf("d%d", worker%3)
			for revision := 1; revision <= 30; revision++ {
				id := fmt.Sprintf("w%d-r%d", worker, revision)
				rows := []Record{rec(fmt.Sprintf("%s-k1", document), fmt.Sprint(revision), "USD"), rec(fmt.Sprintf("%s-k%d", document, revision%4+2), "1", "USD")}
				s := original(t, id, document, uint64(revision), rows)
				if _, _, err := h.r.Register(s); err != nil {
					unexpected.Add(1)
				}
				if _, err := h.r.Stage("ws1", id, 1, rows); err != nil {
					unexpected.Add(1)
				}
				if _, _, err := h.r.Seal("ws1", id); err != nil {
					unexpected.Add(1)
				}
				status, err := h.r.Status(october)
				if err != nil {
					unexpected.Add(1)
				}
				if _, err := h.r.Publish(october, status.Latest); err != nil && err != ErrConflict && err != ErrRejected {
					unexpected.Add(1)
				}
				if revision%3 == 0 {
					if err := h.r.Retire("ws1", id, Reissued); err != nil {
						unexpected.Add(1)
					}
				}
			}
		}(worker)
	}
	wait.Wait()
	if unexpected.Load() != 0 {
		t.Fatal(unexpected.Load())
	}
	for worker := 0; worker < 6; worker++ {
		for revision := 1; revision <= 30; revision++ {
			h.r.Retire("ws1", fmt.Sprintf("w%d-r%d", worker, revision), Reissued)
		}
	}
	for worker := 0; worker < 3; worker++ {
		document := fmt.Sprintf("d%d", worker)
		h.sealRows("final-"+document, document, 31, []Record{rec(document+"-k1", "5", "USD")})
	}
	status := h.status(october)
	out := h.publish(october, status.Latest)
	if got := brief(h.all(october, out.Number)); got != "event//d0-k1=5USD@final-d0 event//d1-k1=5USD@final-d1 event//d2-k1=5USD@final-d2" {
		t.Fatal(got)
	}
	for number := uint64(1); number <= out.Number; number++ {
		view := h.manifest(october, number)
		if view.Number != number || view.Records != len(h.all(october, number)) {
			t.Fatalf("%+v", view)
		}
	}
	if h.r.workspaces["ws1"].records != 3 {
		t.Fatal(h.r.workspaces["ws1"].records)
	}
}
