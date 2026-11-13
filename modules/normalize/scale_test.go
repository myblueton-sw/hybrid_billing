package normalize

import (
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestManyRowsWithErrorsAndDuplicatesAddUp(t *testing.T) {
	const chunkCount, perChunk = 100, 1000
	random := rand.New(rand.NewSource(188))
	h := newHarness(t, func(l *Limits) { l.Page = 5000; l.Findings = 500 })
	h.declare(fullEntries())
	chunks := make([][]Row, chunkCount)
	sums := map[string]int64{}
	wantNormal, wantErrors, wantApart := 0, 0, 0
	for number := 0; number < chunkCount; number++ {
		for index := 0; index < perChunk; index++ {
			sequence := number*perChunk + index
			key := fmt.Sprintf("row-%06d", sequence)
			currency := []string{"USD", "EUR", "KRW"}[random.Intn(3)]
			value := int64(random.Intn(2000000000)) - 500000000
			text := modelText(value * 100)
			switch {
			case sequence%1000 == 7:
				chunks[number] = append(chunks[number], good(key, "not-a-number", currency))
				wantErrors++
			case sequence%2500 == 11:
				chunks[number] = append(chunks[number], good(fmt.Sprintf("dup-%d", sequence/5000), text, currency))
				wantApart++
			default:
				chunks[number] = append(chunks[number], good(key, text, currency))
				sums[currency] += value * 100
				wantNormal++
			}
		}
	}
	expected := map[string]string{}
	for currency, sum := range sums {
		expected[currency] = modelText(sum)
	}
	s := spec("big", chunkCount, chunkCount*perChunk)
	s.Expected = map[string]map[string]string{"billed": expected}
	h.open(s)
	for _, index := range random.Perm(chunkCount) {
		h.check("big", index+1, chunks[index]...)
		if index%10 == 0 {
			h.check("big", index+1, chunks[index]...)
		}
	}
	view := h.close("big")
	if view.Checked != 100000 || view.Normal != wantNormal || view.Errors != wantErrors || view.Quarantined != wantApart || wantErrors != 100 || wantApart != 40 {
		t.Fatalf("%s, generator %d+%d+%d", counts(view), wantNormal, wantErrors, wantApart)
	}
	if !reflect.DeepEqual(view.Sums["billed"], expected) || view.FindingsTotal != 100 {
		t.Fatalf("%v %d", view.Sums["billed"], view.FindingsTotal)
	}
	if verdicts(view) != "nonconformant[rows_in_error rows_quarantined] not_ready[rows_in_error rows_quarantined]" {
		t.Fatal(verdicts(view))
	}
	rows := h.rows("big")
	if len(rows) != wantNormal || rows[0].Chunk != 1 || rows[0].Position != 1 || rows[len(rows)-1].Chunk != chunkCount {
		t.Fatal(len(rows))
	}
	for index := 1; index < len(rows); index++ {
		if rows[index-1].Chunk > rows[index].Chunk || rows[index-1].Chunk == rows[index].Chunk && rows[index-1].Position >= rows[index].Position {
			t.Fatalf("rows out of order at %d", index)
		}
	}
	findings, err := h.r.Findings("ws1", "big", 0, 5000)
	if err != nil || len(findings) != 100 || findings[0].Chunk != 1 || findings[0].Position != 8 || findings[99].Chunk != 100 {
		t.Fatal(len(findings), err)
	}
	apart, err := h.r.Quarantined("ws1", "big", 0, 5000)
	if err != nil || len(apart) != 40 {
		t.Fatal(len(apart), err)
	}
}

func TestConcurrentChecksCountEachChunkOnce(t *testing.T) {
	h := newHarness(t, nil)
	h.declare(fullEntries())
	var chunks [][]Row
	for number := 0; number < 20; number++ {
		var chunk []Row
		for index := 0; index < 25; index++ {
			chunk = append(chunk, good(fmt.Sprintf("k%02d-%02d", number, index), "1.25", "USD"))
		}
		chunks = append(chunks, chunk)
	}
	s := spec("r1", 20, 500)
	s.Expected = map[string]map[string]string{"billed": {"USD": "625"}}
	h.open(s)
	var wait sync.WaitGroup
	var failures atomic.Int64
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for step := 0; step < len(chunks); step++ {
				number := (step*7+worker*3)%len(chunks) + 1
				if state, err := h.r.Check("ws1", "r1", number, chunks[number-1]); err != nil || (state != Open && state != Closed) {
					failures.Add(1)
				}
				h.r.Run("ws1", "r1")
				h.r.Close("ws1", "r1")
				h.r.Rows("ws1", "r1", 0, 10)
			}
		}(worker)
	}
	wait.Wait()
	view := h.view("r1")
	if failures.Load() != 0 || view.State != Closed || counts(view) != "500=500+0+0" || verdicts(view) != "conformant[] ready[]" || h.r.workspaces["ws1"].rows != 500 {
		t.Fatalf("%d %s %s", failures.Load(), counts(view), verdicts(view))
	}
}

func TestConcurrentDeclarersAppendOneRevision(t *testing.T) {
	h := newHarness(t, nil)
	h.declare(testEntries())
	var wait sync.WaitGroup
	var appended, conflicts atomic.Int64
	for worker := 0; worker < 12; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			entries := fullEntries()
			entries[3].Transform, entries[3].Power = Scale, worker%3+1
			number, err := h.r.DeclareMapping(mapping, 1, entries)
			switch {
			case err == nil && number == 2:
				appended.Add(1)
			case err == ErrConflict:
				conflicts.Add(1)
			}
		}(worker)
	}
	wait.Wait()
	latest, _ := h.r.Latest(mapping)
	if latest.Number != 2 || appended.Load() != 4 || conflicts.Load() != 8 {
		t.Fatalf("%d appended or repeated, %d conflicts, latest %d", appended.Load(), conflicts.Load(), latest.Number)
	}
}

func TestRunsOfTwoWorkspacesInParallel(t *testing.T) {
	h := newHarness(t, nil)
	h.declare(fullEntries())
	var wait sync.WaitGroup
	var failures atomic.Int64
	for worker := 0; worker < 6; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for round := 0; round < 20; round++ {
				s := spec(fmt.Sprintf("r%d-%d", worker, round), 2, 4)
				s.WorkspaceID = fmt.Sprintf("ws%d", worker%2)
				s.Expected = map[string]map[string]string{"billed": {"USD": "10"}}
				if _, err := h.r.Open(s); err != nil {
					failures.Add(1)
				}
				for chunk := 1; chunk <= 2; chunk++ {
					rows := []Row{good(fmt.Sprintf("a%d", chunk), "2", "USD"), good(fmt.Sprintf("b%d", chunk), "3", "USD")}
					if _, err := h.r.Check(s.WorkspaceID, s.ID, chunk, rows); err != nil {
						failures.Add(1)
					}
				}
				if _, _, err := h.r.Close(s.WorkspaceID, s.ID); err != nil {
					failures.Add(1)
				}
				view, err := h.r.Run(s.WorkspaceID, s.ID)
				if err != nil || verdicts(view) != "conformant[] ready[]" {
					failures.Add(1)
				}
				if round%2 == 0 {
					if err := h.r.Discard(s.WorkspaceID, s.ID, DiscardSuperseded); err != nil {
						failures.Add(1)
					}
				}
			}
		}(worker)
	}
	wait.Wait()
	if failures.Load() != 0 || h.r.workspaces["ws0"].rows != 120 || h.r.workspaces["ws1"].rows != 120 {
		t.Fatal(failures.Load(), h.r.workspaces["ws0"].rows, h.r.workspaces["ws1"].rows)
	}
}
