package publication

import (
	"sync"
	"testing"
)

func TestKnownAnswers(t *testing.T) {
	c := []Row{row("a", "g", "KRW", "1.5"), row("b", "h", "USD", "-2")}
	if got := chunkDigest(0, c); got != "f2cfba722d680c02eddfead9925e9f72f53f051d2bd0c35111d662bac2d6dc9a" {
		t.Errorf("chunk %s", got)
	}
	if got := digestOf([][]Row{c, {row("c", "g", "KRW", "3")}}); got != "aecc2e1236578e394cb03d1536da20257ce7c3ce316e191c6c1b0574667a7db3" {
		t.Errorf("version %s", got)
	}
}

func TestConcurrency(t *testing.T) {
	f := setup(t)
	var wg sync.WaitGroup
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := "d" + pad(index)
			for n := 0; n < 10; n++ {
				id := "v" + pad(n)
				parts := split(rows(5, "g"), 2)
				if _, err := f.r.Begin(Spec{Tenant: "t", Dataset: name, Version: id, Rows: 5, Digest: digestOf(parts)}); err != nil {
					t.Error(err)
					return
				}
				for i, p := range parts {
					f.r.AddChunk(Chunk{Tenant: "t", Dataset: name, Version: id, Index: i, Rows: p})
				}
				f.r.Seal("t", name, id)
				f.r.Publish("t", name, id)
				f.r.Rows("t", name, id, "", 3)
				f.r.Totals("t", name, id, nil, 3)
			}
		}(index)
	}
	wg.Wait()
	if f.r.held != 8*3*5 || f.r.staged != 0 {
		t.Errorf("held %d staged %d", f.r.held, f.r.staged)
	}
}
