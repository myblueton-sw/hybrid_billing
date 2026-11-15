package migration

import (
	"sync"
	"testing"
)

func TestFinishedBatchesDropItems(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "short", Count: 2, Digest: sum("x")}})
	f.must(f.r.Add("src-1", "short", docs(doc("a", 1))[0], imp))
	f.must(second(f.r.Seal("src-1", "short", imp)))
	if b := f.r.sources["src-1"].batches["short"]; b.items != nil || b.codes != nil {
		t.Error("mismatch kept items")
	}
	d := doc("b", 2)
	f.ev.missing[d.Content] = true
	id, _ := f.batch("src-1", docs(d), "t1")
	if b := f.r.sources["src-1"].batches[id]; b.items != nil || b.codes != nil || b.state != BatchMissing {
		t.Error("evidence_missing kept items")
	}
}

func TestSiblingAcceptsAndRejectRecord(t *testing.T) {
	f := standard(t)
	id, v := f.batch("src-1", docs(doc("a", 1)), "t1")
	sibling, err := f.r.Propose(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}}, cat)
	f.must(err)
	applied := f.approve(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}})
	expect(t, "sibling retired", second(f.r.Approve(sibling.Number, bob)), ErrNotFound)
	if got, err := f.r.Proposal(applied.Number); err != nil || got.State != ProposalApplied || got.Approver != "bob" {
		t.Errorf("applied accept record %+v %v", got, err)
	}
	if again, err := f.r.Approve(applied.Number, bob); err != nil || again.State != ProposalApplied {
		t.Errorf("repeat approval %+v %v", again, err)
	}
	id2, _ := f.batch("src-1", docs(doc("b", 2)), "t1")
	f.must(f.r.Reject("src-1", id2, "Wrong export", ann))
	if b, _ := f.r.Batch("t1", "src-1", id2); b.By != "ann" || b.Reason != "Wrong export" {
		t.Errorf("reject record %+v", b)
	}
}

func TestSealImporterRotated(t *testing.T) {
	f := standard(t)
	d := doc("a", 1)
	f.approve(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "b1", Count: 1, Digest: digestOf(f, "src-1", "b1", "t1", docs(d))}})
	f.must(f.r.Add("src-1", "b1", docs(d)[0], imp))
	var once sync.Once
	f.ev.during = func() {
		once.Do(func() { f.approve(Change{Kind: ChangeImporter, Source: Source{ID: "src-1", Importer: "imp-new"}}) })
	}
	reason(t, "rotated", second(f.r.Seal("src-1", "b1", imp)), ReasonStale)
}

func TestAtomicCapsLeaveFences(t *testing.T) {
	f := standard(t)
	f.accept("src-1", "t1", docs(doc("a", 1)))
	d := doc("b", 5)
	d.Delivered = true
	id, v := f.batch("src-1", docs(d), "t1")
	f.r.history = maxHistoryTotal
	expect(t, "cap", f.try(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}}), ErrLimit)
	if hw, _ := f.r.HighWater("t1", "ws1", "inv"); hw != 1 {
		t.Errorf("high water moved %d", hw)
	}
	if taken, _ := f.r.PrintedTaken("t1", "ws1", "INV-5"); taken {
		t.Error("printed fence applied")
	}
	if fenced, _ := f.r.DeliveryFenced("t1", "ws1", "src-1", "b"); fenced {
		t.Error("delivery fence applied")
	}
}

func TestIssuancePrefixCollision(t *testing.T) {
	f := standard(t)
	legacy := doc("legacy-42", 42)
	legacy.Display = "INV-0042"
	f.accept("src-1", "t1", docs(legacy))
	if taken, _ := f.r.PrintedTaken("t1", "ws1", "INV-0042"); !taken {
		t.Error("issuance-style printed number not fenced")
	}
	if taken, _ := f.r.PrintedTaken("t1", "ws1", "INV-0043"); taken {
		t.Error("unrelated number fenced")
	}
}
